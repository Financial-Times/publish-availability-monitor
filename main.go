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
	"github.com/Financial-Times/publish-availability-monitor/checks"
	"github.com/Financial-Times/publish-availability-monitor/feeds"
	"github.com/Financial-Times/publish-availability-monitor/httpcaller"
	"github.com/Financial-Times/publish-availability-monitor/logformat"
	"github.com/Financial-Times/publish-availability-monitor/models"
	"github.com/Financial-Times/publish-availability-monitor/sender"
	"github.com/Financial-Times/publish-availability-monitor/splunk"
	status "github.com/Financial-Times/service-status-go/httphandlers"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

type publishHistory struct {
	sync.RWMutex
	publishMetrics []models.PublishMetric
}

// AppConfig holds the application's configuration
type AppConfig struct {
	Threshold           int                   `json:"threshold"` //pub SLA in seconds, ex. 120
	QueueConf           consumer.QueueConfig  `json:"queueConfig"`
	MetricConf          []models.MetricConfig `json:"metricConfig"`
	SplunkConf          SplunkConfig          `json:"splunk-config"`
	HealthConf          HealthConfig          `json:"healthConfig"`
	ValidationEndpoints map[string]string     `json:"validationEndpoints"` //contentType to validation endpoint mapping, ex. { "EOM::Story": "http://methode-article-transformer/content-transform" }
	UUIDResolverURL     string                `json:"uuidResolverUrl"`
}

// SplunkConfig holds the SplunkFeeder-specific configuration
type SplunkConfig struct {
	LogPrefix string `json:"logPrefix"`
}

// HealthConfig holds the application's healthchecks configuration
type HealthConfig struct {
	FailureThreshold int `json:"failureThreshold"`
}

// Environment defines an environment in which the publish metrics should be checked
type Environment struct {
	Name     string `json:"name"`
	ReadUrl  string `json:"read-url"`
	S3Url    string `json:"s3-url"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Credentials struct {
	EnvName  string `json:"env-name"`
	Username string `json:"username"`
	Password string `json:"password"`
}

var configFileName = flag.String("config", "", "Path to configuration file")
var envsFileName = flag.String("envs-file-name", "/etc/pam/envs/read-environments.json", "Path to json file that contains environments configuration")
var envCredentialsFileName = flag.String("envs-credentials-file-name", "/etc/pam/credentials/read-environments-credentials.json", "Path to json file that contains environments credentials")
var validatorCredentialsFileName = flag.String("validator-credentials-file-name", "/etc/pam/credentials/validator-credentials.json", "Path to json file that contains validation endpoints configuration")
var configRefreshPeriod = flag.Int("config-refresh-period", 1, "Refresh period for configuration in minutes. By default it is 1 minute.")

var appConfig *AppConfig
var environments = newThreadSafeEnvironments()

var subscribedFeeds = make(map[string][]feeds.Feed)
var metricSink = make(chan models.PublishMetric)
var metricContainer publishHistory
var validatorCredentials string
var configFilesHashValues = make(map[string]string)
var carouselTransactionIDRegExp = regexp.MustCompile(`^.+_carousel_[\d]{10}.*$`)

func init() {
	log.SetFormatter(&logformat.SLF4JFormatter{})
}

func main() {
	flag.Parse()

	var err error
	appConfig, err = ParseConfig(*configFileName)
	if err != nil {
		log.WithError(err).Error("Cannot load configuration")
		return
	}

	wg := new(sync.WaitGroup)
	wg.Add(1)

	log.Info("Sourcing dynamic configs from file")
	go watchConfigFiles(wg, *envsFileName, *envCredentialsFileName, *validatorCredentialsFileName, *configRefreshPeriod)

	wg.Wait()

	metricContainer = publishHistory{sync.RWMutex{}, make([]models.PublishMetric, 0)}

	go startHttpListener()

	startAggregator()
	readMessages()
}

func startHttpListener() {
	router := mux.NewRouter()
	setupHealthchecks(router)
	router.HandleFunc("/__history", loadHistory)

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

func setupHealthchecks(router *mux.Router) {
	hc := newHealthcheck(appConfig, &metricContainer)
	router.HandleFunc("/__health", hc.checkHealth())
	router.HandleFunc(status.GTGPath, status.NewGoodToGoHandler(hc.GTG))
}

func loadHistory(w http.ResponseWriter, r *http.Request) {
	metricContainer.RLock()
	for i := len(metricContainer.publishMetrics) - 1; i >= 0; i-- {
		fmt.Fprintf(w, "%d. %v\n\n", len(metricContainer.publishMetrics)-i, metricContainer.publishMetrics[i])
	}
	metricContainer.RUnlock()
}

func startAggregator() {
	var destinations []sender.MetricDestination

	splunkFeeder := splunk.NewSplunkFeeder(appConfig.SplunkConf.LogPrefix)
	destinations = append(destinations, splunkFeeder)
	aggregator := sender.NewAggregator(metricSink, destinations)
	go aggregator.Run()
}

func readMessages() {
	for !environments.areReady() {
		log.Info("Environments not set, retry in 3s...")
		time.Sleep(3 * time.Second)
	}

	var typeRes typeResolver
	for _, envName := range environments.names() {
		env := environments.environment(envName)
		docStoreCaller := httpcaller.NewCaller(10)
		docStoreClient := checks.NewHTTPDocStoreClient(env.ReadUrl+appConfig.UUIDResolverURL, docStoreCaller, env.Username, env.Password)
		uuidResolver := checks.NewHttpUUIDResolver(docStoreClient, readBrandMappings())
		typeRes = NewMethodeTypeResolver(uuidResolver)
		break
	}

	h := NewKafkaMessageHandler(typeRes)
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
