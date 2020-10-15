package logformat

import (
	"regexp"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestInfoLogging(t *testing.T) {
	f := NewSLF4JFormatter(`.*/github\.com/Financial-Times/.*`)

	now := time.Now().Round(time.Millisecond)
	msg := "Hello world"

	logEntry := log.Entry{
		Data:    log.Fields{},
		Time:    now,
		Level:   log.InfoLevel,
		Message: msg,
	}

	b, e := f.Format(&logEntry)

	assert.NoError(t, e, "no error expected")
	actual := string(b)
	checkLogEntry(t, "INFO", now, "", msg, actual)
}

func TestWarnLogging(t *testing.T) {
	f := NewSLF4JFormatter(`.*/github\.com/Financial-Times/.*`)

	now := time.Now().Round(time.Millisecond)
	msg := "Uh-oh!"

	logEntry := log.Entry{
		Data:    log.Fields{},
		Time:    now,
		Level:   log.WarnLevel,
		Message: msg,
	}

	b, e := f.Format(&logEntry)

	assert.NoError(t, e, "no error expected")
	actual := string(b)
	checkLogEntry(t, "WARN", now, "", msg, actual)
}

func TestErrorLogging(t *testing.T) {
	f := NewSLF4JFormatter(`.*/github\.com/Financial-Times/.*`)

	now := time.Now().Round(time.Millisecond)
	msg := "Terrible!"

	logEntry := log.Entry{
		Data:    log.Fields{},
		Time:    now,
		Level:   log.ErrorLevel,
		Message: msg,
	}

	b, e := f.Format(&logEntry)

	assert.NoError(t, e, "no error expected")
	actual := string(b)
	checkLogEntry(t, "ERROR", now, "", msg, actual)
}

func TestTxIdLogging(t *testing.T) {
	f := NewSLF4JFormatter(`.*/github\.com/Financial-Times/.*`)

	now := time.Now().Round(time.Millisecond)
	msg := "Hello world"
	txId := "tx_test123"

	logEntry := log.Entry{
		Data:    log.Fields{"transaction_id": txId},
		Time:    now,
		Level:   log.InfoLevel,
		Message: msg,
	}

	b, e := f.Format(&logEntry)

	assert.NoError(t, e, "no error expected")
	actual := string(b)
	checkLogEntry(t, "INFO", now, txId, msg, actual)
}

func checkLogEntry(t *testing.T, expectedLogLevel string, expectedTimestamp time.Time, expectedTxId string, expectedMsg string, actual string) {
	assert.True(t, strings.HasPrefix(actual, expectedLogLevel+" "), "formatted entry should begin with %s", expectedLogLevel)

	loggedTimestamp := regexp.MustCompile(`\[(.*)\]`).FindStringSubmatch(actual)[1]
	timestamp, _ := time.Parse("2006-01-02 15:04:05.000", strings.Replace(loggedTimestamp, ",", ".", -1))
	assert.Equal(t, expectedTimestamp.UTC(), timestamp, "log entry timestamp")

	assert.True(t, regexp.MustCompile(`.* logFormatter_test\.go:\d+: `).MatchString(actual), "formatted entry should contain code location")

	if expectedTxId == "" {
		assert.False(t, regexp.MustCompile(`.* transaction_id=\S+ `).MatchString(actual), "formatted entry should not contain transaction_id")
	} else {
		actualTxId := regexp.MustCompile(`.* transaction_id=(\S+) `).FindStringSubmatch(actual)[1]
		assert.Equal(t, expectedTxId, actualTxId, "transaction_id")
	}

	assert.True(t, strings.Index(actual, expectedMsg) > 0, "formatted entry should contain message text")
}
