# NetMonitor API Specification
**Version:** v1.0  
**Base URL:** `http://localhost:8080/api`  
**Format:** JSON (`application/json`)  
**Auth:** Tidak ada (open API)  
**Stack:** Go (Gin) + PostgreSQL (GORM)

---

## Daftar Endpoint

| Method | Path | Grup | Deskripsi |
|--------|------|------|-----------|
| `GET` | `/api/targets` | Targets | Ambil semua target IP |
| `POST` | `/api/targets` | Targets | Tambah target baru |
| `DELETE` | `/api/targets/:id` | Targets | Hapus target berdasarkan ID |
| `GET` | `/api/pings/latest` | Ping | 100 ping terbaru (semua IP) |
| `GET` | `/api/pings/history` | Ping | Riwayat ping per IP / rentang waktu |
| `GET` | `/api/pings/summary` | Ping | Ringkasan uptime & latency per IP |
| `POST` | `/api/surveys` | Survey | Kirim angket kepuasan baru |
| `GET` | `/api/surveys` | Survey | Ambil semua data survey |
| `GET` | `/api/correlation` | Korelasi | Korelasi jaringan vs kepuasan |

---

## 1. Targets

Mengelola daftar IP address atau hostname yang dipantau secara berkala.

---

### 1.1 `GET /api/targets`

Mengambil seluruh daftar target yang terdaftar.

**Query Parameters:** —

**Response `200 OK`**
```json
[
  {
    "id": 1,
    "ip_address": "8.8.8.8",
    "label": "Google DNS",
    "is_active": true,
    "created_at": "2026-01-15T08:00:00+07:00"
  },
  {
    "id": 2,
    "ip_address": "1.1.1.1",
    "label": "Cloudflare DNS",
    "is_active": true,
    "created_at": "2026-01-15T08:00:00+07:00"
  }
]
```

| Field | Type | Deskripsi |
|-------|------|-----------|
| `id` | integer | ID unik target (auto-increment) |
| `ip_address` | string | Alamat IP atau hostname target |
| `label` | string | Nama/deskripsi target |
| `is_active` | boolean | Status aktif target (`true` = dipantau) |
| `created_at` | datetime | Waktu target ditambahkan (ISO 8601) |

---

### 1.2 `POST /api/targets`

Menambahkan target IP baru ke dalam sistem monitoring.

**Request Body**
```json
{
  "ip_address": "192.168.1.1",
  "label": "Gateway Lokal"
}
```

| Parameter | Type | Required | Deskripsi |
|-----------|------|----------|-----------|
| `ip_address` | string | ✅ | Alamat IP atau hostname. Harus unik. |
| `label` | string | ❌ | Nama deskriptif untuk identifikasi |

**Response `201 Created`**
```json
{
  "id": 4,
  "ip_address": "192.168.1.1",
  "label": "Gateway Lokal",
  "is_active": true,
  "created_at": "2026-03-05T10:30:00+07:00"
}
```

> **Catatan:** `is_active` otomatis diset `true` saat target dibuat.

---

### 1.3 `DELETE /api/targets/:id`

Menghapus target berdasarkan ID.

**Path Parameter**

| Parameter | Type | Required | Deskripsi |
|-----------|------|----------|-----------|
| `id` | integer | ✅ | ID target yang akan dihapus |

**Response `200 OK`**
```json
{
  "message": "Target dihapus"
}
```

---

## 2. Ping Results

Hasil monitoring ping yang dikumpulkan otomatis setiap **5 detik** untuk semua target aktif.

---

### 2.1 `GET /api/pings/latest`

Mengambil 100 data ping terbaru dari semua target, diurutkan dari yang paling baru.

**Query Parameters:** —

**Response `200 OK`**
```json
[
  {
    "id": 1024,
    "ip_address": "8.8.8.8",
    "label": "Google DNS",
    "is_reachable": true,
    "latency_ms": 12.45,
    "packet_loss": 0,
    "created_at": "2026-03-05T10:35:00+07:00"
  }
]
```

