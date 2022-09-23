package feeds

import (
	"bufio"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/publish-availability-monitor/httpcaller"
)

const NotificationsPush = "Notifications-Push"

type NotificationsPushFeed struct {
	baseNotificationsFeed
	stopFeed     bool
	stopFeedLock *sync.RWMutex
	connected    bool
	apiKey       string
	log          *logger.UPPLogger
}

func (f *NotificationsPushFeed) Start() {
	f.log.Infof("starting notifications-push feed from %v", f.baseURL)
	f.stopFeedLock.Lock()
	defer f.stopFeedLock.Unlock()

	f.stopFeed = false
	go func() {
		if f.httpCaller == nil {
			f.httpCaller = httpcaller.NewCaller(0)
		}

		for f.consumeFeed() {
			time.Sleep(500 * time.Millisecond)
			f.log.Info("Disconnected from Push feed! Attempting to reconnect.")
		}
	}()
}

func (f *NotificationsPushFeed) Stop() {
	f.log.Infof("shutting down notifications push feed for %s", f.baseURL)
	f.stopFeedLock.Lock()
	defer f.stopFeedLock.Unlock()

	f.stopFeed = true
}

func (f *NotificationsPushFeed) FeedType() string {
	return NotificationsPush
}

func (f *NotificationsPushFeed) IsConnected() bool {
	return f.connected
}

func (f *NotificationsPushFeed) isConsuming() bool {
	f.stopFeedLock.RLock()
	defer f.stopFeedLock.RUnlock()

	return !f.stopFeed
}

func (f *NotificationsPushFeed) consumeFeed() bool {
	tid := f.buildNotificationsTID()
	log := f.log.WithTransactionID(tid)

	resp, err := f.httpCaller.DoCall(httpcaller.Config{
		URL:      f.baseURL,
		Username: f.username,
		Password: f.password,
		APIKey:   f.apiKey,
		TID:      tid,
	})
	if err != nil {
		log.WithError(err).Error("Sending request failed")
		return f.isConsuming()
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Errorf("Received invalid statusCode: [%v]", resp.StatusCode)
		return f.isConsuming()
	}

	log.Info("Reconnected to push feed!")
	f.connected = true
	defer func() { f.connected = false }()

	br := bufio.NewReader(resp.Body)
	for {
		if !f.isConsuming() {
			log.Info("stop consuming feed")
			break
		}
		f.purgeObsoleteNotifications()

		event, err := br.ReadString('\n')
		if err != nil {
			log.WithError(err).Info("Disconnected from push feed")
			return f.isConsuming()
		}

		trimmed := strings.TrimSpace(event)
		if trimmed == "" {
			continue
		}

		data := strings.TrimPrefix(trimmed, "data: ")
		var notifications []Notification

		if err = json.Unmarshal([]byte(data), &notifications); err != nil {
			log.WithError(err).Error("Error unmarshalling notifications")
			continue
		}

		if len(notifications) == 0 {
			continue
		}

		f.storeNotifications(notifications)
	}

	return false
}

func (f *NotificationsPushFeed) storeNotifications(notifications []Notification) {
	f.notificationsLock.Lock()
	defer f.notificationsLock.Unlock()

	for _, n := range notifications {
		uuid := parseUUIDFromURL(n.ID)
		var history []*Notification
		var found bool
		if history, found = f.notifications[uuid]; !found {
			history = make([]*Notification, 0)
		}

		// Linter warning: Implicit memory aliasing in for loop. (gosec)
		// The implementation is based on the assumption that notification-push service
		// will push notifications slice containing only a single item which migh lead to bugs in the future.
		// If the "notifications" slice contains more than one item
		// for every iteration the code will capture only the first one.
		// nolint:gosec
		history = append(history, &n)
		f.notifications[uuid] = history
	}
}

func (f *NotificationsPushFeed) buildNotificationsTID() string {
	return "tid_pam_notifications_push_" + time.Now().Format(time.RFC3339)
}
