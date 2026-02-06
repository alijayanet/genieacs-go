#!/bin/bash
set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=================================================${NC}"
echo -e "${BLUE}      GO-ACS Installer for Ubuntu/Armbian       ${NC}"
echo -e "${BLUE}=================================================${NC}"

# Check root priv
if [ "$EUID" -ne 0 ]; then
  echo "Please run as root (sudo ./install.sh)"
  exit 1
fi

DEST_DIR="/opt/go-acs"
SERVICE_FILE="/etc/systemd/system/go-acs.service"

echo -e "${GREEN}[1/8] Cleaning up old ACS installations...${NC}"
# Stop and disable any old ACS services that might conflict with port 7547
if systemctl is-active --quiet acs 2>/dev/null; then
    echo -e "${BLUE}  → Stopping old 'acs' service...${NC}"
    systemctl stop acs 2>/dev/null || true
    systemctl disable acs 2>/dev/null || true
fi

if systemctl is-active --quiet genieacs-cwmp 2>/dev/null; then
    echo -e "${BLUE}  → Stopping GenieACS service...${NC}"
    systemctl stop genieacs-cwmp 2>/dev/null || true
    systemctl disable genieacs-cwmp 2>/dev/null || true
fi

# Kill any process using port 7547
if ss -tlnp | grep -q ":7547"; then
    echo -e "${BLUE}  → Killing processes using port 7547...${NC}"
    # Kill processes from /opt/acs/
    pkill -9 -f "/opt/acs/acs" 2>/dev/null || true
    pkill -9 -f "genieacs" 2>/dev/null || true
    # Wait a moment for port to be released
    sleep 2
fi

# Remove old ACS service files if they exist
if [ -f "/etc/systemd/system/acs.service" ]; then
    echo -e "${BLUE}  → Removing old acs.service...${NC}"
    systemctl stop acs 2>/dev/null || true
    systemctl disable acs 2>/dev/null || true
    rm -f /etc/systemd/system/acs.service
    systemctl daemon-reload
fi

echo -e "${GREEN}  ✓ Cleanup complete${NC}"

echo -e "${GREEN}[2/8] Updating package lists...${NC}"
apt-get update || echo -e "${RED}Warning: Some repositories failed to update, but trying to continue...${NC}"

echo -e "${GREEN}[3/8] Installing dependencies (GCC, SQLite3, Git, OpenSSL)...${NC}"
apt-get install -y gcc sqlite3 git curl openssl wget

# Check if Go is installed and version is sufficient
GO_VERSION_REQUIRED="1.21"
GO_INSTALLED=false

if command -v go &> /dev/null; then
    CURRENT_GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    if [ "$(printf '%s\n' "$GO_VERSION_REQUIRED" "$CURRENT_GO_VERSION" | sort -V | head -n1)" = "$GO_VERSION_REQUIRED" ]; then
        GO_INSTALLED=true
        echo -e "${GREEN}Go $CURRENT_GO_VERSION is already installed${NC}"
    else
        echo -e "${BLUE}Current Go version ($CURRENT_GO_VERSION) is too old${NC}"
    fi
fi

if [ "$GO_INSTALLED" = false ]; then
    echo -e "${GREEN}Installing Go 1.23.5...${NC}"
    
    # Detect architecture
    ARCH=$(uname -m)
    if [ "$ARCH" = "x86_64" ]; then
        GO_ARCH="amd64"
    elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
        GO_ARCH="arm64"
    elif [ "$ARCH" = "armv7l" ]; then
        GO_ARCH="armv6l"
    else
        echo "Unsupported architecture: $ARCH"
        exit 1
    fi
    
    GO_VERSION="1.23.5"
    GO_TAR="go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    
    # Download Go
    wget -q --show-progress "https://go.dev/dl/${GO_TAR}"
    
    # Remove old Go installation if exists
    rm -rf /usr/local/go
    
    # Extract and install
    tar -C /usr/local -xzf "$GO_TAR"
    rm "$GO_TAR"
    
    # Add Go to PATH if not already there
    if ! grep -q "/usr/local/go/bin" /etc/profile; then
        echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    fi
    
    # Add to current session
    export PATH=/usr/local/go/bin:$PATH
    
    # Verify installation
    go version
    echo -e "${GREEN}Go installed successfully!${NC}"
fi

