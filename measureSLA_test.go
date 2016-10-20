package main

import (
	"testing"
	"github.com/stretchr/testify/mock"
	"sync"
	"github.com/Financial-Times/publish-availability-monitor/content"
	"time"
)

var matchAny = mock.MatchedBy(func(x interface{}) bool { return true })

type MockGTGRunner struct {
	mock.Mock
}

type MockScheduler struct {
	mock.Mock
}

type MockSplunkSLAFeeder struct {
	mock.Mock
}

func (m MockGTGRunner) RunGtgChecks(all map[string]Environment, env Environment) bool {
	args := m.Called(all, env)
	return args.Bool(0)
}

func (m MockScheduler) ScheduleChecks(event ContentEvent, all map[string]Environment) CheckContext {
	args := m.Called(event, all)
	return args.Get(0).(CheckContext)
}

func (m MockSplunkSLAFeeder) SendPublishSLA(sla SLAMetric) {
	m.Called(sla)
}

func setup(sink chan PublishMetric) {
	splunkFeeder := NewSplunkFeeder("[splunkMetrics] ")
	aggregator := NewAggregator(sink, []MetricDestination{splunkFeeder})
	go aggregator.Run()
}

func publishDate() time.Time {
	date, _ := time.Parse(time.RFC3339, "2016-10-20T00:00:00Z")
	return date
}

func contentEvent() ContentEvent {
	return ContentEvent{
		contentToCheck: content.EomFile{UUID: "unique-id"},
		tid: "tid",
		publishDate: publishDate(),
	}
}

func TestMeasureSLA(t *testing.T){
	sink := make(chan PublishMetric)
	defer close(sink)
	setup(sink)

	gtgRunner := new(MockGTGRunner)
	scheduler := new(MockScheduler)

	slaFeeder := new (MockSplunkSLAFeeder)
	slaFeeder.On("SendPublishSLA", SLAMetric{UUID: "unique-id", tid: "tid", publishDate: publishDate(), metPublishSLA: false, okEnvironments: []string{}})

	environments := make(map[string]Environment)
	testContent := contentEvent()

	gtgRunner.On("RunGtgChecks", matchAny, matchAny).Return(true)
	scheduler.On("ScheduleChecks", testContent, environments).Return(CheckContext{make(chan PublishMetric), &sync.WaitGroup{}})

	collator := DefaultCollator{environments, scheduler, gtgRunner, sink, slaFeeder}
	collator.MeasureSLA(testContent)

	scheduler.AssertExpectations(t) // only testing here that the scheduler is called
}

func TestCollateSuccessfulChecks(t *testing.T){
	sink := make(chan PublishMetric)
	defer close(sink)
	setup(sink)

	channel := make(chan PublishMetric, 2)
	channel <- PublishMetric{UUID: "unique-id", platform: "test1", publishOK: true}
	channel <- PublishMetric{UUID: "unique-id", platform: "test2", publishOK: true}

	environments := map[string]Environment {
		"test1": {Name: "test1"},
		"test2": {Name: "test2"},
	}

	wg := &sync.WaitGroup{}
	context := CheckContext{channel, wg}

	gtgRunner := new(MockGTGRunner)

	slaFeeder := new (MockSplunkSLAFeeder)
	slaFeeder.On("SendPublishSLA", SLAMetric{UUID: "unique-id", tid: "tid", publishDate: publishDate(), metPublishSLA: true, okEnvironments: []string{"test1", "test2"}})

	collator := DefaultCollator{environments, new(MockScheduler), gtgRunner, sink, slaFeeder}
	collator.CollateResults(contentEvent(), context)

	gtgRunner.AssertNotCalled(t, "RunGtgChecks", environments, Environment{Name: "test1"})
	gtgRunner.AssertNotCalled(t, "RunGtgChecks", environments, Environment{Name: "test2"})
	slaFeeder.AssertExpectations(t)
}

func TestCollateOneUnsuccessfulCheckAndIgnored(t *testing.T){
	sink := make(chan PublishMetric)
	defer close(sink)
	setup(sink)

	channel := make(chan PublishMetric, 2)
	channel <- PublishMetric{UUID: "unique-id", platform: "test1", publishOK: false}
	channel <- PublishMetric{UUID: "unique-id", platform: "test2", publishOK: true}

	environments := map[string]Environment {
		"test1": {Name: "test1"},
		"test2": {Name: "test2"},
	}

	wg := &sync.WaitGroup{}
	context := CheckContext{channel, wg}

	gtgRunner := new(MockGTGRunner)
	gtgRunner.On("RunGtgChecks", environments, Environment{Name: "test1"}).Return(true)

	slaFeeder := new (MockSplunkSLAFeeder)
	slaFeeder.On("SendPublishSLA", SLAMetric{UUID: "unique-id", tid: "tid", publishDate: publishDate(), metPublishSLA: true, okEnvironments: []string{"test2"}})

	collator := DefaultCollator{environments, new(MockScheduler), gtgRunner, sink, slaFeeder}
	collator.CollateResults(contentEvent(), context)

	gtgRunner.AssertNotCalled(t, "RunGtgChecks", environments, Environment{Name: "test2"})
	gtgRunner.AssertExpectations(t)
	slaFeeder.AssertExpectations(t)
}

