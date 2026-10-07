package prayer

import "testing"

func TestSearchCities(t *testing.T) {
	// Empty query returns no suggestions.
	if got := SearchCities("", 10); len(got) != 0 {
		t.Errorf("empty query should return nothing, got %d", len(got))
	}

	// Prefix matches should rank first.
	got := SearchCities("band", 10)
	if len(got) == 0 {
		t.Fatal("expected results for 'band'")
	}
	if got[0].Name != "Bandung" {
		t.Errorf("expected Bandung first, got %q", got[0].Name)
	}
	for _, c := range got {
		if c.Country != "Indonesia" {
			t.Errorf("city %q has unexpected country %q", c.Name, c.Country)
		}
	}

	// Case-insensitive matching.
	lower := SearchCities("jakarta", 5)
	upper := SearchCities("JAKARTA", 5)
	if len(lower) != len(upper) || len(lower) == 0 {
		t.Errorf("case-insensitive search mismatch: %d vs %d", len(lower), len(upper))
	}

	// Result limit is respected.
	if got := SearchCities("a", 3); len(got) > 3 {
		t.Errorf("limit not respected, got %d", len(got))
	}

	// Unknown query returns empty.
	if got := SearchCities("zzzznotacity", 5); len(got) != 0 {
		t.Errorf("expected no results, got %d", len(got))
	}
}