echo -e "${GREEN}[4/8] Building application...${NC}"
# Check if go.mod exists
if [ ! -f "go.mod" ]; then
    echo "Error: go.mod not found. Please run this script from the project root."
    exit 1
fi

# Determine which Go binary to use
if [ -f "/usr/local/go/bin/go" ]; then
    GO_BIN="/usr/local/go/bin/go"
elif command -v go &> /dev/null; then
    GO_BIN="go"
else
    echo "Error: Go not found."
    exit 1
fi

echo -e "${GREEN}Using Go: $($GO_BIN version)${NC}"

# Set Go Proxy for faster downloads (especially useful if main proxy is slow)
export GOPROXY=https://goproxy.io,direct
export PATH=/usr/local/go/bin:$PATH

echo -e "${GREEN}Downloading dependencies...${NC}"
$GO_BIN mod download

# Build binary
echo -e "${GREEN}Compiling application (this may take 2-5 minutes)...${NC}"
$GO_BIN build -v -p 1 -o go-acs-bin cmd/server/main.go
if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

echo -e "${GREEN}[5/8] Setting up directory structure at $DEST_DIR...${NC}"
mkdir -p "$DEST_DIR"
mkdir -p "$DEST_DIR/data"
mkdir -p "$DEST_DIR/web"

echo -e "${GREEN}[6/8] Copying files...${NC}"
# Stop service if running to prevent "Text file busy" error
systemctl stop go-acs 2>/dev/null || true
cp -f go-acs-bin "$DEST_DIR/go-acs"
cp -r web/* "$DEST_DIR/web/"

# Environment setup
if [ ! -f "$DEST_DIR/.env" ]; then
    echo "Creating default configuration..."
    # Generate random secret
    SECRET=$(openssl rand -hex 32)
    
    cat <<EOF > "$DEST_DIR/.env"
SERVER_PORT=8080
TR069_PORT=7547
DATABASE_URL=./data/goacs.db
JWT_SECRET=$SECRET
LOG_LEVEL=info
AUTH_ENABLED=true
ADMIN_USER=admin
ADMIN_PASS=admin123

# MikroTik Config (Optional)
# MIKROTIK_HOST=192.168.88.1
# MIKROTIK_USER=admin
# MIKROTIK_PASS=
# MIKROTIK_PORT=8728

# Push Notification (Firebase)
FIREBASE_CREDENTIALS_FILE=firebase-service-account.json

# WhatsApp Gateway (Fonnte)
WA_PROVIDER_URL=https://api.fonnte.com/send
WA_API_KEY=

# Payment Gateway (Tripay)
TRIPAY_API_KEY=
TRIPAY_PRIVATE_KEY=
TRIPAY_MERCHANT_CODE=
TRIPAY_MODE=sandbox
EOF
fi

# Set permissions
chmod +x "$DEST_DIR/go-acs"

echo -e "${GREEN}[7/8] Creating Systemd Service...${NC}"
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

echo -e "${GREEN}[8/8] Starting GO-ACS Service...${NC}"
systemctl daemon-reload
systemctl enable go-acs
systemctl restart go-acs

# Wait for service to start
sleep 3

# Verify port 7547 is used by go-acs
if ss -tlnp | grep ":7547" | grep -q "go-acs"; then
    echo -e "${GREEN}  ✓ TR-069 server running on port 7547${NC}"
else
    echo -e "${BLUE}  ⚠ Warning: Checking TR-069 port...${NC}"
    # Try to identify what's using port 7547
    PORT_USER=$(ss -tlnp | grep ":7547" | head -1)
    if [ -n "$PORT_USER" ]; then
        echo -e "${BLUE}  Port 7547 is in use by: $PORT_USER${NC}"
        echo -e "${BLUE}  Attempting to fix...${NC}"
        pkill -9 -f "/opt/acs/acs" 2>/dev/null || true
        sleep 2
        systemctl restart go-acs
        sleep 2
    fi
fi

echo -e "${BLUE}=================================================${NC}"
echo -e "${GREEN}Installation Complete!${NC}"
echo -e "Web Interface: http://$(hostname -I | awk '{print $1}'):8080"
echo -e "TR-069 URL:    http://$(hostname -I | awk '{print $1}'):7547"
echo -e "Admin User:    admin"
echo -e "Admin Pass:    admin123"
echo -e ""
echo -e "Control with:  systemctl status go-acs"
echo -e "Logs:          journalctl -u go-acs -f"
echo -e "${BLUE}=================================================${NC}"

