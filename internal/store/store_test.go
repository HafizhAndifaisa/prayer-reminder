package store

import (
	"context"
	"path/filepath"
	"testing"

	"prayer-reminder/internal/prayer"
)

func TestSettingsAndCache(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	ctx := context.Background()

	// Settings round-trip.
	if _, ok, err := st.GetSetting(ctx, "missing"); err != nil || ok {
		t.Fatalf("expected missing key, got ok=%v err=%v", ok, err)
	}
	if err := st.SetSetting(ctx, "greeting", "salam"); err != nil {
		t.Fatalf("set: %v", err)
	}
	if v, ok, err := st.GetSetting(ctx, "greeting"); err != nil || !ok || v != "salam" {
		t.Fatalf("get: v=%q ok=%v err=%v", v, ok, err)
	}

	// JSON helper with a struct.
	loc := prayer.Location{City: "Bandung", Country: "Indonesia", Method: 20}
	if err := st.SetJSON(ctx, "location", loc); err != nil {
		t.Fatalf("set json: %v", err)
	}
	var got prayer.Location
	if ok, err := st.GetJSON(ctx, "location", &got); err != nil || !ok || got.City != "Bandung" {
		t.Fatalf("get json: %+v ok=%v err=%v", got, ok, err)
	}

	// Schedule cache round-trip.
	sched := &prayer.Schedule{
		Date:     "07-10-2026",
		Readable: "07 Oct 2026",
		Times:    []prayer.Time{{Name: prayer.Fajr, DisplayName: "Subuh", Time: "04:17"}},
	}
	if err := st.SaveSchedule(ctx, sched); err != nil {
		t.Fatalf("save schedule: %v", err)
	}
	loaded, ok, err := st.LoadSchedule(ctx, "07-10-2026")
	if err != nil || !ok {
		t.Fatalf("load schedule: ok=%v err=%v", ok, err)
	}
	if len(loaded.Times) != 1 || loaded.Times[0].Time != "04:17" {
		t.Fatalf("unexpected schedule: %+v", loaded)
	}
	if _, ok, _ := st.LoadSchedule(ctx, "01-01-2000"); ok {
		t.Fatal("expected no cache for unknown date")
	}
}
