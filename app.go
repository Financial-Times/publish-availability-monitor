package main

import (
	"flag"
	"io"
	"log"
	"net/http"
	"net/http/pprof"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"fmt"

	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/Financial-Times/publish-availability-monitor/content"
	"github.com/Financial-Times/publish-availability-monitor/feeds"
	status "github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/gorilla/mux"
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
const logPattern = log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile | log.LUTC

var infoLogger *log.Logger
var warnLogger *log.Logger
var errorLogger *log.Logger
var configFileName = flag.String("config", "", "Path to configuration file")
var envsFileName = flag.String("envs-file-name", "/etc/pam/envs/read-environments.json", "Path to json file that contains environments configuration")
var envCredentialsFileName = flag.String("envs-credentials-file-name", "/etc/pam/credentials/read-environments-credentials.json", "Path to json file that contains environments credentials")
var validatorCredentialsFileName = flag.String("validator-credentials-file-name", "/etc/pam/credentials/validator-credentials.json", "Path to json file that contains validation endpoints configuration")
var appConfig *AppConfig
var environments = make(map[string]Environment)
var subscribedFeeds = make(map[string][]feeds.Feed)
var metricSink = make(chan PublishMetric)
var metricContainer publishHistory
var validatorCredentials Credentials

var carouselTransactionIDRegExp = regexp.MustCompile(`^(tid_[a-zA-Z0-9]+)_carousel_[\d]{10}.*$`)

func main() {
	initLogs(os.Stdout, os.Stdout, os.Stderr)
	flag.Parse()

	var err error
	appConfig, err = ParseConfig(*configFileName)
	if err != nil {
		errorLogger.Printf("Cannot load configuration: [%v]", err)
		return
	}

	err = updateEnvsAndValidationCredentials(*envsFileName, *envCredentialsFileName, *validatorCredentialsFileName)
	if err != nil {
		errorLogger.Printf("Cannot load envs or validation credentials, error was: [%v]", err)
		return
	}

	go watchConfigFiles(*envsFileName, *envCredentialsFileName, *validatorCredentialsFileName)
	metricContainer = publishHistory{sync.RWMutex{}, make([]PublishMetric, 0)}

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

	attachProfiler(router)

	http.Handle("/", router)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		errorLogger.Panicf("Couldn't set up HTTP listener: %+v\n", err)
	}
}

func setupHealthchecks(router *mux.Router) {
	healthcheck := &Healthcheck{http.Client{}, *appConfig, &metricContainer}
	router.HandleFunc("/__health", healthcheck.checkHealth)
	gtgHandler := status.NewGoodToGoHandler(healthcheck.gtg)
	router.HandleFunc(status.GTGPath, gtgHandler)
}

func attachProfiler(router *mux.Router) {
	router.HandleFunc("/debug/pprof/", pprof.Index)
	router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
}

func readMessages() {
	c := consumer.NewConsumer(appConfig.QueueConf, handleMessage, &http.Client{})

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

func handleMessage(msg consumer.Message) {
	tid := msg.Headers["X-Request-Id"]
	infoLogger.Printf("Received message with TID [%v]", tid)

	if isIgnorableMessage(tid) {
		infoLogger.Printf("Message [%v] is ignorable. Skipping...", tid)
		return
	}

	publishDateString := msg.Headers["Message-Timestamp"]
	publishDate, err := time.Parse(dateLayout, publishDateString)
	if err != nil {
		errorLogger.Printf("Cannot parse publish date [%v] from message [%v], error: [%v]",
			publishDateString, tid, err.Error())
		return
	}

	publishedContent, err := content.UnmarshalContent(msg)
	if err != nil {
		warnLogger.Printf("Cannot unmarshal message [%v], error: [%v]", tid, err.Error())
		return
	}

	var paramsToSchedule []*schedulerParam

	for _, preCheck := range mainPreChecks() {
		ok, scheduleParam := preCheck(publishedContent, tid, publishDate)
		if ok {
			paramsToSchedule = append(paramsToSchedule, scheduleParam)
		} else {
			//if a main check is not ok, additional checks make no sense
			return
		}
	}

	for _, preCheck := range additionalPreChecks() {
		ok, scheduleParam := preCheck(publishedContent, tid, publishDate)
		if ok {
			paramsToSchedule = append(paramsToSchedule, scheduleParam)
		}
	}

	for _, scheduleParam := range paramsToSchedule {
		scheduleChecks(scheduleParam)
	}
}

func isIgnorableMessage(tid string) bool {
	return isSyntheticTransactionID(tid) || isContentCarouselTransactionID(tid)
}

func isSyntheticTransactionID(tid string) bool {
	return strings.HasPrefix(tid, "SYNTHETIC")
}

func isContentCarouselTransactionID(tid string) bool {
	return carouselTransactionIDRegExp.MatchString(tid)
}

func initLogs(infoHandle io.Writer, warnHandle io.Writer, errorHandle io.Writer) {
	//to be used for INFO-level logging: info.Println("foo is now bar")
	infoLogger = log.New(infoHandle, "INFO  - ", logPattern)
	//to be used for WARN-level logging: warn.Println("foo is now bar")
	warnLogger = log.New(warnHandle, "WARN  - ", logPattern)
	//to be used for ERROR-level logging: errorL.Println("foo is now bar")
	errorLogger = log.New(errorHandle, "ERROR - ", logPattern)
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
