# Order Service — Studi Kasus Teknis Junior Developer

Track A (Golang). REST API pengelolaan produk, customer, dan pesanan.

**Nama peserta:** Dedi Murphy
**Tanggal kirim:** 21 September 2026

## Stack

| Komponen | Pilihan | Alasan |
|---|---|---|
| Bahasa | Go 1.22+ | — |
| HTTP | Gin v1.10 | `gin.Default()` sudah menyediakan middleware logging & recovery |
| Database | SQLite via `modernc.org/sqlite` | Driver pure Go, tanpa CGO — penguji tidak perlu install database atau compiler C |
| Akses data | `database/sql` (tanpa ORM) | Transaksi dan pengurangan stok terlihat eksplisit |
| Testing | `testing` bawaan | — |

## Cara Menjalankan

```bash
go mod tidy
go run ./cmd/api
```

Server jalan di `http://localhost:8080`. File `order_service.db` dibuat otomatis
dan schema diterapkan saat startup (semua DDL memakai `IF NOT EXISTS`).

### Menjalankan test

```bash
go test ./... -v
```

### Konfigurasi

Lewat environment variable (lihat `.env.example`), tidak ada nilai yang di-hardcode:

| Variable | Default | Keterangan |
|---|---|---|
| `APP_PORT` | `8080` | Port HTTP |
| `DB_PATH` | `order_service.db` | Lokasi file SQLite |

## Struktur Proyek

```
cmd/api/            entry point, wiring dependency
internal/domain/    entity, error aplikasi, state machine status
internal/repository/ akses database (semua SQL di sini)
internal/service/   aturan bisnis
internal/handler/   HTTP: baca request, panggil service, tulis JSON
```

Arah dependensi satu arah: `handler → service → repository → domain`.
Tidak ada aturan bisnis di dalam handler (syarat 1.5 no.3).

## Endpoint

| Method | Endpoint | Keterangan |
|---|---|---|
| POST | `/products` | 201. Validasi: `sku` & `name` wajib, `price > 0`, `stock >= 0` |
| GET | `/products` | `?page=1&limit=10&q=kaos`, response menyertakan metadata total |
| GET | `/products/{id}` | 404 bila tidak ada |
| PUT | `/products/{id}` | Ubah `name`, `price`, `stock` |
| POST | `/customers` | Email divalidasi format & keunikannya |
| POST | `/orders` | Inti studi kasus |
| GET | `/orders/{id}` | Termasuk item, nama produk, harga saat pemesanan, total |
| GET | `/orders` | Filter `?customer_id=` dan `?status=` |
| PATCH | `/orders/{id}/status` | Mengikuti state machine |
| GET | `/health` | Pemeriksaan sederhana, di luar kebutuhan soal |

### Contoh

```bash
curl -X POST localhost:8080/products \
  -H 'Content-Type: application/json' \
  -d '{"sku":"KAOS-01","name":"Kaos Polos","price":50000,"stock":10}'

curl -X POST localhost:8080/customers \
  -H 'Content-Type: application/json' \
  -d '{"name":"Budi","email":"budi@example.com"}'

curl -X POST localhost:8080/orders \
  -H 'Content-Type: application/json' \
  -d '{"customer_id":1,"items":[{"product_id":1,"qty":2}]}'

curl -X PATCH localhost:8080/orders/1/status \
  -H 'Content-Type: application/json' -d '{"status":"PAID"}'
```

## Format Response Error

Satu format dipakai di seluruh endpoint:

```json
{ "error": "insufficient stock", "detail": "product 'KAOS-01' remaining 3, requested 5" }
```

| Status | Kapan |
|---|---|
| 400 | Validasi gagal, stok tidak mencukupi |
| 404 | Produk / customer / order tidak ditemukan |
| 409 | `sku` atau `email` duplikat, transisi status tidak valid |
| 500 | Error tak terduga (detail hanya dicatat di log, tidak dikirim ke client) |

