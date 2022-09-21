package feeds

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Financial-Times/publish-availability-monitor/httpcaller"
)

type baseNotificationsFeed struct {
	feedName          string
	httpCaller        httpcaller.Caller
	baseUrl           string
	username          string
	password          string
	expiry            int
	notifications     map[string][]*Notification
	notificationsLock *sync.RWMutex
}

func cleanupResp(resp *http.Response) {
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

func parseUuidFromUrl(url string) string {
	i := strings.LastIndex(url, "/")
	return url[i+1:]
}

func (f *baseNotificationsFeed) FeedName() string {
	return f.feedName
}

func (f *baseNotificationsFeed) SetCredentials(username string, password string) {
	f.username = username
	f.password = password
}

func (f *baseNotificationsFeed) SetHTTPCaller(httpCaller httpcaller.Caller) {
	f.httpCaller = httpCaller
}

func (f *baseNotificationsFeed) purgeObsoleteNotifications() {
	earliest := time.Now().Add(time.Duration(-f.expiry) * time.Second).Format(time.RFC3339)
	empty := make([]string, 0)

	f.notificationsLock.Lock()
	defer f.notificationsLock.Unlock()

	for u, n := range f.notifications {
		earliestIndex := 0
		for _, e := range n {
			if strings.Compare(e.LastModified, earliest) >= 0 {
				break
			} else {
				earliestIndex++
			}
		}
		f.notifications[u] = n[earliestIndex:]

		if len(f.notifications[u]) == 0 {
			empty = append(empty, u)
		}
	}

	for _, u := range empty {
		delete(f.notifications, u)
	}
}

func (f *baseNotificationsFeed) NotificationsFor(uuid string) []*Notification {
	var history []*Notification
	var found bool

	f.notificationsLock.RLock()
	defer f.notificationsLock.RUnlock()

	if history, found = f.notifications[uuid]; !found {
		history = make([]*Notification, 0)
	}

	return history
}

func (f *baseNotificationsFeed) FeedURL() string {
	return f.baseUrl
}
