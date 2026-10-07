package main

import (
	"context"
	"fmt"
	"time"

	"prayer-reminder/internal/prayer"
	"prayer-reminder/internal/store"
)

// PrayerService exposes prayer time data to the frontend. It fetches from the
// AlAdhan API, caches results in SQLite, and falls back to the cache when the
// network is unavailable.
type PrayerService struct {
	client *prayer.Client
	store  *store.Store
}

// NewPrayerService wires the API client and local store together.
func NewPrayerService(st *store.Store) *PrayerService {
	return &PrayerService{
		client: prayer.NewClient(),
		store:  st,
	}
}

// GetToday returns the prayer schedule for today at the configured location.
// It prefers a fresh API response and falls back to the cached copy on failure.
func (s *PrayerService) GetToday() (*prayer.Schedule, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	loc, err := s.location(ctx)
	if err != nil {
		return nil, err
	}

	today := time.Now()
	dateKey := today.Format("02-01-2006")

	sched, err := s.client.Fetch(ctx, loc, today)
	if err != nil {
		// Network or API failure: serve the cached schedule if we have one.
		if cached, ok, cacheErr := s.store.LoadSchedule(ctx, dateKey); cacheErr == nil && ok {
			return cached, nil
		}
		return nil, fmt.Errorf("fetch prayer times: %w", err)
	}

	if err := s.store.SaveSchedule(ctx, sched); err != nil {
		// Caching is best-effort; a failure should not break the request.
		logWarn("cache schedule", err)
	}
	return sched, nil
}

// GetLocation returns the currently configured location.
func (s *PrayerService) GetLocation() (prayer.Location, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.location(ctx)
}

// SetLocation persists a new location and clears the cached schedule so the
// next GetToday call fetches fresh times for it.
func (s *PrayerService) SetLocation(city, country string, method int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if city == "" || country == "" {
		return fmt.Errorf("city and country must not be empty")
	}
	loc := prayer.Location{City: city, Country: country, Method: method}
	if err := s.store.SetJSON(ctx, settingLocation, loc); err != nil {
		return err
	}
	// Drop stale schedules so the UI does not show times for the old location.
	if err := s.store.ClearScheduleCache(ctx); err != nil {
		logWarn("clear schedule cache", err)
	}
	return nil
}

// ListMethods returns the available calculation methods. It reads the cached
// list first; when empty it returns a small built-in fallback so the UI always
// has something to show. Use SyncMethods to refresh from the API.
func (s *PrayerService) ListMethods() ([]prayer.Method, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if methods, ok, err := s.store.LoadMethods(ctx); err != nil {
		logWarn("load methods", err)
	} else if ok && len(methods) > 0 {
		return methods, nil
	}
	return fallbackMethods(), nil
}

// SyncMethods fetches the calculation method list from the AlAdhan API and
// caches it. On failure the previously cached list is returned unchanged.
func (s *PrayerService) SyncMethods() ([]prayer.Method, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	methods, err := s.client.FetchMethods(ctx)
	if err != nil {
		// Fall back to whatever we have cached (or the built-in list).
		if cached, ok, _ := s.store.LoadMethods(ctx); ok && len(cached) > 0 {
			return cached, nil
		}
		return nil, fmt.Errorf("fetch methods: %w", err)
	}

	if err := s.store.SaveMethods(ctx, methods); err != nil {
		logWarn("cache methods", err)
	}
	return methods, nil
}

// Version returns the application version.
func (s *PrayerService) Version() string {
	return version
}

// SearchCities returns bundled city suggestions for the given query. It runs
// fully offline against a list of major Indonesian cities.
func (s *PrayerService) SearchCities(query string) []prayer.City {
	return prayer.SearchCities(query, 8)
}

// fallbackMethods is a minimal built-in list used before a successful sync.
func fallbackMethods() []prayer.Method {
	return []prayer.Method{
		{ID: 20, Key: "KEMENAG", Name: "Kementerian Agama Republik Indonesia"},
		{ID: 3, Key: "MWL", Name: "Muslim World League"},
		{ID: 2, Key: "ISNA", Name: "Islamic Society of North America (ISNA)"},
		{ID: 5, Key: "EGYPT", Name: "Egyptian General Authority of Survey"},
		{ID: 4, Key: "MAKKAH", Name: "Umm Al-Qura University, Makkah"},
		{ID: 1, Key: "KARACHI", Name: "University of Islamic Sciences, Karachi"},
		{ID: 17, Key: "JAKIM", Name: "Jabatan Kemajuan Islam Malaysia (JAKIM)"},
	}
}

// location loads the stored location, falling back to the default.
func (s *PrayerService) location(ctx context.Context) (prayer.Location, error) {
	var loc prayer.Location
	ok, err := s.store.GetJSON(ctx, settingLocation, &loc)
	if err != nil {
		return prayer.Location{}, err
	}
	if !ok {
		return prayer.DefaultLocation, nil
	}
	return loc, nil
}
