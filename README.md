# Repositori: agroconnect-service-order
## Microservice Transaksi Pemesanan (ACID) & Otentikasi Pengguna (Golang + MySQL)
**Skema Sertifikasi**: BNSP Full-Stack Developer | **Kandidat**: Fredli Fourqoni

---

### Deskripsi
Layanan Service Order menangani seluruh proses transaksi bisnis pemesanan komoditas pertanian dan autentikasi pengguna menggunakan basis data relasional **MySQL** yang menjamin kepatuhan prinsip ACID (*Atomicity, Consistency, Isolation, Durability*).

### Fitur Utama:
1. **Otentikasi Pengguna**: Registrasi akun baru dengan hash password Bcrypt dan Login penghasil token JWT bertanda tangan HMAC-SHA256.
2. **Checkout Transaksi ACID**: Membuat transaksi order dan rincian item belanja dalam satu blok transaksi basis data terisolasi.
3. **Penerbitan Invoice**: Pembuatan kode unik transaksi berformat `ORD-2026-XXXX`.
4. **Riwayat Pesanan**: Melihat daftar transaksi belanja pengguna beserta status pengiriman.
5. **Health Check**: Endpoint `/health` untuk monitoring ketersediaan kontainer.

### Port
- Port Eksternal/Internal: `8082`
