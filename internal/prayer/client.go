package prayer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"
)

// aladhanBaseURL is the base endpoint for the AlAdhan prayer times API.
const aladhanBaseURL = "https://api.aladhan.com/v1"

// Client fetches prayer times from the AlAdhan API.
type Client struct {
	http *http.Client
}

// NewClient returns a Client with a sensible request timeout.
func NewClient() *Client {
	return &Client{
		http: &http.Client{Timeout: 15 * time.Second},
	}
}

// aladhanResponse mirrors the subset of the AlAdhan payload we care about.
type aladhanResponse struct {
	Code   int    `json:"code"`
	Status string `json:"status"`
	Data   struct {
		Timings map[string]string `json:"timings"`
		Date    struct {
			Readable string `json:"readable"`
			Hijri    struct {
				Date string `json:"date"`
			} `json:"hijri"`
		} `json:"date"`
		Meta struct {
			Timezone string `json:"timezone"`
			Method   struct {
				Name string `json:"name"`
			} `json:"method"`
		} `json:"meta"`
	} `json:"data"`
}

// Fetch retrieves the prayer schedule for the given location and date.
func (c *Client) Fetch(ctx context.Context, loc Location, date time.Time) (*Schedule, error) {
	// AlAdhan expects dates formatted as DD-MM-YYYY in the path.
	datePath := date.Format("02-01-2006")

	endpoint := fmt.Sprintf("%s/timingsByCity/%s", aladhanBaseURL, url.PathEscape(datePath))
	q := url.Values{}
	q.Set("city", loc.City)
	q.Set("country", loc.Country)
	q.Set("method", strconv.Itoa(loc.Method))
	endpoint += "?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request prayer times: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("aladhan returned status %d", resp.StatusCode)
	}

	var payload aladhanResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode prayer times: %w", err)
	}
	if payload.Code != 200 {
		return nil, fmt.Errorf("aladhan error: %s", payload.Status)
	}

	schedule := &Schedule{
		Date:      date.Format("02-01-2006"),
		Readable:  payload.Data.Date.Readable,
		Hijri:     payload.Data.Date.Hijri.Date,
		Timezone:  payload.Data.Meta.Timezone,
		City:      loc.City,
		Country:   loc.Country,
		Method:    payload.Data.Meta.Method.Name,
		FetchedAt: time.Now(),
	}

	for _, name := range Ordered {
		// AlAdhan sometimes appends a timezone suffix like "04:17 (WIB)".
		raw := payload.Data.Timings[string(name)]
		schedule.Times = append(schedule.Times, Time{
			Name:        name,
			DisplayName: name.Localised(),
			Time:        cleanTime(raw),
		})
	}

	return schedule, nil
}

// cleanTime strips any trailing timezone annotation, returning "HH:MM".
func cleanTime(raw string) string {
	if len(raw) >= 5 {
		return raw[:5]
	}
	return raw
}

// methodsResponse mirrors the AlAdhan /v1/methods payload.
type methodsResponse struct {
	Code int `json:"code"`
	Data map[string]struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"data"`
}

// FetchMethods retrieves the list of calculation methods from AlAdhan.
// The result is sorted by method id for a stable display order.
func (c *Client) FetchMethods(ctx context.Context) ([]Method, error) {
	endpoint := aladhanBaseURL + "/methods"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request methods: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("aladhan returned status %d", resp.StatusCode)
	}

	var payload methodsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode methods: %w", err)
	}
	if payload.Code != 200 {
		return nil, fmt.Errorf("aladhan methods error: code %d", payload.Code)
	}

	methods := make([]Method, 0, len(payload.Data))
	for key, v := range payload.Data {
		name := v.Name
		if name == "" {
			// Some entries omit the display name; fall back to the key.
			name = key
		}
		methods = append(methods, Method{ID: v.ID, Key: key, Name: name})
	}

	sort.Slice(methods, func(i, j int) bool {
		return methods[i].ID < methods[j].ID
	})
	return methods, nil
}
