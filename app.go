package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/pprof"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"sync"
	"syscall"
	"time"

	logger "github.com/Financial-Times/go-logger/v2"
	consumer "github.com/Financial-Times/message-queue-gonsumer"
	"github.com/Financial-Times/publish-availability-monitor/v2/checks"
	"github.com/Financial-Times/publish-availability-monitor/v2/feeds"
	"github.com/Financial-Times/publish-availability-monitor/v2/logformat"
	status "github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// Interval is a simple representation of an interval of time, with a lower and
// upper boundary
type Interval struct {
	lowerBound int
	upperBound int
}

// PublishMetric holds the information about the metric we are measuring.
type PublishMetric struct {
	UUID            string
	publishOK       bool      //did it meet the SLA?
	publishDate     time.Time //the time WE get the message
	platform        string
	publishInterval Interval //the interval it was actually published in, ex. (10,20)
	config          MetricConfig
	endpoint        url.URL
	tid             string
	isMarkedDeleted bool
}

// MetricConfig is the configuration of a PublishMetric
type MetricConfig struct {
	Granularity  int      `json:"granularity"` //how we split up the threshold, ex. 120/12
	Endpoint     string   `json:"endpoint"`
	ContentTypes []string `json:"contentTypes"` //list of valid eom types for this metric
	Alias        string   `json:"alias"`
	Health       string   `json:"health,omitempty"`
	ApiKey       string   `json:"apiKey,omitempty"`
}

// SplunkConfig holds the SplunkFeeder-specific configuration
type SplunkConfig struct {
	LogPrefix string `json:"logPrefix"`
}

