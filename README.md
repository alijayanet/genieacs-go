# GO-ACS - Go-based Auto Configuration Server

![GO-ACS](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)
![TR-069](https://img.shields.io/badge/Protocol-TR--069-purple?style=for-the-badge)

GO-ACS adalah implementasi ACS (Auto Configuration Server) berbasis Go yang dirancang untuk manajemen ONU (Optical Network Unit) menggunakan protokol TR-069/CWMP. Aplikasi ini menyediakan fitur lengkap untuk monitoring, konfigurasi WiFi, dan manajemen WAN secara remote.

## ✨ Fitur Utama

- **💰 Billing System** - Manajemen paket, tagihan otomatis (invoice), dan laporan keuangan
- **👤 Customer Portal** - Halaman khusus pelanggan untuk cek tagihan dan ganti WiFi
- **🗺️ Coverage Map** - Visualisasi lokasi ONU pelanggan di peta
- **📱 Mobile App API** - API siap pakai untuk integrasi aplikasi Android/iOS (Firebase Ready)
- **💬 Notifications** - Kirim tagihan via WhatsApp (Fonnte) dan Email
- **💳 Online Payment** - Integrasi Tripay untuk pembayaran otomatis
- **🔒 Authentication** - Sistem login admin dan portal pelanggan dengan JWT

## 🏗️ Arsitektur

```
go-acs/
├── cmd/
│   └── server/
│       └── main.go          # Entry point
├── internal/
│   ├── config/              # Konfigurasi aplikasi
│   ├── database/            # SQLite database layer
│   ├── handlers/            # HTTP API handlers
│   ├── models/              # Data models
│   ├── tr069/               # TR-069 CWMP server
│   └── websocket/           # WebSocket hub
├── web/
│   ├── static/              # CSS, JS, images
│   └── templates/           # HTML templates
├── data/                    # SQLite database (auto-created)
├── go.mod
├── go.sum
├── Dockerfile
└── README.md
```

## 🚀 Quick Start

### Prerequisites

- Go 1.21 atau lebih baru
- Git

### Instalasi (Automated)

Jika Anda menggunakan Linux (Ubuntu/Armbian) atau WSL, gunakan script installer otomatis:
```bash
chmod +x install.sh
sudo ./install.sh
```

### Akses Aplikasi

- **Admin Web UI**: [http://localhost:8080](http://localhost:8080)
  - Login: `admin` / `admin123`
- **Customer Portal**: [http://localhost:8080/portal/login](http://localhost:8080/portal/login)
  - Login: Menggunakan **Customer Code** atau **Username** pelanggan
- **TR-069 Endpoint**: `http://localhost:7547` (Gunakan IP Server untuk config di ONU)

### Menggunakan Docker

```bash
# Build image
docker build -t go-acs .

# Run container
docker run -d -p 8080:8080 -p 7547:7547 -v goacs_data:/app/data go-acs
```

## ⚙️ Konfigurasi

Aplikasi dapat dikonfigurasi melalui environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| SERVER_PORT | 8080 | Port untuk Web UI dan API |
| TR069_PORT | 7547 | Port untuk TR-069 endpoint |
| DATABASE_URL | ./data/goacs.db | Path ke file SQLite |
| JWT_SECRET | go-acs-secret... | Secret key untuk JWT |
| ADMIN_USER | admin | Username admin default |
| ADMIN_PASS | admin123 | Password admin default |
| WA_API_KEY | | API Key Fonnte untuk WhatsApp |
| FIREBASE_CREDENTIALS_FILE | firebase-service-account.json | Path file Firebase (JSON) |
| TRIPAY_API_KEY | | API Key Tripay |
| LOG_LEVEL | info | Level logging (debug, info, warn, error) |

## 📡 Konfigurasi ONU

Untuk menghubungkan ONU ke GO-ACS, konfigurasikan ACS URL di ONU:

```
ACS URL: http://<SERVER_IP>:7547/
ACS Username: (kosongkan atau sesuai konfigurasi)
ACS Password: (kosongkan atau sesuai konfigurasi)
```

### Contoh Konfigurasi untuk ZTE F660:
1. Login ke ONU (192.168.1.1)
2. Buka Network > Remote Management > TR069
3. Set ACS URL: `http://SERVER_IP:7547/`
4. Enable TR069

### Contoh Konfigurasi untuk Huawei HG8245H:
1. Login ke ONU (192.168.100.1)
2. Buka System Tools > TR-069 Settings
3. Set ACS URL: `http://SERVER_IP:7547/`
4. Apply changes

## 📚 API Endpoints

### Authentication
- `POST /api/auth/login` - Admin Login
- `POST /api/portal/auth/login` - Customer Portal Login
- `POST /api/auth/logout` - Logout

### Customer Portal (Pelanggan)
- `GET /api/portal/dashboard` - Dashboard data pelanggan
- `GET /api/portal/invoices` - Riwayat tagihan pelanggan
- `PUT /api/portal/wifi/ssid` - Ganti nama WiFi (SSID)
- `PUT /api/portal/wifi/password` - Ganti password WiFi
- `POST /api/customers/{id}/fcm` - Registrasi Push Notification Token (Mobile App)

### Billing & Invoices (Admin)
- `GET /api/invoices` - List semua tagihan
- `POST /api/invoices/generate` - Generate tagihan bulanan otomatis
- `POST /api/invoices/{id}/pay` - Konfirmasi pembayaran manual
- `GET /api/billing/stats` - Statistik keuangan admin

### Devices
- `GET /api/devices` - List semua devices
- `POST /api/devices` - Tambah device baru
- `GET /api/devices/{id}` - Detail device
- `PUT /api/devices/{id}` - Update device
- `DELETE /api/devices/{id}` - Hapus device
- `POST /api/devices/{id}/reboot` - Reboot device
- `POST /api/devices/{id}/refresh` - Refresh parameters

### WiFi Configuration
- `GET /api/devices/{id}/wifi` - Get WiFi config
- `PUT /api/devices/{id}/wifi` - Update WiFi config
- `PUT /api/devices/{id}/wifi/ssid` - Update SSID only
- `PUT /api/devices/{id}/wifi/password` - Update password only

### WAN Configuration
- `GET /api/devices/{id}/wan` - List WAN configs
- `POST /api/devices/{id}/wan` - Create WAN config
- `PUT /api/devices/{id}/wan/{wanId}` - Update WAN config
- `DELETE /api/devices/{id}/wan/{wanId}` - Delete WAN config

### Parameters
- `GET /api/devices/{id}/parameters` - Get all parameters
- `POST /api/devices/{id}/parameters` - Set parameters

### Dashboard
- `GET /api/dashboard/stats` - Dashboard statistics

## 🔧 Development

### Build Binary
```bash
go build -o go-acs cmd/server/main.go
```

### Run Tests
```bash
go test ./...
```

### Build for Linux
```bash
GOOS=linux GOARCH=amd64 go build -o go-acs-linux cmd/server/main.go
```

## 📝 TR-069 Protocol Support

GO-ACS mengimplementasikan protokol TR-069 (CWMP) dengan dukungan untuk:

- **Inform** - Menerima Inform dari CPE
- **GetParameterValues** - Membaca parameter dari CPE
- **SetParameterValues** - Mengatur parameter ke CPE
- **Reboot** - Restart CPE
- **FactoryReset** - Reset ke pengaturan pabrik
- **Download** - Firmware upgrade

### Data Model Support
- **TR-181 (Device:2)** - Data model baru
- **TR-098 (InternetGatewayDevice:1)** - Data model legacy

## 🤝 Contributing

Kontribusi sangat diterima! Silakan buat Pull Request atau Issue untuk perbaikan dan fitur baru.

## 📄 License

MIT License - Lihat file [LICENSE](LICENSE) untuk detail.

## 🙏 Credits

- Terinspirasi oleh [GenieACS](https://genieacs.com/)
- Built with [Go](https://golang.org/)
- UI menggunakan [Font Awesome](https://fontawesome.com/) dan [Inter Font](https://rsms.me/inter/)
