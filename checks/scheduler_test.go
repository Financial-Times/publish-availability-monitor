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

func TestStrSliceContains(testing *testing.T) {
	var tests = []struct {
		validTypes  []string
		typeToCheck string
		expected    bool
	}{
		{
			validTypes:  []string{"ValidType", "AnotherValidType"},
			typeToCheck: "InvalidType",
			expected:    false,
		},
		{
			validTypes:  []string{"ValidType", "AnotherValidType"},
			typeToCheck: "AnotherValidType",
			expected:    true,
		},
	}

	for _, t := range tests {
		actual := strSliceContains(t.validTypes, t.typeToCheck)
		if actual != t.expected {
			testing.Errorf("Test Case: %v\nActual: %v", t, actual)
		}
	}
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
					"application/vnd.ft-upp-image",
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

	var genericImage = content.GenericContent{UUID: "e28b12f7-9796-3331-b030-05082f0b8157", Type: "application/vnd.ft-upp-image"}
	capturingMetrics := runScheduleChecks(testing, genericImage, mockEnvironments, appConfig)

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
					"application/vnd.ft-upp-article-internal",
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

	var genericArticleInternal = content.GenericContent{UUID: "a24da1d4-1524-2322-c231-25032d0f8334", Type: "application/vnd.ft-upp-article-internal"}
	capturingMetrics := runScheduleChecks(testing, genericArticleInternal, mockEnvironments, appConfig)

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
