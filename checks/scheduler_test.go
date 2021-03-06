package checks

import (
	"testing"
	"time"

	"github.com/Financial-Times/publish-availability-monitor/config"
	"github.com/Financial-Times/publish-availability-monitor/content"
	"github.com/Financial-Times/publish-availability-monitor/envs"
	"github.com/Financial-Times/publish-availability-monitor/feeds"
	"github.com/Financial-Times/publish-availability-monitor/metrics"
	"github.com/stretchr/testify/require"
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
	appConfig := &config.AppConfig{
		MetricConf: []config.MetricConfig{
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

	var mockEnvironments = envs.NewEnvironments()
	readURL := "http://env1.example.org"
	s3URL := "http://s1.example.org"

	mockEnvironments.SetEnvironment("env1", envs.Environment{
		Name:     "env1",
		ReadURL:  readURL,
		S3Url:    s3URL,
		Username: "user1",
		Password: "pass1",
	})

	capturingMetrics := runScheduleChecks(testing, validImageEomFile, mockEnvironments, appConfig)

	require.NotNil(testing, capturingMetrics)
	require.Equal(testing, 1, capturingMetrics.Len())
	require.Equal(testing, s3URL+"/whatever/", capturingMetrics.First().Endpoint.String())
}

func TestScheduleChecksForContentAreCorrect(testing *testing.T) {
	//redefine appConfig to have only Content
	appConfig := &config.AppConfig{
		MetricConf: []config.MetricConfig{
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

	var mockEnvironments = envs.NewEnvironments()
	readURL := "http://env1.example.org"
	s3URL := "http://s1.example.org"
	mockEnvironments.SetEnvironment("env1", envs.Environment{
		Name:     "env1",
		ReadURL:  readURL,
		S3Url:    s3URL,
		Username: "user1",
		Password: "pass1",
	})

	capturingMetrics := runScheduleChecks(testing, validImageEomFile, mockEnvironments, appConfig)

	require.NotNil(testing, capturingMetrics)
	require.Equal(testing, 1, capturingMetrics.Len())
	require.Equal(testing, readURL+"/whatever/", capturingMetrics.First().Endpoint.String())
}

func TestScheduleChecksForContentWithInternalComponentsAreCorrect(testing *testing.T) {
	appConfig := &config.AppConfig{
		MetricConf: []config.MetricConfig{
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

	var mockEnvironments = envs.NewEnvironments()
	readURL := "http://env1.example.org"
	s3URL := "http://s1.example.org"

	mockEnvironments.SetEnvironment("env1", envs.Environment{
		Name:     "env1",
		ReadURL:  readURL,
		S3Url:    s3URL,
		Username: "user1",
		Password: "pass1",
	})

	mockArticleEomFile.Type = "InternalComponents"

	capturingMetrics := runScheduleChecks(testing, mockArticleEomFile, mockEnvironments, appConfig)

	require.NotNil(testing, capturingMetrics)
	require.Equal(testing, 1, capturingMetrics.Len())
	require.Equal(testing, readURL+"/internalcomponents/", capturingMetrics.First().Endpoint.String())
}

func TestScheduleChecksForDynamicContentWithInternalComponentsAreCorrect(testing *testing.T) {
	appConfig := &config.AppConfig{
		MetricConf: []config.MetricConfig{
			{
				Endpoint:    "/internalcomponents/",
				Granularity: 1,
				Alias:       "internal-components",
				ContentTypes: []string{
					"InternalComponents",
					"EOM::CompoundStory_DynamicContent",
				},
			},
		},
		Threshold: 1,
	}

	var mockEnvironments = envs.NewEnvironments()
	readURL := "http://env1.example.org"
	s3URL := "http://s1.example.org"

	mockEnvironments.SetEnvironment("env1", envs.Environment{
		Name:     "env1",
		ReadURL:  readURL,
		S3Url:    s3URL,
		Username: "user1",
		Password: "pass1",
	})

	mockArticleEomFile.Type = "EOM::CompoundStory_DynamicContent"

	capturingMetrics := runScheduleChecks(testing, mockArticleEomFile, mockEnvironments, appConfig)

	require.NotNil(testing, capturingMetrics)
	require.Equal(testing, 1, capturingMetrics.Len())
	require.Equal(testing, readURL+"/internalcomponents/", capturingMetrics.First().Endpoint.String())
}

func runScheduleChecks(testing *testing.T, content content.Content, mockEnvironments *envs.Environments, appConfig *config.AppConfig) *metrics.History {
	capturingMetrics := metrics.NewHistory(make([]metrics.PublishMetric, 0))

	tid := "tid_1234"
	publishDate, err := time.Parse(DateLayout, "2016-01-08T14:22:06.271Z")
	if err != nil {
		testing.Error("Failure in setting up test data")
		return nil
	}

	//redefine map to avoid actual checks
	endpointSpecificChecks := map[string]EndpointSpecificCheck{}
	//redefine metricSink to avoid hang
	metricSink := make(chan metrics.PublishMetric, 2)
	subscribedFeeds := map[string][]feeds.Feed{}

	ScheduleChecks(&SchedulerParam{content, publishDate, tid, true, capturingMetrics, mockEnvironments}, subscribedFeeds, endpointSpecificChecks, appConfig, metricSink, nil)
	for {
		if capturingMetrics.Len() == mockEnvironments.Len() {
			return capturingMetrics
		}

		time.Sleep(1 * time.Second)
	}
}
