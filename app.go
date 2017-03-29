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
	Name     string
	ReadUrl  string
	S3Url    string
	Username string
	Password string
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
var etcdPeers = flag.String("etcd-peers", "http://localhost:2379", "Comma-separated list of addresses of etcd endpoints to connect to")
var etcdReadEnvKey = flag.String("etcd-read-env-key", "/ft/config/monitoring/read-urls", "etcd key that lists the read environment URLs")
var etcdS3EnvKey = flag.String("etcd-s3-env-key", "/ft/config/monitoring/s3-image-bucket-urls", "etcd key that lists the S3 image bucket URLs")
var etcdCredKey = flag.String("etcd-cred-key", "/ft/_credentials/publish-read/read-credentials", "etcd key that lists the read environment credentials")
var etcdValidatorCredKey = flag.String("etcd-validator-cred-key", "/ft/_credentials/publish-read/validator-credentials", "etcd key that specifies the validator credentials")

var appConfig *AppConfig
var environments = make(map[string]Environment)
var subscribedFeeds = make(map[string][]feeds.Feed)
var metricSink = make(chan PublishMetric)
var metricContainer publishHistory
var validatorCredentials string

func main() {
	initLogs(os.Stdout, os.Stdout, os.Stderr)
	flag.Parse()

	var err error
	appConfig, err = ParseConfig(*configFileName)
	if err != nil {
		errorLogger.Printf("Cannot load configuration: [%v]", err)
		return
	}

	go DiscoverEnvironmentsAndValidators(etcdPeers, etcdReadEnvKey, etcdCredKey, etcdS3EnvKey, etcdValidatorCredKey, environments)

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
	router.HandleFunc("/__gtg", healthcheck.gtg)
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

	publishedContent, err := content.UnmarshalContent(msg)
	if err != nil {
		warnLogger.Printf("Cannot unmarshal message [%v], error: [%v]", tid, err.Error())
		return
	}

	uuid := publishedContent.GetUUID()
	validationEndpointKey := getValidationEndpointKey(publishedContent, tid, uuid)
	var validationEndpoint string
	var found bool
	var username string
	var password string

	if validationEndpoint, found = appConfig.ValidationEndpoints[validationEndpointKey]; found {
		username, password = getValidationCredentials(validationEndpoint)
	}

	valRes := publishedContent.Validate(validationEndpoint, tid, username, password)
	if !valRes.IsValid {
		infoLogger.Printf("Message [%v] with UUID [%v] is INVALID, skipping...", tid, uuid)
		return
	}

	infoLogger.Printf("Message [%v] with UUID [%v] is VALID.", tid, uuid)

	publishDateString := msg.Headers["Message-Timestamp"]
	publishDate, err := time.Parse(dateLayout, publishDateString)
	if err != nil {
		errorLogger.Printf("Cannot parse publish date [%v] from message [%v], error: [%v]",
			publishDateString, tid, err.Error())
		return
	}

	if isMessagePastPublishSLA(publishDate, appConfig.Threshold) {
		infoLogger.Printf("Message [%v] with UUID [%v] is past publish SLA, skipping.", tid, uuid)
		return
	}

	scheduleChecks(publishedContent, publishDate, tid, valRes.IsMarkedDeleted, &metricContainer, environments)

	// for images we need to check their corresponding image sets
	// the image sets don't have messages of their own so we need to create one
	if publishedContent.GetType() == "Image" {
		eomFile, ok := publishedContent.(content.EomFile)
		if !ok {
			errorLogger.Printf("Cannot assert that message [%v] with UUID [%v] and type 'Image' is an EomFile.", tid, uuid)
			return
		}
		imageSetEomFile := spawnImageSet(eomFile)
		if imageSetEomFile.UUID != "" {
			scheduleChecks(imageSetEomFile, publishDate, tid, false, &metricContainer, environments)
		}
	}

	// if this is normal content, schedule checks for internal components also
	if publishedContent.GetType() == "EOM::CompoundStory" {
		eomFileForInternalComponentsCheck, ok := publishedContent.(content.EomFile)
		if !ok {
			errorLogger.Printf("Cannot assert that message [%v] with UUID [%v] and type 'EOM::CompoundStory' is an EomFile.", tid, uuid)
			return
		}
		eomFileForInternalComponentsCheck.Type = "InternalComponents"

		var internalComponentsValidationEndpoint = appConfig.ValidationEndpoints["InternalComponents"]
		var usr, pass = getValidationCredentials(internalComponentsValidationEndpoint)

		icValRes := publishedContent.Validate(internalComponentsValidationEndpoint, tid, usr, pass)
		if !icValRes.IsValid {
			infoLogger.Printf("Message [%v] with UUID [%v] has INVALID internal components, skipping internal components schedule check.", tid, uuid)
			return
		}

		scheduleChecks(eomFileForInternalComponentsCheck, publishDate, tid, icValRes.IsMarkedDeleted, &metricContainer, environments)
	}
}

func getCredentials(url string) (string, string) {
	for _, env := range environments {
		if strings.HasPrefix(url, env.ReadUrl) {
			return env.Username, env.Password
		}
	}

	return "", ""
}

func getValidationCredentials(url string) (string, string) {
	if strings.Contains(validatorCredentials, ":") {
		unpw := strings.SplitN(validatorCredentials, ":", 2)
		return unpw[0], unpw[1]
	}

	return "", ""
}

func spawnImageSet(imageEomFile content.EomFile) content.EomFile {
	imageSetEomFile := imageEomFile
	imageSetEomFile.Type = "ImageSet"

	imageUUID, err := content.NewUUIDFromString(imageEomFile.UUID)
	if err != nil {
		warnLogger.Printf("Cannot generate UUID from image UUID string [%v]: [%v], skipping image set check.",
			imageEomFile.UUID, err.Error())
		return content.EomFile{}
	}

	imageSetUUID, err := content.GenerateImageSetUUID(*imageUUID)
	if err != nil {
		warnLogger.Printf("Cannot generate image set UUID: [%v], skipping image set check",
			err.Error())
		return content.EomFile{}
	}

	imageSetEomFile.UUID = imageSetUUID.String()
	return imageSetEomFile
}

func isMessagePastPublishSLA(date time.Time, threshold int) bool {
	passedSLA := date.Add(time.Duration(threshold) * time.Second)
	return time.Now().After(passedSLA)
}

func isIgnorableMessage(tid string) bool {
	return strings.HasPrefix(tid, "SYNTHETIC")
}

func getValidationEndpointKey(publishedContent content.Content, tid string, uuid string) string {
	validationEndpointKey := publishedContent.GetType()
	if strings.Contains(publishedContent.GetType(), "EOM::CompoundStory") {
		_, ok := publishedContent.(content.EomFile)
		if !ok {
			errorLogger.Printf("Cannot assert that message [%v] with UUID [%v] and type 'EOM::CompoundStory' is an EomFile.", tid, uuid)
			return ""
		}

	}
	return validationEndpointKey
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
