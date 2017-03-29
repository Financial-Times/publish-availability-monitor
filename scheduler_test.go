package main

import (
	"github.com/Financial-Times/publish-availability-monitor/content"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestValidType(testing *testing.T) {
	var tests = []struct {
		validTypes []string
		eomType    string
		expected   bool
	}{
		{
			[]string{"Image", "EOM:WebContainer"},
			"EOM:CompoundStory",
			false,
		},
		{
			[]string{"Image", "EOM:WebContainer"},
			"EOM:WebContainer",
			true,
		},
	}

	for _, t := range tests {
		actual := validType(t.validTypes, t.eomType)
		if actual != t.expected {
			testing.Errorf("Test Case: %v\nActual: %v", t, actual)
		}
	}
}

var validImageEomFile = content.EomFile{
	UUID:             "e28b12f7-9796-3331-b030-05082f0b8157",
	Type:             "Image",
	Value:            "/9j/4QAYRXhpZgAASUkqAAgAAAAAAAAAAAAAAP/sABFEdWNr",
	Attributes:       "attributes",
	SystemAttributes: "system attributes",
}

var mockArticleEomFile = content.EomFile{
	UUID:             "a24da1d4-1524-2322-c231-25032d0f8334",
	Type:             "EOM:CompoundStory",
	Value:            "/9j/4QAYRXhpZgAASUkqAAgAAAAAAAAAAAAAAP/sABFEdWNr",
	Attributes:       "attributes",
	SystemAttributes: "system attributes",
}

func TestScheduleChecksForS3AreCorrect(testing *testing.T) {
	//redefine appConfig to have only S3
	appConfig = &AppConfig{
		MetricConf: []MetricConfig{
			{
				Endpoint:    "/whatever/",
				Granularity: 1,
				Alias:       "S3",
				ContentTypes: []string{
					"Image",
				},
			},
		},
		Threshold: 1,
	}

	var mockEnvironments = make(map[string]Environment)
	readURL := "http://env1.example.org"
	s3URL := "http://s1.example.org"
	mockEnvironments["env1"] = Environment{"env1", readURL, s3URL, "user1", "pass1"}

	capturingMetrics := runScheduleChecks(testing, validImageEomFile, mockEnvironments)
	defer capturingMetrics.RUnlock()

	require.NotNil(testing, capturingMetrics)
	require.Equal(testing, 1, len(capturingMetrics.publishMetrics))
	require.Equal(testing, s3URL+"/whatever/", capturingMetrics.publishMetrics[0].endpoint.String())
}

func TestScheduleChecksForContentAreCorrect(testing *testing.T) {
	//redefine appConfig to have only Content
	appConfig = &AppConfig{
		MetricConf: []MetricConfig{
			{
				Endpoint:    "/whatever/",
				Granularity: 1,
				Alias:       "content",
				ContentTypes: []string{
					"Image",
				},
			},
		},
		Threshold: 1,
	}

	var mockEnvironments = make(map[string]Environment)
	readURL := "http://env1.example.org"
	s3URL := "http://s1.example.org"
	mockEnvironments["env1"] = Environment{"env1", readURL, s3URL, "user1", "pass1"}

	capturingMetrics := runScheduleChecks(testing, validImageEomFile, mockEnvironments)
	defer capturingMetrics.RUnlock()

	require.NotNil(testing, capturingMetrics)
	require.Equal(testing, 1, len(capturingMetrics.publishMetrics))
	require.Equal(testing, readURL+"/whatever/", capturingMetrics.publishMetrics[0].endpoint.String())
}

func TestScheduleChecksForContentWithInternalComponentsAreCorrect(testing *testing.T) {
	appConfig = &AppConfig{
		MetricConf: []MetricConfig{
			{
				Endpoint:    "/internalcomponents/",
				Granularity: 1,
				Alias:       "internal-components",
				ContentTypes: []string{
					"InternalComponents",
				},
			},
		},
		Threshold: 1,
	}

	var mockEnvironments = make(map[string]Environment)
	readURL := "http://env1.example.org"
	s3URL := "http://s1.example.org"
	mockEnvironments["env1"] = Environment{"env1", readURL, s3URL, "user1", "pass1"}

	mockArticleEomFile.Type = "InternalComponents"

	capturingMetrics := runScheduleChecks(testing, mockArticleEomFile, mockEnvironments)
	defer capturingMetrics.RUnlock()

	require.NotNil(testing, capturingMetrics)
	require.Equal(testing, 1, len(capturingMetrics.publishMetrics))
	require.Equal(testing, readURL+"/internalcomponents/", capturingMetrics.publishMetrics[0].endpoint.String())
}

func runScheduleChecks(testing *testing.T, content content.Content, mockEnvironments map[string]Environment) *publishHistory {
	capturingMetrics := &publishHistory{sync.RWMutex{}, make([]PublishMetric, 0)}
	tid := "tid_1234"
	publishDate, err := time.Parse(dateLayout, "2016-01-08T14:22:06.271Z")
	if err != nil {
		testing.Error("Failure in setting up test data")
		return nil
	}

	//redefine map to avoid actual checks
	endpointSpecificChecks = map[string]EndpointSpecificCheck{}
	//redefine metricSink to avoid hang
	metricSink = make(chan PublishMetric, 2)

	scheduleChecks(content, publishDate, tid, true, capturingMetrics, mockEnvironments)
	for {
		capturingMetrics.RLock()
		if len(capturingMetrics.publishMetrics) == len(mockEnvironments) {
			return capturingMetrics // with a read lock
		}

		capturingMetrics.RUnlock()
		time.Sleep(1 * time.Second)
	}
}