| Field | Type | Deskripsi |
|-------|------|-----------|
| `id` | integer | ID unik hasil ping |
| `ip_address` | string | IP/host yang di-ping |
| `label` | string | Label target saat ping dilakukan |
| `is_reachable` | boolean | `true` jika host merespon ping |
| `latency_ms` | float | Rata-rata round-trip time (ms) |
| `packet_loss` | float | Persentase packet loss (0–100) |
| `created_at` | datetime | Waktu ping dilakukan |

---

### 2.2 `GET /api/pings/history`

Mengambil riwayat ping berdasarkan IP dan rentang waktu. Cocok untuk grafik time-series.

**Query Parameters**

| Parameter | Type | Required | Deskripsi |
|-----------|------|----------|-----------|
| `ip` | string | ❌ | Filter berdasarkan IP. Jika kosong, ambil semua IP. |
| `hours` | integer | ❌ | Rentang waktu dalam jam ke belakang (default: `1`) |

**Contoh Request**
```
GET /api/pings/history?ip=8.8.8.8&hours=3
```

**Response `200 OK`**
```json
[
  {
    "id": 800,
    "ip_address": "8.8.8.8",
    "label": "Google DNS",
    "is_reachable": true,
    "latency_ms": 11.20,
    "packet_loss": 0,
    "created_at": "2026-03-05T07:35:00+07:00"
  }
]
```

---

### 2.3 `GET /api/pings/summary`

Ringkasan statistik per IP untuk **1 jam terakhir**: uptime, rata-rata latensi, dan status terakhir.

**Query Parameters:** —

**Response `200 OK`**
```json
[
  {
    "ip_address": "8.8.8.8",
    "label": "Google DNS",
    "total_pings": 720,
    "reachable_pings": 720,
    "uptime_percent": 100,
    "avg_latency_ms": 13.75,
    "last_seen": "2026-03-05T10:35:00+07:00",
    "last_status": true
  }
]
```

| Field | Type | Deskripsi |
|-------|------|-----------|
| `ip_address` | string | Alamat IP target |
| `label` | string | Label target |
| `total_pings` | integer | Total ping dalam 1 jam terakhir |
| `reachable_pings` | integer | Jumlah ping yang berhasil |
| `uptime_percent` | float | `(reachable / total) * 100` |
| `avg_latency_ms` | float | Rata-rata latensi dari ping berhasil (ms) |
| `last_seen` | datetime | Waktu ping terakhir dilakukan |
| `last_status` | boolean | Status ping paling terakhir |

---

## 3. Survey Kepuasan

Angket kepuasan pengguna jaringan menggunakan **skala Likert 1–5**.

| Skor | Keterangan |
|------|-----------|
| 1 | Sangat Buruk |
| 2 | Buruk |
| 3 | Cukup |
| 4 | Baik |
| 5 | Sangat Baik |

---

### 3.1 `POST /api/surveys`

Mengirimkan satu entri angket kepuasan dari seorang responden.

**Request Body**
```json
{
  "respondent_name": "Budi Santoso",
  "location": "Lab Jaringan Lt. 2",
  "q1_speed": 4,
  "q2_stability": 3,
  "q3_latency": 4,
  "q4_availability": 5,
  "q5_satisfaction": 4,
  "comment": "Koneksi cukup baik, hanya sesekali lambat"
}
```

| Parameter | Type | Required | Deskripsi |
|-----------|------|----------|-----------|
| `respondent_name` | string | ❌ | Nama responden |
| `location` | string | ❌ | Lokasi responden saat mengisi survey |
| `q1_speed` | integer | ✅ | Kecepatan internet memadai? (1–5) |
| `q2_stability` | integer | ✅ | Koneksi stabil? (1–5) |
| `q3_latency` | integer | ✅ | Latensi terasa rendah? (1–5) |
| `q4_availability` | integer | ✅ | Internet selalu tersedia? (1–5) |
| `q5_satisfaction` | integer | ✅ | Kepuasan keseluruhan? (1–5) |
| `comment` | string | ❌ | Komentar / saran tambahan |

