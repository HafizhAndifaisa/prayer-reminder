package prayer

import (
	"sort"
	"strings"
)

// City is a selectable location with its country, used for autocomplete.
type City struct {
	Name    string `json:"name"`
	Country string `json:"country"`
}

// indonesiaCities is a bundled list of major Indonesian cities and regencies
// used for offline autocomplete. Names match what the AlAdhan API resolves.
var indonesiaCities = []City{
	{Name: "Banda Aceh", Country: "Indonesia"},
	{Name: "Langsa", Country: "Indonesia"},
	{Name: "Lhokseumawe", Country: "Indonesia"},
	{Name: "Sabang", Country: "Indonesia"},
	{Name: "Medan", Country: "Indonesia"},
	{Name: "Binjai", Country: "Indonesia"},
	{Name: "Pematangsiantar", Country: "Indonesia"},
	{Name: "Tebing Tinggi", Country: "Indonesia"},
	{Name: "Padang Sidempuan", Country: "Indonesia"},
	{Name: "Sibolga", Country: "Indonesia"},
	{Name: "Gunungsitoli", Country: "Indonesia"},
	{Name: "Padang", Country: "Indonesia"},
	{Name: "Bukittinggi", Country: "Indonesia"},
	{Name: "Payakumbuh", Country: "Indonesia"},
	{Name: "Solok", Country: "Indonesia"},
	{Name: "Padang Panjang", Country: "Indonesia"},
	{Name: "Pariaman", Country: "Indonesia"},
	{Name: "Pekanbaru", Country: "Indonesia"},
	{Name: "Dumai", Country: "Indonesia"},
	{Name: "Jambi", Country: "Indonesia"},
	{Name: "Sungai Penuh", Country: "Indonesia"},
	{Name: "Palembang", Country: "Indonesia"},
	{Name: "Prabumulih", Country: "Indonesia"},
	{Name: "Pagar Alam", Country: "Indonesia"},
	{Name: "Lubuklinggau", Country: "Indonesia"},
	{Name: "Bengkulu", Country: "Indonesia"},
	{Name: "Bandar Lampung", Country: "Indonesia"},
	{Name: "Metro", Country: "Indonesia"},
	{Name: "Pangkal Pinang", Country: "Indonesia"},
	{Name: "Tanjungpinang", Country: "Indonesia"},
	{Name: "Batam", Country: "Indonesia"},
	{Name: "Jakarta", Country: "Indonesia"},
	{Name: "Bogor", Country: "Indonesia"},
	{Name: "Depok", Country: "Indonesia"},
	{Name: "Tangerang", Country: "Indonesia"},
	{Name: "Tangerang Selatan", Country: "Indonesia"},
	{Name: "Bekasi", Country: "Indonesia"},
	{Name: "Bandung", Country: "Indonesia"},
	{Name: "Cimahi", Country: "Indonesia"},
	{Name: "Sukabumi", Country: "Indonesia"},
	{Name: "Tasikmalaya", Country: "Indonesia"},
	{Name: "Banjar", Country: "Indonesia"},
	{Name: "Cirebon", Country: "Indonesia"},
	{Name: "Semarang", Country: "Indonesia"},
	{Name: "Surakarta", Country: "Indonesia"},
	{Name: "Magelang", Country: "Indonesia"},
	{Name: "Salatiga", Country: "Indonesia"},
	{Name: "Pekalongan", Country: "Indonesia"},
	{Name: "Tegal", Country: "Indonesia"},
	{Name: "Yogyakarta", Country: "Indonesia"},
	{Name: "Surabaya", Country: "Indonesia"},
	{Name: "Malang", Country: "Indonesia"},
	{Name: "Kediri", Country: "Indonesia"},
	{Name: "Blitar", Country: "Indonesia"},
	{Name: "Madiun", Country: "Indonesia"},
	{Name: "Mojokerto", Country: "Indonesia"},
	{Name: "Pasuruan", Country: "Indonesia"},
	{Name: "Probolinggo", Country: "Indonesia"},
	{Name: "Batu", Country: "Indonesia"},
	{Name: "Denpasar", Country: "Indonesia"},
	{Name: "Mataram", Country: "Indonesia"},
	{Name: "Bima", Country: "Indonesia"},
	{Name: "Kupang", Country: "Indonesia"},
	{Name: "Pontianak", Country: "Indonesia"},
	{Name: "Singkawang", Country: "Indonesia"},
	{Name: "Palangka Raya", Country: "Indonesia"},
	{Name: "Banjarmasin", Country: "Indonesia"},
	{Name: "Banjarbaru", Country: "Indonesia"},
	{Name: "Samarinda", Country: "Indonesia"},
	{Name: "Balikpapan", Country: "Indonesia"},
	{Name: "Bontang", Country: "Indonesia"},
	{Name: "Tarakan", Country: "Indonesia"},
	{Name: "Manado", Country: "Indonesia"},
	{Name: "Bitung", Country: "Indonesia"},
	{Name: "Tomohon", Country: "Indonesia"},
	{Name: "Kotamobagu", Country: "Indonesia"},
	{Name: "Palu", Country: "Indonesia"},
	{Name: "Makassar", Country: "Indonesia"},
	{Name: "Parepare", Country: "Indonesia"},
	{Name: "Palopo", Country: "Indonesia"},
	{Name: "Kendari", Country: "Indonesia"},
	{Name: "Baubau", Country: "Indonesia"},
	{Name: "Gorontalo", Country: "Indonesia"},
	{Name: "Mamuju", Country: "Indonesia"},
	{Name: "Ambon", Country: "Indonesia"},
	{Name: "Tual", Country: "Indonesia"},
	{Name: "Ternate", Country: "Indonesia"},
	{Name: "Tidore Kepulauan", Country: "Indonesia"},
	{Name: "Sofifi", Country: "Indonesia"},
	{Name: "Jayapura", Country: "Indonesia"},
	{Name: "Sorong", Country: "Indonesia"},
	{Name: "Manokwari", Country: "Indonesia"},
	{Name: "Merauke", Country: "Indonesia"},
	{Name: "Timika", Country: "Indonesia"},
	{Name: "Nabire", Country: "Indonesia"},
	{Name: "Wamena", Country: "Indonesia"},
}

