package checks

import (
	"fmt"
	"testing"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/stretchr/testify/assert"
)

func TestIsCurrentOperationFinished_ContentNeo4jCheck_InvalidContent(t *testing.T) {
	currentTid := "tid_1234"
	testResponse := `{ "uuid" : "1234-1234"`
	response := buildResponse(200, testResponse)
	defer response.Body.Close()
	contentCheck := &ContentNeo4jCheck{
		mockHTTPCaller(t, "tid_pam_1234", response),
	}
	log := logger.NewUPPLogger("test", "PANIC")

	pm := newPublishMetricBuilder().withTID(currentTid).build()
	finished, _ := contentCheck.isCurrentOperationFinished(NewPublishCheck(pm, "", "", 0, 0, nil, nil, log))
	assert.False(t, finished, "Expected error.")
}

func TestIsCurrentOperationFinished_ContentNeo4jCheck_InvalidUUID(t *testing.T) {
	currentTid := "tid_1234"
	testResponse := `{ "uuid" : "1234-1235"}`
	response := buildResponse(200, testResponse)
	defer response.Body.Close()
	contentCheck := &ContentNeo4jCheck{
		mockHTTPCaller(t, "tid_pam_1234", response),
	}
	log := logger.NewUPPLogger("test", "PANIC")

	pm := newPublishMetricBuilder().withTID(currentTid).build()
	finished, _ := contentCheck.isCurrentOperationFinished(NewPublishCheck(pm, "", "", 0, 0, nil, nil, log))
	assert.False(t, finished, "Expected error.")
}

func TestIsCurrentOperationFinished_ContentNeo4jCheck_Finished(t *testing.T) {
	currentTid := "tid_1234"
	testResponse := fmt.Sprintf(`{ "uuid" : "1234-1234", "publishReference" : "%s"}`, currentTid)
	response := buildResponse(200, testResponse)
	defer response.Body.Close()
	contentCheck := &ContentNeo4jCheck{
		mockHTTPCaller(t, "tid_pam_1234", response),
	}
	log := logger.NewUPPLogger("test", "PANIC")

	pm := newPublishMetricBuilder().withUUID("1234-1234").withTID(currentTid).build()
	finished, _ := contentCheck.isCurrentOperationFinished(NewPublishCheck(pm, "", "", 0, 0, nil, nil, log))
	assert.True(t, finished, "operation should have finished successfully")
}

func TestIsCurrentOperationFinished_ContentNeo4jCheck_WithAuthentication(t *testing.T) {
	currentTid := "tid_5678"
	testResponse := fmt.Sprintf(`{ "uuid" : "1234-1234", "publishReference" : "%s"}`, currentTid)
	username := "jdoe"
	password := "frodo"
	response := buildResponse(200, testResponse)
	defer response.Body.Close()
	contentCheck := &ContentNeo4jCheck{
		mockAuthenticatedHTTPCaller(t, "tid_pam_5678", username, password, response),
	}
	log := logger.NewUPPLogger("test", "PANIC")

	pm := newPublishMetricBuilder().withUUID("1234-1234").withTID(currentTid).build()
	finished, _ := contentCheck.isCurrentOperationFinished(NewPublishCheck(pm, username, password, 0, 0, nil, nil, log))
	assert.True(t, finished, "operation should have finished successfully")
}

func TestIsCurrentOperationFinished_ContentNeo4jCheck_NotFinished(t *testing.T) {
	currentTid := "tid_1234"
	testResponse := `{ "uuid" : "1234-1234", "publishReference" : "tid_1235"}`
	response := buildResponse(200, testResponse)
	defer response.Body.Close()
	contentCheck := &ContentNeo4jCheck{
		mockHTTPCaller(t, "tid_pam_1234", response),
	}
	log := logger.NewUPPLogger("test", "PANIC")

	pm := newPublishMetricBuilder().withTID(currentTid).build()
	finished, _ := contentCheck.isCurrentOperationFinished(NewPublishCheck(pm, "", "", 0, 0, nil, nil, log))
	assert.False(t, finished, "Expected failure.")
}

func TestIsCurrentOperationFinished_ContentNeo4jCheck_MarkedDeleted_Finished(t *testing.T) {
	currentTid := "tid_1234"
	testResponse := fmt.Sprintf(`{ "uuid" : "1234-1234", "publishReference" : "%s"}`, currentTid)
	response := buildResponse(404, testResponse)
	defer response.Body.Close()
	contentCheck := &ContentNeo4jCheck{
		mockHTTPCaller(t, "tid_pam_1234", response),
	}
	log := logger.NewUPPLogger("test", "PANIC")

	pm := newPublishMetricBuilder().withTID(currentTid).withMarkedDeleted(true).build()
	finished, _ := contentCheck.isCurrentOperationFinished(NewPublishCheck(pm, "", "", 0, 0, nil, nil, log))
	assert.True(t, finished, "operation should have finished successfully.")
}

func TestIsCurrentOperationFinished_ContentNeo4jCheck_MarkedDeleted_NotFinished(t *testing.T) {
	currentTid := "tid_1234"
	testResponse := fmt.Sprintf(`{ "uuid" : "1234-1234", "publishReference" : "%s"}`, currentTid)
	response := buildResponse(200, testResponse)
	defer response.Body.Close()
	contentCheck := &ContentNeo4jCheck{
		mockHTTPCaller(t, "tid_pam_1234", response),
	}
	log := logger.NewUPPLogger("test", "PANIC")

	pm := newPublishMetricBuilder().withTID(currentTid).withMarkedDeleted(true).build()
	finished, _ := contentCheck.isCurrentOperationFinished(NewPublishCheck(pm, "", "", 0, 0, nil, nil, log))
	assert.False(t, finished, "operation should not have finished")
}
