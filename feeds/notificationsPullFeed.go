package feeds

import (
	"encoding/json"
	"net/url"
	"sync"
	"time"

	"github.com/Financial-Times/publish-availability-monitor/v2/checks"
	log "github.com/sirupsen/logrus"
)

const NotificationsPull = "Notifications-Pull"

type NotificationsPullFeed struct {
	baseNotificationsFeed
	notificationsUrl         string
	notificationsQueryString string
	notificationsUrlLock     *sync.Mutex
	interval                 int
	ticker                   *time.Ticker
	poller                   chan struct{}
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
	log.Infof("shutting down notifications pull feed for %s", f.baseUrl)
	close(f.poller)
}

func (f *NotificationsPullFeed) FeedType() string {
	return NotificationsPull
}

func (f *NotificationsPullFeed) pollNotificationsFeed() {
	f.notificationsUrlLock.Lock()
	defer f.notificationsUrlLock.Unlock()

	txId := f.buildNotificationsTxId()
	notificationsUrl := f.notificationsUrl + "?" + f.notificationsQueryString
	resp, err := f.httpCaller.DoCall(checks.Config{Url: notificationsUrl, Username: f.username, Password: f.password, TxId: txId})

	if err != nil {
		log.WithField("transaction_id", txId).WithError(err).Errorf("error calling notifications %s", notificationsUrl)
		return
	}
	defer cleanupResp(resp)

	if resp.StatusCode != 200 {
		log.WithField("transaction_id", txId).Errorf("Notifications [%s] status NOT OK: [%d]", notificationsUrl, resp.StatusCode)
		return
	}

	var notifications notificationsResponse
	err = json.NewDecoder(resp.Body).Decode(&notifications)
	if err != nil {
		log.WithField("transaction_id", txId).Errorf("Cannot decode json response: [%s]", err.Error())
		return
	}

	f.notificationsLock.Lock()
	defer f.notificationsLock.Unlock()

	for _, v := range notifications.Notifications {
		n := v
		uuid := parseUuidFromUrl(n.ID)
		var history []*Notification
		var found bool
		if history, found = f.notifications[uuid]; !found {
			history = make([]*Notification, 0)
		}

		history = append(history, &n)
		f.notifications[uuid] = history
	}

	nextPageUrl, err := url.Parse(notifications.Links[0].Href)
	if err != nil {
		log.Errorf("unparseable next url: [%s]", notifications.Links[0].Href)
		return // and hope that a retry will fix this
	}

	f.notificationsQueryString = nextPageUrl.RawQuery
}

func (f *NotificationsPullFeed) buildNotificationsTxId() string {
	return "tid_pam_notifications_pull_" + time.Now().Format(time.RFC3339)
}
