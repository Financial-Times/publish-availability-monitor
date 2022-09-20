package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"sync"
	"syscall"
	"time"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/kafka-client-go/v3"
	"github.com/Financial-Times/publish-availability-monitor/config"
	"github.com/Financial-Times/publish-availability-monitor/envs"
	"github.com/Financial-Times/publish-availability-monitor/feeds"
	"github.com/Financial-Times/publish-availability-monitor/metrics"
	status "github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/gorilla/mux"
)

var configFileName = flag.String("config", "", "Path to configuration file")
var envsFileName = flag.String("envs-file-name", "/etc/pam/envs/read-environments.json", "Path to json file that contains environments configuration")
var envCredentialsFileName = flag.String("envs-credentials-file-name", "/etc/pam/credentials/read-environments-credentials.json", "Path to json file that contains environments credentials")
var validatorCredentialsFileName = flag.String("validator-credentials-file-name", "/etc/pam/credentials/validator-credentials.json", "Path to json file that contains validation endpoints configuration")
var configRefreshPeriod = flag.Int("config-refresh-period", 1, "Refresh period for configuration in minutes. By default it is 1 minute.")

var carouselTransactionIDRegExp = regexp.MustCompile(`^.+_carousel_[\d]{10}.*$`)

func main() {
	flag.Parse()

	log := logger.NewUPPLogger("publish-availability-monitor", "INFO")

	var err error
	appConfig, err := config.NewAppConfig(*configFileName, log)
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

	metricContainer := metrics.NewHistory(make([]metrics.PublishMetric, 0))

	var e2eTestUUIDs []string
	for _, c := range appConfig.Capabilities {
		for _, id := range c.TestIDs {
			if !sliceContains(e2eTestUUIDs, id) {
				e2eTestUUIDs = append(e2eTestUUIDs, id)
			}
		}
	}

	h := NewKafkaMessageHandler(appConfig, environments, subscribedFeeds, metricSink, metricContainer, e2eTestUUIDs, log)
	c := kafka.NewConsumer(kafka.ConsumerConfig{
		BrokersConnectionString: appConfig.QueueConf.BrokersConnectionString,
		ConsumerGroup:           appConfig.QueueConf.ConsumerGroup,
		Options:                 kafka.DefaultConsumerOptions(),
	}, []*kafka.Topic{kafka.NewTopic(
		appConfig.QueueConf.Topic,
		kafka.WithLagTolerance(int64(appConfig.QueueConf.KafkaLagTolerance)))},
		log)

	go startHTTPServer(appConfig, environments, subscribedFeeds, metricContainer, c, log)

	publishMetricDestinations := []metrics.Destination{
		metrics.NewSplunkFeeder(appConfig.SplunkConf.LogPrefix),
	}

	capabilityMetricDestinations := []metrics.Destination{
		metrics.NewGraphiteSender(appConfig),
	}

	aggregator := metrics.NewAggregator(metricSink, publishMetricDestinations, capabilityMetricDestinations)
	go aggregator.Run()

	readKafkaMessages(c, h, environments, log)
}

func startHTTPServer(appConfig *config.AppConfig, environments *envs.Environments, subscribedFeeds map[string][]feeds.Feed, metricContainer *metrics.History, c *kafka.Consumer, log *logger.UPPLogger) {
	router := mux.NewRouter()

	hc := newHealthcheck(appConfig, metricContainer, environments, subscribedFeeds, c, log)
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

func loadHistory(metricContainer *metrics.History) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, metricContainer.String())
	}
}

func readKafkaMessages(c *kafka.Consumer, h MessageHandler, environments *envs.Environments, log *logger.UPPLogger) {
	for !environments.AreReady() {
		log.Info("Environments not set, retry in 3s...")
		time.Sleep(3 * time.Second)
	}

	go func() {
		c.Start(h.HandleMessage)
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	c.Close()
}

func sliceContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
