package feeds

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/Financial-Times/publish-availability-monitor/checks"
)

const NotificationsPull = "Notifications-Pull"

type NotificationsPullFeed struct {
	baseNotificationsFeed
	notificationsUrl     string
	notificationsUrlLock *sync.Mutex
	interval             int
	ticker               *time.Ticker
	poller               chan struct{}
}

// ignore unused field (e.g. requestUrl)
type notificationsResponse struct {
	Notifications []Notification
	Links         []Link
}

func (f *NotificationsPullFeed) Start() {
	if f.httpCaller == nil {
		f.httpCaller = checks.NewHttpCaller(10)
	}

	f.ticker = time.NewTicker(time.Duration(f.interval) * time.Second)
	f.poller = make(chan struct{})
	go func() {
		for {
			select {
			case <-f.ticker.C:
				go func() {
					f.pollNotificationsFeed()
					f.purgeObsoleteNotifications()
				}()
			case <-f.poller:
				f.ticker.Stop()
				return
			}
		}
	}()
}

func (f *NotificationsPullFeed) Stop() {
	infoLogger.Printf("shutting down notifications pull feed for %s", f.baseUrl)
	close(f.poller)
}

func (f *NotificationsPullFeed) FeedType() string {
	return NotificationsPull
}

func (f *NotificationsPullFeed) pollNotificationsFeed() {
	f.notificationsUrlLock.Lock()
	defer f.notificationsUrlLock.Unlock()

	txId := f.buildNotificationsTxId()
	resp, err := f.httpCaller.DoCall(f.notificationsUrl, f.username, f.password, txId)

	if err != nil {
		infoLogger.Printf("error calling notifications %s", f.notificationsUrl)
		return
	}
	defer cleanupResp(resp)

	if resp.StatusCode != 200 {
		infoLogger.Printf("Notifications [%s] status NOT OK: [%d]", f.notificationsUrl, resp.StatusCode)
		return
	}

	var notifications notificationsResponse
	err = json.NewDecoder(resp.Body).Decode(&notifications)
	if err != nil {
		infoLogger.Printf("Cannot decode json response: [%s]", err.Error())
		return
	}

	f.notificationsLock.Lock()
	defer f.notificationsLock.Unlock()

	for _, n := range notifications.Notifications {
		uuid := parseUuidFromUrl(n.ID)
		var history []*Notification
		var found bool
		if history, found = f.notifications[uuid]; !found {
			history = make([]*Notification, 0)
		}

		history = append(history, &n)
		f.notifications[uuid] = history
	}

	f.notificationsUrl = notifications.Links[0].Href
}

func (f *NotificationsPullFeed) buildNotificationsTxId() string {
	return "tid_pam_notifications_pull_" + time.Now().Format(time.RFC3339)
}
