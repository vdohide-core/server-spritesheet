#!/bin/bash

# Server Spritesheet Installation Script
# Usage: curl -fsSL https://raw.githubusercontent.com/vdohide-core/server-spritesheet/main/install.sh | sudo -E bash -s -- [OPTIONS]

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

WORKER_COUNT=1
UNINSTALL=false
MONGODB_URI=""
STORAGE_ID=""
STORAGE_PATH="/home/files"

APP_NAME="server-spritesheet"
APP_DIR="/opt/$APP_NAME"
SERVICE_NAME="server-spritesheet"
GITHUB_REPO="vdohide-core/server-spritesheet"
RELEASES_URL="https://github.com/$GITHUB_REPO/releases/latest/download"

print_status()  { echo -e "${GREEN}[INFO]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_error()   { echo -e "${RED}[ERROR]${NC} $1"; }

while [[ $# -gt 0 ]]; do
    case $1 in
        --uninstall)       UNINSTALL=true; shift ;;
        --count|-w)        WORKER_COUNT="$2"; shift 2 ;;
        --mongodb-uri)     MONGODB_URI="$2"; shift 2 ;;
        --storage-id)      STORAGE_ID="$2"; shift 2 ;;
        --storage-path)    STORAGE_PATH="$2"; shift 2 ;;
        -h|--help)
            echo "Server Spritesheet Installer"
            echo ""
            echo "Options:"
            echo "  --uninstall          Uninstall completely"
            echo "  --count NUM          Number of worker instances (default: 1)"
            echo "  --mongodb-uri URI    MongoDB connection string"
            echo "  --storage-id ID      Storage ID when co-located with files"
            echo "  --storage-path DIR   Local files path (default: /home/files)"
            exit 0 ;;
        *) print_error "Unknown option: $1"; exit 1 ;;
    esac
done

if [ "$UNINSTALL" = true ]; then
    print_warning "Uninstalling..."
    for i in $(seq 1 20); do
        systemctl stop "${SERVICE_NAME}@${i}"    2>/dev/null || true
        systemctl disable "${SERVICE_NAME}@${i}" 2>/dev/null || true
    done
    [ -f "/etc/systemd/system/${SERVICE_NAME}@.service" ] && rm "/etc/systemd/system/${SERVICE_NAME}@.service"
    systemctl daemon-reload
    [ -d "$APP_DIR" ] && rm -rf "$APP_DIR"
    print_status "Uninstalled."
    exit 0
fi

if [ "$(id -u)" -ne 0 ]; then
    print_error "Run as root (sudo)"
    exit 1
fi

print_status "Installing server-spritesheet (workers: $WORKER_COUNT)..."

print_status "Installing dependencies (curl, jq, ffmpeg)..."
if command -v apt-get &>/dev/null; then
    apt-get update -qq
    apt-get install -y -qq curl jq ffmpeg
elif command -v yum &>/dev/null; then
    yum install -y curl jq ffmpeg
elif command -v dnf &>/dev/null; then
    dnf install -y curl jq ffmpeg
fi

for cmd in curl jq ffmpeg; do
    command -v $cmd &>/dev/null || { print_error "$cmd not found"; exit 1; }
done

systemctl stop ${SERVICE_NAME}@* 2>/dev/null || true
mkdir -p "$APP_DIR"
cd "$APP_DIR"

ARCH=$(uname -m)
case "$ARCH" in
    x86_64)  BINARY="linux" ;;
    aarch64) BINARY="linux-arm64" ;;
    *) print_error "Unsupported arch: $ARCH"; exit 1 ;;
esac

print_status "Downloading binary ($BINARY)..."
curl -fsSL "$RELEASES_URL/$BINARY" -o "$APP_DIR/$APP_NAME"
chmod +x "$APP_DIR/$APP_NAME"

print_status "Creating .env..."
cat > "$APP_DIR/.env" <<EOF
MONGODB_URI=$MONGODB_URI
STORAGE_ID=$STORAGE_ID
STORAGE_PATH=$STORAGE_PATH
PORT=8084
EOF

print_status "Creating systemd service..."
cat > /etc/systemd/system/${SERVICE_NAME}@.service <<EOF
[Unit]
Description=Server Spritesheet Worker %i
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$APP_DIR
ExecStart=$APP_DIR/$APP_NAME
Restart=always
RestartSec=5
EnvironmentFile=$APP_DIR/.env
Environment="WORKER_ID=spritesheet_$(hostname)@%i"

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
for i in $(seq 1 $WORKER_COUNT); do
    systemctl enable ${SERVICE_NAME}@$i
    systemctl start  ${SERVICE_NAME}@$i
done

sleep 2
RUNNING=0
for i in $(seq 1 $WORKER_COUNT); do
    systemctl is-active --quiet ${SERVICE_NAME}@$i && RUNNING=$((RUNNING+1))
done

echo ""
echo "============================================"
if [ $RUNNING -eq $WORKER_COUNT ]; then
    print_status "Installation complete ($RUNNING/$WORKER_COUNT workers)"
else
    print_warning "$RUNNING/$WORKER_COUNT workers running"
    journalctl -u "${SERVICE_NAME}@1" -n 15 --no-pager
fi
echo "  Directory: $APP_DIR"
echo "  Enable:    db.settings.spritesheet_enabled = true"
echo "  Logs:      journalctl -u \"${SERVICE_NAME}@*\" -f"
echo "============================================"