**Response `201 Created`**
```json
{
  "message": "Terima kasih atas penilaian Anda!",
  "avg_score": 4.0,
  "survey": {
    "id": 15,
    "respondent_name": "Budi Santoso",
    "location": "Lab Jaringan Lt. 2",
    "q1_speed": 4,
    "q2_stability": 3,
    "q3_latency": 4,
    "q4_availability": 5,
    "q5_satisfaction": 4,
    "comment": "Koneksi cukup baik",
    "created_at": "2026-03-05T10:45:00+07:00"
  }
}
```

> **Catatan:** `avg_score` = rata-rata Q1+Q2+Q3+Q4+Q5 dibagi 5, dibulatkan 2 desimal.

---

### 3.2 `GET /api/surveys`

Mengambil seluruh data survey, diurutkan dari yang terbaru.

**Query Parameters:** —

**Response `200 OK`**
```json
[
  {
    "id": 15,
    "respondent_name": "Budi Santoso",
    "location": "Lab Jaringan Lt. 2",
    "q1_speed": 4,
    "q2_stability": 3,
    "q3_latency": 4,
    "q4_availability": 5,
    "q5_satisfaction": 4,
    "comment": "Koneksi cukup baik",
    "avg_score": 4.0,
    "created_at": "2026-03-05T10:45:00+07:00"
  }
]
```

---

## 4. Analisis Korelasi

Menghitung **korelasi Pearson** antara metrik kualitas jaringan (latensi, packet loss, uptime) dan skor kepuasan pengguna dari survey, dikelompokkan per hari.

---

### 4.1 `GET /api/correlation`

**Query Parameters**

| Parameter | Type | Required | Deskripsi |
|-----------|------|----------|-----------|
| `days` | integer | ❌ | Jumlah hari ke belakang (default: `7`) |

**Contoh Request**
```
GET /api/correlation?days=14
```

**Response `200 OK`**
```json
{
  "period_days": 14,
  "data": [
    {
      "date": "2026-02-20",
      "avg_latency": 14.32,
      "avg_packet_loss": 0.5,
      "uptime_percent": 99.72,
      "avg_satisfaction": 4.2,
      "survey_count": 5
    }
  ],
  "correlations": {
    "latency_vs_satisfaction": -0.812,
    "uptime_vs_satisfaction": 0.764,
    "packetloss_vs_satisfaction": -0.691
  },
  "interpretation": {
    "latency": "Korelasi kuat positif",
    "uptime": "Korelasi kuat positif",
    "packetloss": "Korelasi sedang positif"
  }
}
```

| Field | Type | Deskripsi |
|-------|------|-----------|
| `period_days` | integer | Jumlah hari yang dianalisis |
| `data` | array | Data harian gabungan ping & survey |
| `data[].date` | string | Tanggal (YYYY-MM-DD, Asia/Jakarta) |
| `data[].avg_latency` | float | Rata-rata latensi harian (ms) |
| `data[].avg_packet_loss` | float | Rata-rata packet loss harian (%) |
| `data[].uptime_percent` | float | Persentase uptime harian |
| `data[].avg_satisfaction` | float | Rata-rata skor kepuasan harian |
| `data[].survey_count` | integer | Jumlah responden pada hari tersebut |
| `correlations` | object | Koefisien Pearson (-1 hingga 1) |
| `correlations.latency_vs_satisfaction` | float | Korelasi latensi vs kepuasan |
| `correlations.uptime_vs_satisfaction` | float | Korelasi uptime vs kepuasan |
| `correlations.packetloss_vs_satisfaction` | float | Korelasi packet loss vs kepuasan |
| `interpretation` | object | Interpretasi tekstual nilai korelasi |

