#!/bin/bash
# Skrip Deployment GO-ACS Portabel (Cukup jalankan di folder ekstraksi)
set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}=================================================${NC}"
echo -e "${BLUE}      GO-ACS Portable Deployer (No-Build)      ${NC}"
echo -e "${BLUE}=================================================${NC}"

if [ "$EUID" -ne 0 ]; then
  echo -e "${RED}Mohon jalankan sebagai root (sudo ./deploy.sh)${NC}"
  exit 1
fi

DEST_DIR="/opt/go-acs"
SERVICE_FILE="/etc/systemd/system/go-acs.service"

# 1. Cek ketersediaan file di folder saat ini
echo -e "${GREEN}[1/5] Memeriksa file paket...${NC}"
for file in go-acs web .env; do
    if [ ! -e "$file" ]; then
        echo -e "${RED}Error: File '$file' tidak ditemukan di folder ini!${NC}"
        echo "Pastikan Anda menjalankan skrip ini di dalam folder hasil ekstraksi."
        exit 1
    fi
done

# 2. Siapkan folder tujuan
echo -e "${GREEN}[2/5] Menyiapkan folder tujuan di $DEST_DIR...${NC}"
systemctl stop go-acs 2>/dev/null || true
mkdir -p "$DEST_DIR"
mkdir -p "$DEST_DIR/data"

# 3. Salin file
echo -e "${GREEN}[3/5] Mendistribusikan file ke $DEST_DIR...${NC}"
cp -f go-acs "$DEST_DIR/"
cp -rf web "$DEST_DIR/"
# Jangan overwrite .env jika sudah ada di server baru (agar config tidak hilang)
if [ ! -f "$DEST_DIR/.env" ]; then
    cp .env "$DEST_DIR/"
fi

# 4. Instal dependensi dasar
echo -e "${GREEN}[4/5] Menginstal dependensi sistem...${NC}"
apt-get update && apt-get install -y sqlite3 openssl ca-certificates tzdata

# 5. Atur izin dan service
echo -e "${GREEN}[5/5] Konfigurasi Systemd Service...${NC}"
chmod +x "$DEST_DIR/go-acs"

cat <<EOF > "$SERVICE_FILE"
[Unit]
Description=GO-ACS TR-069 Server
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$DEST_DIR
ExecStart=$DEST_DIR/go-acs
Restart=always
RestartSec=5
EnvironmentFile=-$DEST_DIR/.env
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

echo -e "${GREEN}Mengaktifkan dan Menjalankan Service...${NC}"
systemctl daemon-reload
systemctl enable go-acs
systemctl restart go-acs

echo -e "${BLUE}=================================================${NC}"
echo -e "${GREEN}BERHASIL! GO-ACS siap digunakan di server baru.${NC}"
echo -e "Web UI: http://$(hostname -I | awk '{print $1}'):8080"
echo -e "${BLUE}=================================================${NC}"
