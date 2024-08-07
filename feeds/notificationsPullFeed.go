package feeds

import (
	"encoding/json"
	"github.com/Financial-Times/publish-availability-monitor/config"
	"net/url"
	"sync"
	"time"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/publish-availability-monitor/httpcaller"
)

const NotificationsPull = "Notifications-Pull"

type NotificationsPullFeed struct {
	baseNotificationsFeed
	notificationsURL         string
	notificationsQueryString string
	notificationsURLLock     *sync.Mutex
	interval                 int
	ticker                   *time.Ticker
	poller                   chan struct{}
	log                      *logger.UPPLogger
	xpolicies                []string
	publicationsConfig       *config.PublicationsConfig
}

// ignore unused field (e.g. requestUrl)
type notificationsResponse struct {
	Notifications []Notification
	Links         []Link
}

func (f *NotificationsPullFeed) Start() {
	if f.httpCaller == nil {
		f.httpCaller = httpcaller.NewCaller(10)
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
	f.log.Infof("shutting down notifications pull feed for %s", f.baseURL)
	close(f.poller)
}

func (f *NotificationsPullFeed) FeedType() string {
	return NotificationsPull
}

func (f *NotificationsPullFeed) pollNotificationsFeed() {
	f.notificationsURLLock.Lock()
	defer f.notificationsURLLock.Unlock()

	tid := f.buildNotificationsTID()
	log := f.log.WithTransactionID(tid)
	notificationsURL := f.notificationsURL + "?" + f.notificationsQueryString
	f.log.Info("policies ", f.publicationsConfig.EnabledPublications)
	resp, err := f.httpCaller.DoCall(httpcaller.Config{
		URL:       notificationsURL,
		Username:  f.username,
		Password:  f.password,
		XPolicies: f.publicationsConfig.GetXReadPolicies(),
		TID:       tid,
	})
	if err != nil {
		log.WithError(err).Errorf("error calling notifications %s", notificationsURL)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != 200 {
		log.Errorf("Notifications [%s] status NOT OK: [%d]", notificationsURL, resp.StatusCode)
		return
	}

	var notifications notificationsResponse
	err = json.NewDecoder(resp.Body).Decode(&notifications)
	if err != nil {
		log.WithError(err).Error("Cannot decode json response")
		return
	}

	f.notificationsLock.Lock()
	defer f.notificationsLock.Unlock()

	for _, v := range notifications.Notifications {
		n := v
		uuid := parseUUIDFromURL(n.ID)
		var history []*Notification
		var found bool
		if history, found = f.notifications[uuid]; !found {
			history = make([]*Notification, 0)
		}

		history = append(history, &n)
		f.notifications[uuid] = history
	}

	nextPageURL, err := url.Parse(notifications.Links[0].Href)
	if err != nil {
		log.WithError(err).Errorf("unparseable next url: [%s]", notifications.Links[0].Href)
		return // and hope that a retry will fix this
	}

	f.notificationsQueryString = nextPageURL.RawQuery
}

func (f *NotificationsPullFeed) buildNotificationsTID() string {
	return "tid_pam_notifications_pull_" + time.Now().Format(time.RFC3339)
}
