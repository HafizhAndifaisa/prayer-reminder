// Package prayer provides prayer time retrieval and modelling for the
// Prayer Reminder application. Prayer times are fetched from the AlAdhan API
// and cached locally so the application keeps working while offline.
package prayer

import "time"

// Name identifies one of the five daily prayers.
type Name string

const (
	Fajr    Name = "Fajr"
	Dhuhr   Name = "Dhuhr"
	Asr     Name = "Asr"
	Maghrib Name = "Maghrib"
	Isha    Name = "Isha"
)

// Ordered lists the five daily prayers in the order they occur.
var Ordered = []Name{Fajr, Dhuhr, Asr, Maghrib, Isha}

// Localised returns the Indonesian display name for a prayer.
func (n Name) Localised() string {
	switch n {
	case Fajr:
		return "Subuh"
	case Dhuhr:
		return "Dzuhur"
	case Asr:
		return "Ashar"
	case Maghrib:
		return "Maghrib"
	case Isha:
		return "Isya"
	default:
		return string(n)
	}
}

// Time is a single prayer with its scheduled wall-clock time.
type Time struct {
	Name        Name   `json:"name"`
	DisplayName string `json:"displayName"`
	Time        string `json:"time"` // "HH:MM" in the location's timezone
}

// Schedule holds the prayer times for one calendar day.
type Schedule struct {
	Date      string    `json:"date"`      // "DD-MM-YYYY"
	Readable  string    `json:"readable"`  // e.g. "07 Oct 2026"
	Hijri     string    `json:"hijri"`     // e.g. "26-04-1448"
	Timezone  string    `json:"timezone"`  // IANA name, e.g. "Asia/Jakarta"
	City      string    `json:"city"`
	Country   string    `json:"country"`
	Method    string    `json:"method"`
	Times     []Time    `json:"times"`
	FetchedAt time.Time `json:"fetchedAt"`
}

// Location identifies where prayer times are calculated for.
type Location struct {
	City    string `json:"city"`
	Country string `json:"country"`
	Method  int    `json:"method"` // AlAdhan calculation method id
}

// Method is a prayer time calculation method offered by the AlAdhan API.
type Method struct {
	ID   int    `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

// DefaultLocation is used before the user has chosen their own.
var DefaultLocation = Location{
	City:    "Jakarta",
	Country: "Indonesia",
	Method:  20, // KEMENAG (Kementerian Agama Republik Indonesia)
}
