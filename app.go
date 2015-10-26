package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/Financial-Times/message-queue-gonsumer/consumer"
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
	//TODO feeder configs
}

type PublishMessageListener struct{}

const dateLayout = "2006-01-02T15:04:05.000Z"

func main() {
	//read config (into structs?)
	configFileName := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	appConfig, err := ParseConfig(*configFileName)
	if err != nil {
		log.Printf("ERROR - %v", err)
		return
	}
	log.Printf("INFO - AppConfig: %#v", *appConfig)
	//TODO handle err
	iterator := consumer.NewIterator(appConfig.QueueConf)

	for {
		msgs, _ := iterator.NextMessages()
		for _, m := range msgs {
			PublishMessageListener{}.OnMessage(m)
		}
	}

	if err != nil {
		fmt.Println(err.Error)
	}
	/*
		scheduler := scheduler.NewScheduler()
		aggregator := aggregator.NewAggregator()
		validator := validator.NewValidator()
	*/
	//maybe separate the distributor so it just waits for metrics from the aggregator like a servlet?
}

func (listener PublishMessageListener) OnMessage(msg consumer.Message) error {
	fmt.Printf("message headers: %v\n", msg.Headers)
	fmt.Printf("message body: %v\n", msg.Body)

	//if message is not valid, skip
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
