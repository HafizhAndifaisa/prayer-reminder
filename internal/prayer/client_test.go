package prayer

import (
	"context"
	"testing"
	"time"
)

// TestFetchLive hits the real AlAdhan API. It is skipped in short mode.
func TestFetchLive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live API test in short mode")
	}

	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sched, err := client.Fetch(ctx, DefaultLocation, time.Now())
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(sched.Times) != len(Ordered) {
		t.Fatalf("expected %d prayers, got %d", len(Ordered), len(sched.Times))
	}
	for _, tm := range sched.Times {
		if tm.Time == "" {
			t.Errorf("prayer %s has empty time", tm.Name)
		}
		t.Logf("%s (%s): %s", tm.Name, tm.DisplayName, tm.Time)
	}
	t.Logf("date=%s hijri=%s tz=%s", sched.Readable, sched.Hijri, sched.Timezone)
}

// TestFetchMethodsLive verifies the calculation method list is fetched and
// sorted. It is skipped in short mode.
func TestFetchMethodsLive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live API test in short mode")
	}

	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	methods, err := client.FetchMethods(ctx)
	if err != nil {
		t.Fatalf("fetch methods: %v", err)
	}
	if len(methods) < 10 {
		t.Fatalf("expected many methods, got %d", len(methods))
	}
	for i := 1; i < len(methods); i++ {
		if methods[i-1].ID > methods[i].ID {
			t.Fatalf("methods not sorted at index %d", i)
		}
	}
	for _, m := range methods {
		if m.Name == "" {
			t.Errorf("method id=%d has empty name", m.ID)
		}
	}
	t.Logf("fetched %d methods, first=%+v last=%+v", len(methods), methods[0], methods[len(methods)-1])
}
