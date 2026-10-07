# Prayer Reminder

Aplikasi pengingat waktu sholat untuk macOS, dibangun dengan Go (Wails v3) dan React + TypeScript.

Aplikasi berjalan di menu bar (system tray), menampilkan jadwal sholat harian lengkap dengan
hitung mundur ke sholat berikutnya, dan mengirim notifikasi native tepat pada waktunya.

## Fitur

- **Jadwal sholat harian** — Subuh, Dzuhur, Ashar, Maghrib, Isya
- **Hitung mundur** ke waktu sholat berikutnya, diperbarui tiap detik
- **Notifikasi native macOS** tepat saat masuk waktu sholat, dengan pengecekan permission
- **Fallback window** jika native notification gagal
- **Tes notifikasi** dari dalam aplikasi
- **System tray (menu bar)** — aplikasi tetap hidup saat jendela ditutup, pengingat tetap berjalan
- **Autocomplete kota** — daftar kota besar Indonesia dibundel, pencarian instan tanpa internet
- **Pilihan metode kalkulasi** — daftar metode diambil dari API, bisa disinkron ulang
- **Cache offline** — jadwal & pengaturan disimpan di SQLite; jadwal terakhir tetap tampil tanpa koneksi
- **Tanggal Hijriah** ditampilkan bersama tanggal Masehi

## Stack

| Bagian | Teknologi |
|---|---|
| Backend | Go 1.26, Wails v3 |
| Frontend | React 18, TypeScript, Vite |
| Penyimpanan | SQLite (`modernc.org/sqlite`, pure Go tanpa CGO) |
| Sumber data | [AlAdhan API](https://aladhan.com/prayer-times-api) |

## Prasyarat

- macOS 12+
- [Go 1.26+](https://go.dev/dl/)
- [Node.js 20+](https://nodejs.org/)
- Xcode Command Line Tools (`xcode-select --install`)
- Wails v3 CLI:

  ```bash
  go install github.com/wailsapp/wails/v3/cmd/wails3@latest
  ```

Pastikan `$(go env GOPATH)/bin` ada di `PATH`.

## Menjalankan

```bash
# Mode pengembangan dengan hot reload
wails3 dev

# Build untuk produksi (binary)
wails3 build

# Buat bundle .app macOS
wails3 package

# Jalankan aplikasi
open bin/prayer-reminder.app
```

### Menjalankan test

```bash
go test ./...
```

### Notifikasi macOS

Saat pertama kali dijalankan, Prayer Reminder akan meminta izin untuk mengirim
notifikasi. Klik **Allow** agar pengingat dapat muncul sebagai banner macOS.

Jika notifikasi tidak muncul, buka:

**System Settings → Notifications → Prayer Reminder**

Pastikan **Allow Notifications** aktif dan pilih gaya notifikasi **Banners**
atau **Alerts**. Tombol **Tes notifikasi** di dalam aplikasi dapat digunakan
untuk menguji notifikasi tanpa menunggu waktu sholat berikutnya.

Jika native notification gagal atau diblokir, Prayer Reminder akan membuka
window aplikasi dan menampilkan banner reminder sebagai fallback.

## Versi dan nama aplikasi

Versi saat ini: **1.0.1**

Nama aplikasi yang ditampilkan di macOS adalah **Prayer Reminder**. Nama
executable dan bundle menggunakan format teknis **prayer-reminder** dan
**prayer-reminder.app**. Bundle identifier aplikasi adalah:

```text
com.hafizhandifaisa.prayer-reminder
```

### Build bundle macOS

```bash
wails3 package
open bin/prayer-reminder.app
```

## Struktur Proyek

```
.
├── main.go                  # Entry point: window, tray, registrasi service
├── app.go                   # Helper app (data dir, logging, konstanta)
├── prayerservice.go         # Service jadwal sholat (API + cache)
├── reminderservice.go       # Scheduler notifikasi latar belakang
├── internal/
│   ├── prayer/              # Model, client AlAdhan, daftar kota
│   │   ├── types.go
│   │   ├── client.go
│   │   └── cities.go
│   └── store/               # Persistensi SQLite
│       └── store.go
├── frontend/                # UI React + TypeScript
│   ├── src/App.tsx
│   └── public/style.css
└── build/                   # Aset build & bundel (ikon, Info.plist, Taskfile)
```

## Catatan Distribusi

Aplikasi ini **belum di-code-sign dan belum di-notarize**. Saat dijalankan di perangkat macOS
lain, Gatekeeper akan memblokirnya. Untuk membukanya:

**Klik kanan aplikasi → Open → Open**, atau lewat **System Settings → Privacy & Security → Open Anyway**.

## Sumber Data

Waktu sholat diambil dari [AlAdhan API](https://aladhan.com/prayer-times-api). Pengguna dapat memilih
metode kalkulasi (mis. Kementerian Agama Republik Indonesia, Muslim World League, ISNA, dan lainnya)
sesuai lokasi.

## Lisensi

Dirilis di bawah [MIT License](LICENSE).

© 2026 Hafizh Andifaisa