// AppConfig holds the application's configuration
type AppConfig struct {
	Threshold           int                  `json:"threshold"` //pub SLA in seconds, ex. 120
	QueueConf           consumer.QueueConfig `json:"queueConfig"`
	MetricConf          []MetricConfig       `json:"metricConfig"`
	SplunkConf          SplunkConfig         `json:"splunk-config"`
	HealthConf          HealthConfig         `json:"healthConfig"`
	ValidationEndpoints map[string]string    `json:"validationEndpoints"` //contentType to validation endpoint mapping, ex. { "EOM::Story": "http://methode-article-transformer/content-transform" }
	UUIDResolverUrl     string               `json:"uuidResolverUrl"`
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

type publishHistory struct {
	sync.RWMutex
	publishMetrics []PublishMetric
}

const dateLayout = time.RFC3339Nano

var configFileName = flag.String("config", "", "Path to configuration file")

var etcdPeers = flag.String("etcd-peers", "http://localhost:2379", "Comma-separated list of addresses of etcd endpoints to connect to")
var etcdReadEnvKey = flag.String("etcd-read-env-key", "/ft/config/monitoring/read-urls", "etcd key that lists the read environment URLs")
var etcdS3EnvKey = flag.String("etcd-s3-env-key", "/ft/config/monitoring/s3-image-bucket-urls", "etcd key that lists the S3 image bucket URLs")
var etcdCredKey = flag.String("etcd-cred-key", "/ft/_credentials/publish-read/read-credentials", "etcd key that lists the read environment credentials")
var etcdValidatorCredKey = flag.String("etcd-validator-cred-key", "/ft/_credentials/publish-read/validator-credentials", "etcd key that specifies the validator credentials")

var envsFileName = flag.String("envs-file-name", "/etc/pam/envs/read-environments.json", "Path to json file that contains environments configuration")
var envCredentialsFileName = flag.String("envs-credentials-file-name", "/etc/pam/credentials/read-environments-credentials.json", "Path to json file that contains environments credentials")
var validatorCredentialsFileName = flag.String("validator-credentials-file-name", "/etc/pam/credentials/validator-credentials.json", "Path to json file that contains validation endpoints configuration")
var configRefreshPeriod = flag.Int("config-refresh-period", 1, "Refresh period for configuration in minutes. By default it is 1 minute.")

var appConfig *AppConfig
var environments = newThreadSafeEnvironments()
var subscribedFeeds = make(map[string][]feeds.Feed)
var metricSink = make(chan PublishMetric)
var metricContainer publishHistory
var validatorCredentials string
var configFilesHashValues = make(map[string]string)
var carouselTransactionIDRegExp = regexp.MustCompile(`^.+_carousel_[\d]{10}.*$`)

func init() {
	f := logformat.NewSLF4JFormatter(`.*/github\.com/Financial-Times/.*`)
	log.SetFormatter(f)
}

func main() {
	flag.Parse()

	brandMappings := readBrandMappings()

	var err error
	appConfig, err = ParseConfig(*configFileName)
	if err != nil {
		log.WithError(err).Error("Cannot load configuration")
		return
	}

	wg := new(sync.WaitGroup)
	wg.Add(1)
	if *etcdPeers == "NOT_AVAILABLE" {
		log.Info("Sourcing dynamic configs from file")
		go watchConfigFiles(wg, *envsFileName, *envCredentialsFileName, *validatorCredentialsFileName, *configRefreshPeriod)
	} else {
		log.Info("Sourcing dynamic configs from ETCD")
		go DiscoverEnvironmentsAndValidators(wg, etcdPeers, etcdReadEnvKey, etcdCredKey, etcdS3EnvKey, etcdValidatorCredKey)
	}
	wg.Wait()

	metricContainer = publishHistory{sync.RWMutex{}, make([]PublishMetric, 0)}

	go startHttpListener()

	startAggregator()
	readMessages(brandMappings)
}

func startHttpListener() {
	router := mux.NewRouter()
	setupHealthchecks(router)
	router.HandleFunc("/__history", loadHistory)

	router.HandleFunc(status.PingPath, status.PingHandler)
	router.HandleFunc(status.PingPathDW, status.PingHandler)

	router.HandleFunc(status.BuildInfoPath, status.BuildInfoHandler)
	router.HandleFunc(status.BuildInfoPathDW, status.BuildInfoHandler)

	attachProfiler(router)

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

func attachProfiler(router *mux.Router) {
	router.HandleFunc("/debug/pprof/", pprof.Index)
	router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
}

func readMessages(brandMappings map[string]string) {

	for !environments.areReady() {
		log.Info("Environments not set, retry in 3s...")
		time.Sleep(3 * time.Second)
	}

	var typeRes typeResolver
	for _, envName := range environments.names() {
		env := environments.environment(envName)
		docStoreCaller := checks.NewHttpCaller(10)
		docStoreClient := checks.NewHttpDocStoreClient(env.ReadUrl+appConfig.UUIDResolverUrl, docStoreCaller, env.Username, env.Password)
		uuidResolver := checks.NewHttpUUIDResolver(docStoreClient, brandMappings)
		typeRes = NewMethodeTypeResolver(uuidResolver)
		break
	}

	l := logger.NewUnstructuredLogger()
	h := NewKafkaMessageHandler(typeRes)
	c := consumer.NewConsumer(appConfig.QueueConf, h.HandleMessage, &http.Client{}, l)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		c.Start()
		wg.Done()
	}()

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	c.Stop()
	wg.Wait()
}

func startAggregator() {
	var destinations []MetricDestination

	splunkFeeder := NewSplunkFeeder(appConfig.SplunkConf.LogPrefix)
	destinations = append(destinations, splunkFeeder)
	aggregator := NewAggregator(metricSink, destinations)
	go aggregator.Run()
}

func loadHistory(w http.ResponseWriter, r *http.Request) {
	metricContainer.RLock()
	for i := len(metricContainer.publishMetrics) - 1; i >= 0; i-- {
		fmt.Fprintf(w, "%d. %v\n\n", len(metricContainer.publishMetrics)-i, metricContainer.publishMetrics[i])
	}
	metricContainer.RUnlock()
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

func (pm PublishMetric) String() string {
	return fmt.Sprintf("Tid: %s, UUID: %s, Platform: %s, Endpoint: %s, PublishDate: %s, Duration: %d, Succeeded: %t.",
		pm.tid,
		pm.UUID,
		pm.platform,
		pm.config.Alias,
		pm.publishDate.String(),
		pm.publishInterval.upperBound,
		pm.publishOK,
	)

}
