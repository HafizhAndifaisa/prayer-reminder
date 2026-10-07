import { useCallback, useEffect, useMemo, useState } from 'react'
import { PrayerService, ReminderService } from '../bindings/prayer-reminder'

// Derive the model types from the generated bindings so we stay in sync with Go.
type Schedule = NonNullable<Awaited<ReturnType<typeof PrayerService.GetToday>>>
type Time = NonNullable<Schedule['times']>[number]
type Method = NonNullable<Awaited<ReturnType<typeof PrayerService.ListMethods>>>[number]
type City = NonNullable<Awaited<ReturnType<typeof PrayerService.SearchCities>>>[number]

// A prayer with a resolved absolute Date, used for countdown calculations.
interface NextPrayer {
  time: Time
  at: Date
}

function parseTimeOnDate(hhmm: string): Date | null {
  const match = /^(\d{1,2}):(\d{2})/.exec(hhmm)
  if (!match) return null
  const [, h, m] = match
  const now = new Date()
  return new Date(now.getFullYear(), now.getMonth(), now.getDate(), Number(h), Number(m), 0, 0)
}

function formatCountdown(ms: number): string {
  if (ms <= 0) return 'sekarang'
  const totalSec = Math.floor(ms / 1000)
  const h = Math.floor(totalSec / 3600)
  const m = Math.floor((totalSec % 3600) / 60)
  const s = totalSec % 60
  if (h > 0) return `${h}j ${m}m ${s}s`
  if (m > 0) return `${m}m ${s}s`
  return `${s}s`
}

