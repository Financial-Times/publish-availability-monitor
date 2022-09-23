package httpcaller

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/giantswarm/retry-go"
)

// Caller abstracts http calls
type Caller interface {
	DoCall(config Config) (*http.Response, error)
}

// Default implementation of Caller
type DefaultCaller struct {
	client *http.Client
}

type Config struct {
	HTTPMethod  string
	URL         string
	Username    string
	Password    string
	APIKey      string
	TID         string
	ContentType string
	Entity      io.Reader
}

func NewCaller(timeoutSeconds int) DefaultCaller {
	var client http.Client
	if timeoutSeconds > 0 {
		client = http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	} else {
		client = http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}

	return DefaultCaller{&client}
}

// Performs http GET calls using the default http client
func (c DefaultCaller) DoCall(config Config) (resp *http.Response, err error) {
	if config.HTTPMethod == "" {
		config.HTTPMethod = "GET"
	}
	req, err := http.NewRequest(config.HTTPMethod, config.URL, config.Entity)
	if config.Username != "" && config.Password != "" {
		req.SetBasicAuth(config.Username, config.Password)
	}

	if config.APIKey != "" {
		req.Header.Add("X-Api-Key", config.APIKey)
	}

	if config.TID != "" {
		req.Header.Add("X-Request-Id", config.TID)
	}

	if config.ContentType != "" {
		req.Header.Add("Content-Type", config.ContentType)
	}

	req.Header.Add("User-Agent", "UPP Publish Availability Monitor")

	op := func() error {
		resp, err = c.client.Do(req) //nolint:bodyclose
		if err != nil {
			return err
		}

		if resp.StatusCode >= 500 && resp.StatusCode < 600 {
			//Error status code: create an err in order to trigger a retry
			return fmt.Errorf("error status code received: %d", resp.StatusCode)
		}
		return nil
	}

	_ = retry.Do(op, retry.RetryChecker(func(err error) bool { return err != nil }), retry.MaxTries(2))
	return resp, err
}
