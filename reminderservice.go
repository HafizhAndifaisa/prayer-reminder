package main

import (
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	prayernotifications "github.com/wailsapp/wails/v3/pkg/services/notifications"

	"prayer-reminder/internal/prayer"
	"prayer-reminder/internal/store"
)

// ReminderService runs a background loop that fires a native notification at
// each prayer time. It keeps running while the window is hidden.
type ReminderService struct {
	app      *application.App
	prayer   *PrayerService
	notifier *prayernotifications.NotificationService
	store    *store.Store

	// notifyEnabled controls whether reminders are delivered.
	notifyEnabled bool

	mu       sync.Mutex
	stop     chan struct{}
	stopOnce sync.Once
}

// NewReminderService constructs the scheduler. Call Start to begin the loop.
func NewReminderService(
	app *application.App,
	prayerSvc *PrayerService,
	notifier *prayernotifications.NotificationService,
	st *store.Store,
) *ReminderService {
	return &ReminderService{
		app:           app,
		prayer:        prayerSvc,
		notifier:      notifier,
		store:         st,
		notifyEnabled: true,
		stop:          make(chan struct{}),
	}
}

// start begins the background reminder loop. It is unexported so it is not
// bound to the frontend, which could otherwise start the loop twice.
func (r *ReminderService) start() {
	go r.loop()
}

// halt stops the reminder loop. It is unexported so it is not bound to the
// frontend.
func (r *ReminderService) halt() {
	r.stopOnce.Do(func() { close(r.stop) })
}

// SetNotificationsEnabled turns reminders on or off from the UI.
func (r *ReminderService) SetNotificationsEnabled(enabled bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notifyEnabled = enabled
}

// NotificationsEnabled reports the current reminder toggle state.
func (r *ReminderService) NotificationsEnabled() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.notifyEnabled
}

// loop wakes periodically, checks whether a prayer time has been reached since
// the last check, and notifies accordingly.
func (r *ReminderService) loop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	lastFired := map[string]bool{}

	for {
		select {
		case <-r.stop:
			return
		case <-ticker.C:
			r.check(lastFired)
		}
	}
}

// check loads today's schedule and fires any prayer whose time has just passed.
func (r *ReminderService) check(lastFired map[string]bool) {
	sched, err := r.prayer.GetToday()
	if err != nil {
		logWarn("reminder: get schedule", err)
		return
	}

	loc := time.Local
	if sched.Timezone != "" {
		if parsed, perr := time.LoadLocation(sched.Timezone); perr == nil {
			loc = parsed
		}
	}
	now := time.Now().In(loc)

	for _, t := range sched.Times {
		at, perr := time.ParseInLocation("15:04", t.Time, loc)
		if perr != nil {
			continue
		}
		at = time.Date(now.Year(), now.Month(), now.Day(), at.Hour(), at.Minute(), 0, 0, loc)

		key := sched.Date + "/" + string(t.Name)
		// Fire once, within a two-minute window after the scheduled time.
		if lastFired[key] || now.Before(at) || now.Sub(at) > 2*time.Minute {
			continue
		}
		lastFired[key] = true
		r.notify(t)
	}
}

// notify sends a native notification for a prayer.
func (r *ReminderService) notify(t prayer.Time) {
	if !r.NotificationsEnabled() {
		return
	}

	body := "Waktunya sholat " + t.DisplayName + " (" + t.Time + ")"
	if err := r.notifier.SendNotification(prayernotifications.NotificationOptions{
		ID:    string(t.Name),
		Title: "Prayer Reminder",
		Body:  body,
	}); err != nil {
		logWarn("send notification", err)
	}
}