## Keputusan & Asumsi

**Harga disimpan sebagai INTEGER, bukan REAL.** Floating point tidak akurat untuk
uang (`0.1 + 0.2 != 0.3`), dan galatnya menumpuk saat menjumlahkan `total_amount`.
Karena rupiah tidak memiliki pecahan sen, satuan yang dipakai adalah rupiah penuh.

**Produk/customer tidak ditemukan saat membuat pesanan → 404** (soal 1.4 no.8
memperbolehkan 404 atau 400, asal konsisten). Alasannya: `product_id` dan
`customer_id` merujuk resource konkret yang seharusnya ada. Bentuk request-nya
sendiri sudah valid, jadi yang salah bukan format payload melainkan resource
yang dirujuk. 400 disimpan untuk kesalahan bentuk/isi payload dan pelanggaran
aturan bisnis seperti stok kurang.

**Stok kurang → 400, bukan 409.** Soal 1.4 no.3 menyebut 400 secara eksplisit.

**`price_at_order` menyimpan ulang harga.** Harga di tabel `products` bisa berubah
kapan saja. Bila total pesanan dihitung ulang dari `products.price`, tagihan
pesanan lama ikut berubah setiap kali harga diperbarui — persis masalah yang
disebut di latar belakang soal. `price_at_order` adalah snapshot harga pada saat
transaksi terjadi, sehingga pesanan menjadi catatan historis yang tidak berubah.
`GET /orders/{id}` selalu membaca `price_at_order`, bukan harga produk terkini.

**Pengurangan stok memakai satu perintah UPDATE berkondisi:**

```sql
UPDATE products SET stock = stock - ? WHERE id = ? AND stock >= ?
```

Bila `RowsAffected` bernilai 0, berarti stok tidak mencukupi dan transaksi
di-rollback. Pengecekan dan pengurangan terjadi dalam satu operasi atomik di
database, bukan `SELECT` lalu `UPDATE` terpisah, sehingga tidak ada celah waktu
di antara keduanya. Sebagai lapisan terakhir, kolom `stock` juga memiliki
constraint `CHECK (stock >= 0)`.

**Atomicity.** Pembuatan pesanan dan pembatalan pesanan masing-masing dibungkus
satu transaksi (`BeginTx` … `Commit`), dengan `defer tx.Rollback()` sehingga jalur
error mana pun membatalkan seluruh perubahan.

**Koneksi database dibatasi satu (`SetMaxOpenConns(1)`).** SQLite hanya
mengizinkan satu penulis aktif; pembatasan ini mencegah error
`database is locked`. Pada PostgreSQL pembatasan ini tidak diperlukan karena
penguncian dilakukan per baris.

**Deteksi duplikasi UNIQUE** dilakukan dengan memeriksa teks error dari driver
SQLite. Cara ini sederhana dan memadai untuk SQLite; pada PostgreSQL pendekatan
yang lebih tepat adalah memeriksa kode error `23505`.

## Yang Belum Dikerjakan

Bagian WAJIB (1.3, 1.4, 1.5) sudah selesai. Dari bagian BONUS (1.6), yang sudah
ada hanya graceful shutdown dan logging request bawaan Gin.

Belum dikerjakan:

- **Antarmuka React.js** — rencana: satu halaman Vite + React, memanggil
  `GET /products` untuk daftar produk dan `POST /orders` untuk form pemesanan.
- **CI pipeline** — rencana: `.gitlab-ci.yml` dengan dua tahap, `go build ./...`
  dan `go test ./...` pada setiap push.
- **Dockerfile / docker-compose** — rencana: multi-stage build (`golang:1.22`
  untuk compile, `alpine` untuk runtime).
- **Dokumentasi OpenAPI/Swagger** — rencana: Postman collection yang diekspor ke
  dalam repo; saat ini contoh pemakaian tersedia sebagai perintah `curl` di atas.
- **Autentikasi dan rate limiting** — belum ada.
