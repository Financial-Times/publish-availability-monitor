package main

import (
	"testing"
	"net/http/httptest"
	"net/http"
	"github.com/satori/go.uuid"
	"strings"
	"encoding/base64"
	"github.com/stretchr/testify/assert"
)


func getTestGtgEnvironment(t *testing.T, status int) (*httptest.Server, Environment) {
	uniq := uuid.NewV4().String()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/__gtg", r.URL.Path, "The check should request for the /__gtg endpoint!")

		auth := r.Header.Get("Authorization")
		b64, _ := base64.StdEncoding.DecodeString(strings.Split(auth, " ")[1])
		credentials := string(b64)

		assert.Equal(t, uniq, strings.Split(credentials, ":")[1], "Credentials supplied to GTG endpoint do not much the credentials we expected!")

		w.WriteHeader(status)
	}))

	infoLogger.Printf("Started test gtg endpoint on [%s] with uuid [%s]", ts.URL, uniq)

	return ts, Environment {
		Name: uniq,
		Username: "next",
		Password: uniq,
		ReadUrl: ts.URL,
	}
}

func TestAllClustersHealthy(t *testing.T){
	mock1, environment := getTestGtgEnvironment(t, 200)
	mock2, otherEnvironment := getTestGtgEnvironment(t, 200)
	mock3, anotherEnvironment := getTestGtgEnvironment(t, 200)

	defer mock1.Close()
	defer mock2.Close()
	defer mock3.Close()

	envs := map [string]Environment{
		environment.Name: environment,
		otherEnvironment.Name: otherEnvironment,
		anotherEnvironment.Name: anotherEnvironment,
	}

	ignore := runGtgChecks(envs, environment)

	assert.False(t, ignore, "All clusters are healthy! This should NOT be ignored!")
}

func TestRequestedClusterHealthy(t *testing.T){
	mock1, environment := getTestGtgEnvironment(t, 200)
	mock2, otherEnvironment := getTestGtgEnvironment(t, 500)
	mock3, anotherEnvironment := getTestGtgEnvironment(t, 500)

	defer mock1.Close()
	defer mock2.Close()
	defer mock3.Close()

	envs := map [string]Environment{
		environment.Name: environment,
		otherEnvironment.Name: otherEnvironment,
		anotherEnvironment.Name: anotherEnvironment,
	}

	ignore := runGtgChecks(envs, environment)

	assert.False(t, ignore, "The cluster we checked on is healthy! This should NOT be ignored!")
}

func TestOtherClusterHealthy(t *testing.T){
	mock1, environment := getTestGtgEnvironment(t, 500)
	mock2, otherEnvironment := getTestGtgEnvironment(t, 200)
	mock3, anotherEnvironment := getTestGtgEnvironment(t, 500)

	defer mock1.Close()
	defer mock2.Close()
	defer mock3.Close()

	envs := map [string]Environment{
		environment.Name: environment,
		otherEnvironment.Name: otherEnvironment,
		anotherEnvironment.Name: anotherEnvironment,
	}

	ignore := runGtgChecks(envs, environment)

	assert.True(t, ignore, "At least one other cluster is healthy! This SHOULD be ignored!")
}

func TestOtherClustersHealthy(t *testing.T){
	mock1, environment := getTestGtgEnvironment(t, 500)
	mock2, otherEnvironment := getTestGtgEnvironment(t, 200)
	mock3, anotherEnvironment := getTestGtgEnvironment(t, 200)

	defer mock1.Close()
	defer mock2.Close()
	defer mock3.Close()

	envs := map [string]Environment{
		environment.Name: environment,
		otherEnvironment.Name: otherEnvironment,
		anotherEnvironment.Name: anotherEnvironment,
	}

	ignore := runGtgChecks(envs, environment)

	assert.True(t, ignore, "At least one other cluster is healthy! This SHOULD be ignored!")
}

func TestAllClustersUnhealthy(t *testing.T){
	mock1, environment := getTestGtgEnvironment(t, 500)
	mock2, otherEnvironment := getTestGtgEnvironment(t, 500)
	mock3, anotherEnvironment := getTestGtgEnvironment(t, 500)

	defer mock1.Close()
	defer mock2.Close()
	defer mock3.Close()

	envs := map [string]Environment{
		environment.Name: environment,
		otherEnvironment.Name: otherEnvironment,
		anotherEnvironment.Name: anotherEnvironment,
	}

	ignore := runGtgChecks(envs, environment)

	assert.False(t, ignore, "None of our clusters are healthy! This should NOT be ignored!")
}
