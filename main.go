package main

import (
	"embed"
	"log"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"

	"prayer-reminder/internal/store"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded and served to the UI.
//
//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/trayicon-32.png
var trayIcon []byte

// main is the application entry point. It wires up the SQLite store, the
// prayer and reminder services, a menu bar (system tray) item, and a main
// window. Closing the window hides it while reminders keep running.
func main() {
	// Resolve where to keep the database.
	dir, err := dataDir()
	if err != nil {
		log.Fatalf("resolve data directory: %v", err)
	}

	// Open the local database (settings + schedule cache).
	st, err := store.Open(filepath.Join(dir, "prayer.db"))
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	// Native notification service.
	notifier := notifications.New()

	// Domain services exposed to the frontend.
	prayerSvc := NewPrayerService(st)

	app := application.New(application.Options{
		Name:        "Prayer Reminder",
		Description: "Prayer times and reminders for macOS",
		Services: []application.Service{
			application.NewService(prayerSvc),
			application.NewService(notifier),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			// Keep the app alive in the menu bar when the window is closed so
			// scheduled reminders still fire.
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})

	// Background scheduler that fires notifications at prayer times.
	// It is created after application.New because it needs the app instance,
	// so it is registered explicitly instead of via Options.Services.
	scheduler := NewReminderService(app, prayerSvc, notifier, st)
	app.RegisterService(application.NewService(scheduler))

	// Main window. Closing it hides the window rather than quitting.
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "Prayer Reminder",
		Width:  420,
		Height: 640,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 44,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(10, 14, 24),
		URL:              "/",
	})

	// Hide the app when the window is closed instead of terminating, so the
	// scheduler keeps running in the background.
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})
	scheduler.setWindow(window)

	// Menu bar (system tray) so the app stays reachable while hidden.
	tray := app.SystemTray.New()
	tray.SetLabel("Prayer Reminder")
	// Template icon (black + alpha) adapts automatically to light/dark menus.
	tray.SetTemplateIcon(trayIcon)
	menu := app.NewMenu()
	menu.Add("Show Prayer Reminder").OnClick(func(ctx *application.Context) {
		window.Show()
		window.Focus()
	})
	menu.AddSeparator()
	menu.Add("Quit").OnClick(func(ctx *application.Context) {
		app.Quit()
	})
	tray.SetMenu(menu)

	// Start the reminder loop only after the native application and window are
	// fully initialized. This also makes the first immediate scheduler check
	// safe when it needs to show the window as a notification fallback.
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		scheduler.start()
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