---

### 4.2 Interpretasi Nilai Korelasi

| Rentang \|r\| | Kekuatan | Output Teks |
|--------------|----------|-------------|
| \|r\| ≥ 0.7 | Kuat | `"Korelasi kuat positif/negatif"` |
| 0.4 ≤ \|r\| < 0.7 | Sedang | `"Korelasi sedang positif/negatif"` |
| 0.2 ≤ \|r\| < 0.4 | Lemah | `"Korelasi lemah positif/negatif"` |
| \|r\| < 0.2 | Tidak signifikan | `"Tidak ada korelasi signifikan"` |

> **Catatan:** Latensi & packet loss bersifat inverse terhadap kepuasan. Nilai korelasi **negatif** pada kedua metrik ini menandakan hubungan yang **baik** bagi kualitas layanan (latensi rendah → pengguna puas).

---

## 5. Error Handling

Semua error dikembalikan dalam format JSON dengan field `error`.

| HTTP Status | Kondisi | Contoh Response |
|-------------|---------|-----------------|
| `400 Bad Request` | Body tidak valid / validasi gagal | `{ "error": "Skor harus antara 1-5" }` |
| `404 Not Found` | Resource tidak ditemukan | `{ "error": "record not found" }` |
| `500 Internal Server Error` | Error database / server | `{ "error": "Gagal menyimpan angket" }` |

---

## 6. Data Model

### `targets`

| Column | Type | Keterangan |
|--------|------|-----------|
| `id` | SERIAL PK | Primary key auto-increment |
| `ip_address` | VARCHAR UNIQUE | Alamat IP / hostname (harus unik) |
| `label` | VARCHAR | Nama deskriptif target |
| `is_active` | BOOLEAN | Status monitoring (default: `true`) |
| `created_at` | TIMESTAMP | Waktu pembuatan record |

### `ping_results`

| Column | Type | Keterangan |
|--------|------|-----------|
| `id` | SERIAL PK | Primary key auto-increment |
| `ip_address` | VARCHAR INDEX | IP yang di-ping |
| `label` | VARCHAR | Label target saat ping |
| `is_reachable` | BOOLEAN | Host merespon atau tidak |
| `latency_ms` | FLOAT | Rata-rata RTT (ms) |
| `packet_loss` | FLOAT | Persentase packet loss (0–100) |
| `created_at` | TIMESTAMP INDEX | Waktu ping dilakukan |

### `surveys`

| Column | Type | Keterangan |
|--------|------|-----------|
| `id` | SERIAL PK | Primary key auto-increment |
| `respondent_name` | VARCHAR | Nama responden (opsional) |
| `location` | VARCHAR | Lokasi responden (opsional) |
| `q1_speed` | INTEGER | Skor kecepatan (1–5) |
| `q2_stability` | INTEGER | Skor stabilitas (1–5) |
| `q3_latency` | INTEGER | Skor latensi (1–5) |
| `q4_availability` | INTEGER | Skor ketersediaan (1–5) |
| `q5_satisfaction` | INTEGER | Skor kepuasan keseluruhan (1–5) |
| `comment` | TEXT | Komentar tambahan (opsional) |
| `created_at` | TIMESTAMP INDEX | Waktu pengisian survey |

---

## 7. CORS & Environment

Server mengizinkan request dari origin berikut secara default:

```
http://localhost:5173   # React / Vite dev server
http://localhost:3000   # Create React App / Next.js
```

### Variabel Environment (`.env`)

| Variabel | Default | Keterangan |
|----------|---------|-----------|
| `DB_HOST` | `localhost` | Host database PostgreSQL |
| `DB_PORT` | `5432` | Port database PostgreSQL |
| `DB_USER` | `postgres` | Username database |
| `DB_PASSWORD` | `password` | Password database |
| `DB_NAME` | `netmonitor` | Nama database |
| `PORT` | `8080` | Port HTTP server |