function App() {
  const [schedule, setSchedule] = useState<Schedule | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [notifyEnabled, setNotifyEnabled] = useState(true)
  const [now, setNow] = useState<Date>(new Date())

  // Settings modal state.
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [city, setCity] = useState('')
  const [country, setCountry] = useState('')
  const [methodID, setMethodID] = useState(20)
  const [methods, setMethods] = useState<Method[]>([])
  const [saving, setSaving] = useState(false)
  const [syncing, setSyncing] = useState(false)

  // City autocomplete suggestions.
  const [citySuggestions, setCitySuggestions] = useState<City[]>([])
  const [showSuggestions, setShowSuggestions] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const sched = await PrayerService.GetToday()
      setSchedule(sched)
      const enabled = await ReminderService.NotificationsEnabled()
      setNotifyEnabled(enabled)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  // Tick every second so the countdown updates live.
  useEffect(() => {
    const id = setInterval(() => setNow(new Date()), 1000)
    return () => clearInterval(id)
  }, [])

  const next: NextPrayer | null = useMemo(() => {
    const times = schedule?.times
    if (!times || times.length === 0) return null
    for (const t of times) {
      const at = parseTimeOnDate(t.time ?? '')
      if (at && at.getTime() > now.getTime()) {
        return { time: t, at }
      }
    }
    // All prayers passed: next is tomorrow's Fajr.
    const first = times[0]
    if (!first) return null
    const at = parseTimeOnDate(first.time ?? '')
    if (!at) return null
    at.setDate(at.getDate() + 1)
    return { time: first, at }
  }, [schedule, now])

  const toggleNotify = useCallback(async () => {
    const nextValue = !notifyEnabled
    setNotifyEnabled(nextValue)
    try {
      await ReminderService.SetNotificationsEnabled(nextValue)
    } catch (e) {
      setNotifyEnabled(!nextValue)
      setError(e instanceof Error ? e.message : String(e))
    }
  }, [notifyEnabled])

  // Open settings, loading the current location and method list.
  const openSettings = useCallback(async () => {
    setSettingsOpen(true)
    try {
      const loc = await PrayerService.GetLocation()
      setCity(loc.city ?? '')
      setCountry(loc.country ?? '')
      setMethodID(loc.method ?? 20)
      const list = await PrayerService.ListMethods()
      setMethods(list ?? [])
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }, [])

  // Pull the latest method list from the API and cache it.
  const syncMethods = useCallback(async () => {    setSyncing(true)
    setError(null)
    try {
      const list = await PrayerService.SyncMethods()
      setMethods(list ?? [])
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setSyncing(false)
    }
  }, [])

  // Debounced city search: query the backend as the user types.
  useEffect(() => {
    const q = city.trim()
    if (q.length < 2) {
      setCitySuggestions([])
      return
    }
    const id = setTimeout(() => {
      PrayerService.SearchCities(q)
        .then((list) => setCitySuggestions(list ?? []))
        .catch(() => setCitySuggestions([]))
    }, 150)
    return () => clearTimeout(id)
  }, [city])

  // Selecting a suggestion fills both the city and the country.
  const selectCity = useCallback((c: City) => {
    setCity(c.name)
    setCountry(c.country)
    setCitySuggestions([])
    setShowSuggestions(false)
  }, [])

  const saveSettings = useCallback(async () => {
    if (!city.trim() || !country.trim()) {
      setError('Kota dan negara harus diisi.')
      return
    }
    setSaving(true)
    setError(null)
    try {
      await PrayerService.SetLocation(city.trim(), country.trim(), methodID)
      setSettingsOpen(false)
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setSaving(false)
    }
  }, [city, country, methodID, load])

  const countdown = next ? formatCountdown(next.at.getTime() - now.getTime()) : '—'

  return (
    <main className="app">
      <header className="header">
        <div>
          <h1 className="city">{schedule ? `${schedule.city}, ${schedule.country}` : 'Prayer Reminder'}</h1>
          <p className="date">
            {schedule ? schedule.readable : 'Memuat jadwal…'}
            {schedule?.hijri ? <span className="hijri"> · {schedule.hijri} H</span> : null}
          </p>
        </div>
        <div className="header-actions">
          <button className="icon-btn" onClick={() => void openSettings()} title="Pengaturan">
            ⚙
          </button>
          <button className="icon-btn" onClick={() => void load()} disabled={loading} title="Muat ulang">
            {loading ? '…' : '⟳'}
          </button>
        </div>
      </header>

      {error ? <div className="error">{error}</div> : null}

      <section className="countdown">
        <span className="countdown-label">Menuju {next ? next.time.displayName : '—'}</span>
        <span className="countdown-value">{countdown}</span>
      </section>

      <ul className="times">
        {schedule?.times?.map((t) => {
          const isNext = next?.time.name === t.name
          return (
            <li key={t.name} className={isNext ? 'time is-next' : 'time'}>
              <span className="time-name">{t.displayName}</span>
              <span className="time-clock">{t.time}</span>
            </li>
          )
        })}
      </ul>

      <footer className="footer">
        <label className="toggle">
          <input type="checkbox" checked={notifyEnabled} onChange={() => void toggleNotify()} />
          <span>Notifikasi pengingat</span>
        </label>
        <span className="method">{schedule?.method ?? ''}</span>
      </footer>

      {settingsOpen ? (
        <div className="modal-backdrop" onClick={() => setSettingsOpen(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h2 className="modal-title">Pengaturan</h2>

            <label className="field">
              <span>Kota</span>
              <div className="autocomplete">
                <input
                  type="text"
                  value={city}
                  placeholder="mis. Jakarta"
                  autoComplete="off"
                  onChange={(e) => {
                    setCity(e.target.value)
                    setShowSuggestions(true)
                  }}
                  onFocus={() => setShowSuggestions(true)}
                  onBlur={() => {
                    // Delay so a click on a suggestion registers first.
                    setTimeout(() => setShowSuggestions(false), 150)
                  }}
                />
                {showSuggestions && citySuggestions.length > 0 ? (
                  <ul className="suggestions">
                    {citySuggestions.map((c) => (
                      <li key={`${c.name}-${c.country}`}>
                        <button
                          type="button"
                          className="suggestion"
                          onMouseDown={(e) => e.preventDefault()}
                          onClick={() => selectCity(c)}
                        >
                          {c.name}
                          <span className="suggestion-country">{c.country}</span>
                        </button>
                      </li>
                    ))}
                  </ul>
                ) : null}
              </div>
            </label>

            <label className="field">
              <span>Negara</span>
              <input
                type="text"
                value={country}
                placeholder="mis. Indonesia"
                onChange={(e) => setCountry(e.target.value)}
              />
            </label>

            <label className="field">
              <span>Metode kalkulasi</span>
              <select value={methodID} onChange={(e) => setMethodID(Number(e.target.value))}>
                {methods.map((m) => (
                  <option key={m.id} value={m.id}>
                    {m.name}
                  </option>
                ))}
              </select>
            </label>

            <button className="sync-btn" onClick={() => void syncMethods()} disabled={syncing}>
              {syncing ? 'Menyinkron…' : '⟳ Sinkron metode dari API'}
            </button>

            <div className="modal-actions">
              <button className="btn-secondary" onClick={() => setSettingsOpen(false)} disabled={saving}>
                Batal
              </button>
              <button className="btn-primary" onClick={() => void saveSettings()} disabled={saving}>
                {saving ? 'Menyimpan…' : 'Simpan'}
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </main>
  )
}

export default App
