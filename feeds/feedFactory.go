package feeds

import (
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Financial-Times/go-logger/v2"
)

func NewNotificationsFeed(name string, baseURL url.URL, expiry, interval int, username, password, apiKey string, log *logger.UPPLogger) Feed {
	if isNotificationsPullFeed(name) {
		return newNotificationsPullFeed(name, baseURL, expiry, interval, username, password, log)
	} else if isNotificationsPushFeed(name) {
		return newNotificationsPushFeed(name, baseURL, expiry, interval, username, password, apiKey, log)
	}

	return nil
}

func isNotificationsPullFeed(feedName string) bool {
	return feedName == "notifications" ||
		feedName == "list-notifications" ||
		feedName == "page-notifications"
}

func isNotificationsPushFeed(feedName string) bool {
	return strings.HasSuffix(feedName, "notifications-push")
}

func newNotificationsPullFeed(name string, baseURL url.URL, expiry, interval int, username, password string, log *logger.UPPLogger) *NotificationsPullFeed {
	feedURL := baseURL.String()

	bootstrapValues := baseURL.Query()
	bootstrapValues.Add("since", time.Now().Format(time.RFC3339))
	baseURL.RawQuery = ""

	log.Infof("constructing NotificationsPullFeed for [%s], baseURL = [%s], bootstrapValues = [%s]", feedURL, baseURL.String(), bootstrapValues.Encode())
	return &NotificationsPullFeed{
		baseNotificationsFeed: baseNotificationsFeed{
			feedName:          name,
			baseURL:           feedURL,
			username:          username,
			password:          password,
			expiry:            expiry + 2*interval,
			notifications:     make(map[string][]*Notification),
			notificationsLock: &sync.RWMutex{},
		},
		notificationsURL:         baseURL.String(),
		notificationsQueryString: bootstrapValues.Encode(),
		notificationsURLLock:     &sync.Mutex{},
		interval:                 interval,
		log:                      log,
	}
}

func newNotificationsPushFeed(name string, baseURL url.URL, expiry int, interval int, username string, password string, apiKey string, log *logger.UPPLogger) *NotificationsPushFeed {
	log.Infof("constructing NotificationsPushFeed, bootstrapUrl = [%s]", baseURL.String())
	return &NotificationsPushFeed{
		baseNotificationsFeed: baseNotificationsFeed{
			feedName:          name,
			baseURL:           baseURL.String(),
			username:          username,
			password:          password,
			expiry:            expiry + 2*interval,
			notifications:     make(map[string][]*Notification),
			notificationsLock: &sync.RWMutex{},
		},
		stopFeed:     true,
		stopFeedLock: &sync.RWMutex{},
		apiKey:       apiKey,
		log:          log,
	}
}
