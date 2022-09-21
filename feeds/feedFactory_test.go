package feeds

import (
	"net/url"
	"testing"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/stretchr/testify/assert"
)

func TestNewPullFeed(t *testing.T) {
	baseUrl, _ := url.Parse("http://www.example.org/")
	log := logger.NewUPPLogger("test", "PANIC")

	actual := NewNotificationsFeed("notifications", *baseUrl, 10, 10, "expectedUser", "expectedPwd", "", log)
	assert.IsType(t, (*NotificationsPullFeed)(nil), actual, "expected a NotificationsPullFeed")

	npf := actual.(*NotificationsPullFeed)
	assert.Equal(t, "expectedUser", npf.username)
	assert.Equal(t, "expectedPwd", npf.password)
}

func TestNewPushFeed(t *testing.T) {
	baseUrl, _ := url.Parse("http://www.example.org/")
	log := logger.NewUPPLogger("test", "PANIC")

	actual := NewNotificationsFeed("notifications-push", *baseUrl, 10, 10, "expectedUser", "expectedPwd", "expectedApiKey", log)
	assert.IsType(t, (*NotificationsPushFeed)(nil), actual, "expected a NotificationsPushFeed")

	npf := actual.(*NotificationsPushFeed)
	assert.Equal(t, "expectedApiKey", npf.apiKey)
	assert.Equal(t, "expectedUser", npf.username)
	assert.Equal(t, "expectedPwd", npf.password)
}