// SearchCities returns bundled cities whose name contains the query,
// case-insensitively. An empty query returns an empty slice. Results are
// capped at limit.
func SearchCities(query string, limit int) []City {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return []City{}
	}
	if limit <= 0 {
		limit = 10
	}

	// Rank matches so the most relevant city appears first:
	//   0 = exact name match
	//   1 = a word inside the name starts with the query (e.g. "band" -> "Bandung")
	//   2 = the name starts with the query
	//   3 = the name merely contains the query
	// Ties break toward shorter names so "Bandung" beats "Banda Aceh".
	rank := func(name string) int {
		n := strings.ToLower(name)
		switch {
		case n == q:
			return 0
		}
		for _, word := range strings.Fields(n) {
			if strings.HasPrefix(word, q) {
				return 1
			}
		}
		if strings.HasPrefix(n, q) {
			return 2
		}
		if strings.Contains(n, q) {
			return 3
		}
		return -1
	}

	type scored struct {
		city City
		r    int
	}
	var matches []scored
	for _, c := range indonesiaCities {
		if r := rank(c.Name); r >= 0 {
			matches = append(matches, scored{city: c, r: r})
		}
	}

	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].r != matches[j].r {
			return matches[i].r < matches[j].r
		}
		return len(matches[i].city.Name) < len(matches[j].city.Name)
	})

	if len(matches) > limit {
		matches = matches[:limit]
	}

	out := make([]City, len(matches))
	for i, m := range matches {
		out[i] = m.city
	}
	return out
}
