package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"sync"
	"syscall"
	"time"

	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/Financial-Times/publish-availability-monitor/config"
	"github.com/Financial-Times/publish-availability-monitor/content"
	"github.com/Financial-Times/publish-availability-monitor/envs"
	"github.com/Financial-Times/publish-availability-monitor/feeds"
	"github.com/Financial-Times/publish-availability-monitor/httpcaller"
	"github.com/Financial-Times/publish-availability-monitor/logformat"
	"github.com/Financial-Times/publish-availability-monitor/metrics"
	"github.com/Financial-Times/publish-availability-monitor/sender"
	status "github.com/Financial-Times/service-status-go/httphandlers"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

var configFileName = flag.String("config", "", "Path to configuration file")
var envsFileName = flag.String("envs-file-name", "/etc/pam/envs/read-environments.json", "Path to json file that contains environments configuration")
var envCredentialsFileName = flag.String("envs-credentials-file-name", "/etc/pam/credentials/read-environments-credentials.json", "Path to json file that contains environments credentials")
var validatorCredentialsFileName = flag.String("validator-credentials-file-name", "/etc/pam/credentials/validator-credentials.json", "Path to json file that contains validation endpoints configuration")
var configRefreshPeriod = flag.Int("config-refresh-period", 1, "Refresh period for configuration in minutes. By default it is 1 minute.")

var carouselTransactionIDRegExp = regexp.MustCompile(`^.+_carousel_[\d]{10}.*$`)

func init() {
	log.SetFormatter(&logformat.SLF4JFormatter{})
}

func main() {
	flag.Parse()

	var err error
	appConfig, err := config.NewAppConfig(*configFileName)
	if err != nil {
		log.WithError(err).Error("Cannot load configuration")
		return
	}

	var environments = envs.NewEnvironments()
	var subscribedFeeds = make(map[string][]feeds.Feed)
	var metricSink = make(chan metrics.PublishMetric)
	var configFilesHashValues = make(map[string]string)

	wg := new(sync.WaitGroup)
	wg.Add(1)

	log.Info("Sourcing dynamic configs from file")

	go envs.WatchConfigFiles(wg, *envsFileName, *envCredentialsFileName, *validatorCredentialsFileName, *configRefreshPeriod, configFilesHashValues, environments, subscribedFeeds, appConfig)

	wg.Wait()

	metricContainer := &metrics.PublishMetricsHistory{
		RWMutex:        sync.RWMutex{},
		PublishMetrics: make([]metrics.PublishMetric, 0),
	}

	go startHTTPServer(appConfig, environments, subscribedFeeds, metricContainer)

	metricDestinations := []sender.MetricDestination{
		sender.NewSplunkFeeder(appConfig.SplunkConf.LogPrefix),
	}

	aggregator := sender.NewAggregator(metricSink, metricDestinations)
	go aggregator.Run()

	readKafkaMessages(appConfig, environments, subscribedFeeds, metricSink, metricContainer)
}

func startHTTPServer(appConfig *config.AppConfig, environments *envs.Environments, subscribedFeeds map[string][]feeds.Feed, metricContainer *metrics.PublishMetricsHistory) {
	router := mux.NewRouter()

	hc := newHealthcheck(appConfig, metricContainer, environments, subscribedFeeds)
	router.HandleFunc("/__health", hc.checkHealth())
	router.HandleFunc(status.GTGPath, status.NewGoodToGoHandler(hc.GTG))

	router.HandleFunc("/__history", loadHistory(metricContainer))

	router.HandleFunc(status.PingPath, status.PingHandler)
	router.HandleFunc(status.PingPathDW, status.PingHandler)

	router.HandleFunc(status.BuildInfoPath, status.BuildInfoHandler)
	router.HandleFunc(status.BuildInfoPathDW, status.BuildInfoHandler)

	http.Handle("/", router)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Panicf("Couldn't set up HTTP listener: %+v\n", err)
	}
}

func loadHistory(metricContainer *metrics.PublishMetricsHistory) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		metricContainer.RLock()
		for i := len(metricContainer.PublishMetrics) - 1; i >= 0; i-- {
			fmt.Fprintf(w, "%d. %v\n\n", len(metricContainer.PublishMetrics)-i, metricContainer.PublishMetrics[i])
		}
		metricContainer.RUnlock()
	}
}

func readKafkaMessages(appConfig *config.AppConfig, environments *envs.Environments, subscribedFeeds map[string][]feeds.Feed, metricSink chan metrics.PublishMetric, metricContainer *metrics.PublishMetricsHistory) {
	for !environments.AreReady() {
		log.Info("Environments not set, retry in 3s...")
		time.Sleep(3 * time.Second)
	}

	var typeRes content.TypeResolver
	for _, envName := range environments.Names() {
		env := environments.Environment(envName)
		docStoreCaller := httpcaller.NewCaller(10)
		docStoreClient := content.NewHTTPDocStoreClient(env.ReadURL+appConfig.UUIDResolverURL, docStoreCaller, env.Username, env.Password)
		uuidResolver := content.NewHTTPUUIDResolver(docStoreClient, readBrandMappings())
		typeRes = content.NewMethodeTypeResolver(uuidResolver)
		break
	}

	h := NewKafkaMessageHandler(typeRes, appConfig, environments, subscribedFeeds, metricSink, metricContainer)
	c := consumer.NewConsumer(appConfig.QueueConf, h.HandleMessage, &http.Client{})

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		c.Start()
		wg.Done()
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	c.Stop()
	wg.Wait()
}

func readBrandMappings() map[string]string {
	brandMappingsFile, err := ioutil.ReadFile("brandMappings.json")
	if err != nil {
		log.Errorf("Couldn't read brand mapping configuration: %v\n", err)
		os.Exit(1)
	}
	var brandMappings map[string]string
	err = json.Unmarshal(brandMappingsFile, &brandMappings)
	if err != nil {
		log.Errorf("Couldn't unmarshal brand mapping configuration: %v\n", err)
		os.Exit(1)
	}
	return brandMappings
}