func TestCollateBothUnsuccessfulChecksAndIgnored(t *testing.T){
	sink := make(chan PublishMetric)
	defer close(sink)
	setup(sink)

	channel := make(chan PublishMetric, 2)
	channel <- PublishMetric{UUID: "unique-id", platform: "test1", publishOK: false}
	channel <- PublishMetric{UUID: "unique-id", platform: "test2", publishOK: false}

	environments := map[string]Environment {
		"test1": {Name: "test1"},
		"test2": {Name: "test2"},
	}

	wg := &sync.WaitGroup{}
	context := CheckContext{channel, wg}

	gtgRunner := new(MockGTGRunner)
	gtgRunner.On("RunGtgChecks", environments, Environment{Name: "test1"}).Return(true)
	gtgRunner.On("RunGtgChecks", environments, Environment{Name: "test2"}).Return(true)

	slaFeeder := new (MockSplunkSLAFeeder)
	slaFeeder.On("SendPublishSLA", SLAMetric{UUID: "unique-id", tid: "tid", publishDate: publishDate(), metPublishSLA: false, okEnvironments: []string{}}) // SLA should be false!

	collator := DefaultCollator{environments, new(MockScheduler), gtgRunner, sink, slaFeeder}
	collator.CollateResults(contentEvent(), context)

	gtgRunner.AssertExpectations(t)
	slaFeeder.AssertExpectations(t)
}

func TestCollateBothUnsuccessfulChecksAndNotIgnored(t *testing.T){
	sink := make(chan PublishMetric)
	defer close(sink)
	setup(sink)

	channel := make(chan PublishMetric, 2)
	channel <- PublishMetric{UUID: "unique-id", platform: "test1", publishOK: false}
	channel <- PublishMetric{UUID: "unique-id", platform: "test2", publishOK: false}

	environments := map[string]Environment {
		"test1": {Name: "test1"},
		"test2": {Name: "test2"},
	}

	wg := &sync.WaitGroup{}
	context := CheckContext{channel, wg}

	gtgRunner := new(MockGTGRunner)
	gtgRunner.On("RunGtgChecks", environments, Environment{Name: "test1"}).Return(false)
	gtgRunner.On("RunGtgChecks", environments, Environment{Name: "test2"}).Return(false)

	slaFeeder := new (MockSplunkSLAFeeder)
	slaFeeder.On("SendPublishSLA", SLAMetric{UUID: "unique-id", tid: "tid", publishDate: publishDate(), metPublishSLA: false, okEnvironments: []string{}})

	collator := DefaultCollator{environments, new(MockScheduler), gtgRunner, sink, slaFeeder}
	collator.CollateResults(contentEvent(), context)

	gtgRunner.AssertExpectations(t)
	slaFeeder.AssertExpectations(t)
}

func TestCollateBothUnsuccessfulChecksAndOneNotIgnored(t *testing.T){
	sink := make(chan PublishMetric)
	defer close(sink)
	setup(sink)

	channel := make(chan PublishMetric, 2)
	channel <- PublishMetric{UUID: "unique-id", platform: "test1", publishOK: false}
	channel <- PublishMetric{UUID: "unique-id", platform: "test2", publishOK: false}

	environments := map[string]Environment {
		"test1": {Name: "test1"},
		"test2": {Name: "test2"},
	}

	wg := &sync.WaitGroup{}
	context := CheckContext{channel, wg}

	gtgRunner := new(MockGTGRunner)
	gtgRunner.On("RunGtgChecks", environments, Environment{Name: "test1"}).Return(false)
	gtgRunner.On("RunGtgChecks", environments, Environment{Name: "test2"}).Return(true)

	slaFeeder := new (MockSplunkSLAFeeder)
	slaFeeder.On("SendPublishSLA", SLAMetric{UUID: "unique-id", tid: "tid", publishDate: publishDate(), metPublishSLA: false, okEnvironments: []string{}})

	collator := DefaultCollator{environments, new(MockScheduler), gtgRunner, sink, slaFeeder}
	collator.CollateResults(contentEvent(), context)

	gtgRunner.AssertExpectations(t)
	slaFeeder.AssertExpectations(t)
}

func TestCollateOneUnsuccessfulCheckAndNotIgnored(t *testing.T){
	sink := make(chan PublishMetric)
	defer close(sink)
	setup(sink)

	channel := make(chan PublishMetric, 2)
	channel <- PublishMetric{UUID: "unique-id", platform: "test1", publishOK: false}
	channel <- PublishMetric{UUID: "unique-id", platform: "test2", publishOK: true}

	environments := map[string]Environment {
		"test1": {Name: "test1"},
		"test2": {Name: "test2"},
	}

	wg := &sync.WaitGroup{}
	context := CheckContext{channel, wg}

	gtgRunner := new(MockGTGRunner)
	gtgRunner.On("RunGtgChecks", environments, Environment{Name: "test1"}).Return(false)

	slaFeeder := new (MockSplunkSLAFeeder)
	slaFeeder.On("SendPublishSLA", SLAMetric{UUID: "unique-id", tid: "tid", publishDate: publishDate(), metPublishSLA: false, okEnvironments: []string{"test2"}}) // SLA should be false!

	collator := DefaultCollator{environments, new(MockScheduler), gtgRunner, sink, slaFeeder}
	collator.CollateResults(contentEvent(), context)

	gtgRunner.AssertNotCalled(t, "RunGtgChecks", environments, Environment{Name: "test2"})
	gtgRunner.AssertExpectations(t)
	slaFeeder.AssertExpectations(t)
}