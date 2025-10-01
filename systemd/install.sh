#!/bin/bash

SERVICE_USER="sql-proxy"
SERVICE_GROUP="sql-proxy"
SERVICE_DIR="/opt/sql-proxy"
LOG_DIR="/var/log/sql-proxy"
EXECUTABLE="sql-proxy"
SERVICE_FILE="/etc/systemd/system/sql-proxy.service"

if ! id "$SERVICE_USER" &>/dev/null; then
    useradd --system --no-create-home --shell /bin/false "$SERVICE_USER"
    echo "User $SERVICE_USER created."
else
    echo "User $SERVICE_USER already exists."
fi

mkdir -p "$SERVICE_DIR"
mkdir -p "$LOG_DIR"
chown "$SERVICE_USER:$SERVICE_GROUP" "$SERVICE_DIR"
chown "$SERVICE_USER:$SERVICE_GROUP" "$LOG_DIR"

cp "./$EXECUTABLE" "$SERVICE_DIR/"
chown "$SERVICE_USER:$SERVICE_GROUP" "$SERVICE_DIR/$EXECUTABLE"
chmod +x "$SERVICE_DIR/$EXECUTABLE"

cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=SQL Proxy Service
After=network.target

[Service]
Environment="BIND_ADDR=127.0.0.1"
Environment="BIND_PORT=8080"
Environment="MAX_ROWS=10000"
#Environment="TLS_CERT=/etc/ssl/certs/cert.pem"
#Environment="TLS_KEY=/etc/ssl/private/key.pem"

Type=simple
User=$SERVICE_USER
Group=$SERVICE_GROUP
WorkingDirectory=$SERVICE_DIR
ExecStartPre=-/bin/mkdir -p $LOG_DIR
ExecStartPre=-/bin/chown $SERVICE_USER:$SERVICE_GROUP $LOG_DIR
ExecStart=$SERVICE_DIR/$EXECUTABLE
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable sql-proxy --now

if systemctl is-active --quiet sql-proxy; then
    echo "Service is running. Restarting..."
    systemctl restart sql-proxy
else
    echo "Starting sql-proxy service..."
    systemctl start sql-proxy
fi

echo "SQL Proxy service installed and running."