package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/kr/pretty"
)

type Interval struct {
	lowerBound int
	upperBound int
}

type PublishMetric struct {
	UUID            string
	publishOK       bool      //did it meet the SLA?
	publishDate     time.Time //the time WE get the message
	platform        string
	publishInterval Interval //the interval it was actually published in, ex. (10,20)
	config          MetricConfig
	endpoint        url.URL
}

type MetricConfig struct {
	Granularity int    `json:"granularity"` //how we split up the threshold, ex. 120/12
	Endpoint    string `json:"endpoint"`
}

type AppConfig struct {
	Threshold  int                  `json:"threshold"` //pub SLA in seconds, ex. 120
	QueueConf  consumer.QueueConfig `json:"queueConfig"`
	MetricConf []MetricConfig       `json:"metricConfig"`
	Platform   string               `json:"platform"`
	//TODO feeder configs
}

type PublishMessageListener struct{}

type EomFile struct {
	UUID             string `json:"uuid"`
	Type             string `json:"type"`
	Value            string `json:"value"`
	Attributes       string `json:"attributes"`
	SystemAttributes string `json:"systemAttributes"`
}

const dateLayout = "2006-01-02T15:04:05.000Z"
const logPattern = log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile | log.LUTC

var info *log.Logger
var warn *log.Logger
var configFileName = flag.String("config", "", "Path to configuration file")
var appConfig AppConfig
var err error

func main() {
	initLogs(os.Stdout, os.Stdout, os.Stderr)
	flag.Parse()

	appConfig, err := ParseConfig(*configFileName)
	if err != nil {
		log.Printf("Cannot load configuration: [%v]", err)
		return
	}

	iterator := consumer.NewIterator(appConfig.QueueConf)

	for {
		msgs, _ := iterator.NextMessages()
		for _, m := range msgs {
			PublishMessageListener{}.OnMessage(m)
		}
	}

	/*
		scheduler := scheduler.NewScheduler()
		aggregator := aggregator.NewAggregator()
		validator := validator.NewValidator()
	*/
	//maybe separate the distributor so it just waits for metrics from the aggregator like a servlet?
}

func (listener PublishMessageListener) OnMessage(msg consumer.Message) error {
	tid := msg.Headers["X-Request-Id"]
	info.Printf("Received message with TID [%v]", tid)

	if !isMessageValid(msg) {
		return nil
	}

	var eomFile EomFile
	err := json.Unmarshal([]byte(msg.Body), &eomFile)
	if err != nil {
		log.Printf("Cannot unmarshal message [%v], error: [%v]", tid, err.Error())
		return err
	}

	if !isEomfileValid(eomFile) {
		info.Printf("Message [%v] is INVALID, skipping...", tid)
		return nil
	}

	info.Printf("Message [%v] is VALID, scheduling checks...", tid)

	publishDateString := msg.Headers["Message-Timestamp"]
	publishDate, err := time.Parse(dateLayout, publishDateString)
	if err != nil {
		log.Printf("Cannot parse publish date [%v] from message [%v], error: [%v]",
			publishDateString, tid, err.Error())
		return nil
	}

	var publishMetrics []PublishMetric

	for _, metricConf := range appConfig.MetricConf {

		endpointUrl, err := url.Parse(metricConf.Endpoint)
		if err != nil {
			log.Printf("Cannot parse url [%v], error: [%v]", metricConf.Endpoint, err.Error())
			continue
		}

		var publishMetric = PublishMetric{
			eomFile.UUID,
			false,
			publishDate,
			appConfig.Platform,
			Interval{},
			metricConf,
			*endpointUrl,
		}
		publishMetrics = append(publishMetrics, publishMetric)
	}

	info.Println("Metrics to schedule: %# v", pretty.Formatter(publishMetrics))
	//read publish timestamp (this is the moment we measure the publish from)
	//Message-Timestamp: 2015-10-21T10:27:00.597Z
	//TODO check if exists and not null
	//publishDateString := msg.Headers["Message-Timestamp"]
	//publishDate, err := time.Parse(dateLayout, publishDateString)

	//scheduler.scheduleChecks(message, publishMetric)
	//connect the scheduler with the aggregator with channels or something
	//so when each scheduler is finished, the aggregator reads the results
	return nil
}

func initLogs(infoHandle io.Writer, warnHandle io.Writer, panicHandle io.Writer) {
	//to be used for INFO-level logging: info.Println("foo is now bar")
	info = log.New(infoHandle, "INFO  - ", logPattern)
	//to be used for WARN-level logging: warn.Println("foo is now bar")
	warn = log.New(warnHandle, "WARN  - ", logPattern)

	log.SetFlags(logPattern)
	log.SetPrefix("ERROR - ")
	log.SetOutput(panicHandle)
}
