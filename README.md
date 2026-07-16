# Plant Monitoring System

Sistem monitoring tanaman berbasis Internet of Things yang mengintegrasikan
ESP32, Blynk Cloud, backend Go, frontend React, dan PostgreSQL Supabase.

## Fitur utama

- Monitoring suhu udara
- Monitoring kelembapan udara
- Monitoring dua sensor kelembapan tanah
- Dashboard berbasis React
- Riwayat pembacaan sensor
- Decision Support System kondisi tanaman
- Pemantauan status ESP32
- Pengambilan data melalui Blynk
- Direct device ingestion endpoint
- Autentikasi Admin dan Viewer
- Audit log aktivitas keamanan
- Database backup dan retensi
- Cloudflare Tunnel untuk akses jarak jauh

## Arsitektur

```text
ESP32
  │
  ├── Blynk Cloud
  │       │
  │       └── Go Collector
  │
  └── Direct Device API
          │
          ▼
     Go Backend
          │
          ├── React Production Build
          │
          └── Supabase PostgreSQL