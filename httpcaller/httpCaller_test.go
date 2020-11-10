package httpcaller

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func stubServer(t *testing.T, expectedMethod string, expectedHeaders map[string]string, expectedBody []byte) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, expectedMethod, r.Method, "HTTP method")
		for k, v := range expectedHeaders {
			actualValue := r.Header.Get(k)
			assert.Equal(t, v, actualValue, k+" header")
		}

		if expectedBody != nil {
			actualBody := make([]byte, 1024)
			defer r.Body.Close()
			size, _ := r.Body.Read(actualBody)
			assert.Equal(t, expectedBody, actualBody[0:size], "entity")
			//	reflect.DeepEqual()
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello world"))
	}))

	return server
}

func assertExpectedResponse(t *testing.T, resp *http.Response) {
	by := make([]byte, 32)
	i, _ := resp.Body.Read(by)
	body := string(by[:i])

	assert.Equal(t, http.StatusOK, resp.StatusCode, "response status")
	assert.Equal(t, "Hello world", body, "response body")
}

func TestUnauthenticated(t *testing.T) {
	server := stubServer(t, "GET", map[string]string{
		"User-Agent": "UPP Publish Availability Monitor",
	}, nil)
	defer server.Close()

	httpCaller := NewCaller(10)
	resp, err := httpCaller.DoCall(Config{URL: server.URL})
	assert.Nil(t, err, "unexpected error")

	assertExpectedResponse(t, resp)
}

func TestAuthenticated(t *testing.T) {
	username := "scott"
	password := "tiger"
	apiKey := "someApiKey"

	server := stubServer(t, "GET", map[string]string{
		"User-Agent":    "UPP Publish Availability Monitor",
		"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password)),
		"X-Api-Key":     apiKey,
	}, nil)
	defer server.Close()

	httpCaller := NewCaller(10)
	resp, err := httpCaller.DoCall(Config{URL: server.URL, Username: username, Password: password, APIKey: apiKey})
	assert.Nil(t, err, "unexpected error")

	assertExpectedResponse(t, resp)
}

func TestTransactionId(t *testing.T) {
	txId := "tid_myTxId"

	server := stubServer(t, "GET", map[string]string{
		"User-Agent":   "UPP Publish Availability Monitor",
		"X-Request-Id": txId,
	}, nil)
	defer server.Close()

	httpCaller := NewCaller(10)
	resp, err := httpCaller.DoCall(Config{URL: server.URL, TxID: txId})
	assert.Nil(t, err, "unexpected error")

	assertExpectedResponse(t, resp)
}

func TestRequestWithEntity(t *testing.T) {
	contentType := "text/plain"
	body := "Hello world"

	server := stubServer(t, "POST",
		map[string]string{
			"User-Agent":   "UPP Publish Availability Monitor",
			"Content-Type": contentType,
		},
		[]byte(body),
	)
	defer server.Close()

	httpCaller := NewCaller(10)
	resp, err := httpCaller.DoCall(Config{HTTPMethod: "POST", URL: server.URL, ContentType: contentType, Entity: strings.NewReader(body)})
	assert.Nil(t, err, "unexpected error")

	assertExpectedResponse(t, resp)
}

func TestClientDoesRetry(t *testing.T) {
	var retryCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		retryCount++
		if retryCount%2 != 0 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	httpCaller := NewCaller(10)
	_, err := httpCaller.DoCall(Config{HTTPMethod: "GET", URL: server.URL}) //nolint:bodyclose
	assert.NoError(t, err)
	assert.Equal(t, 2, retryCount)
}

func TestClientDoesOnlyTwoRetries(t *testing.T) {
	var retryCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		retryCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	httpCaller := NewCaller(10)
	_, err := httpCaller.DoCall(Config{HTTPMethod: "GET", URL: server.URL}) //nolint:bodyclose
	assert.NoError(t, err)
	assert.Equal(t, 2, retryCount)
}

func TestClientDoesNoRetryAfterSuccessfullResponse(t *testing.T) {
	var retryCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		retryCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	httpCaller := NewCaller(10)
	_, err := httpCaller.DoCall(Config{HTTPMethod: "GET", URL: server.URL}) //nolint:bodyclose
	assert.NoError(t, err)
	assert.Equal(t, 1, retryCount)
}
