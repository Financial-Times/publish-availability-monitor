package main

import (
	"flag"
	"fmt"
	"github.com/Financial-Times/go-message-queue-consumer"
	"log"
	"net/url"
	"time"
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
	myConsumer := consumer.NewConsumer(appConfig.QueueConf)
	err = myConsumer.Consume(PublishMessageListener{}, 8)
	if err != nil {
		fmt.Println(err.Error)
	}
	/*
		consumer := consumer.NewConsumer(QueueConfig.address etc.)
		scheduler := scheduler.NewScheduler()
		aggregator := aggregator.NewAggregator()
		validator := validator.NewValidator()

		consumer.Consume(/*this?/)
	*/
	//maybe separate the distributor so it just waits for metrics from the aggregator like a servlet?
	//call consume
}

func (listener PublishMessageListener) OnMessage(msg consumer.Message) error {
	fmt.Printf("message: %v\n", msg)
	//if message is not valid, skip
	//generate timestamp (this is the moment we measure the publish from)
	//scheduler.scheduleChecks(message, publishMetric)
	//connect the scheduler with the aggregator with channels or something
	//so when each scheduler is finished, the aggregator reads the results
	return nil
}
