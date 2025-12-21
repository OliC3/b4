#!/bin/sh
# Create systemd service file (for systems with systemd)
create_systemd_service() {
    # Only create if systemd is actually available and functioning
    if ! [ -d "/etc/systemd/system" ] || ! command_exists systemctl; then
        return
    fi

    # Check if systemd is actually running (not just installed)
    if ! systemctl list-units >/dev/null 2>&1; then
        return
    fi

    cat >"/etc/systemd/system/b4.service" <<EOF
[Unit]
Description=B4 DPI Bypass Service
After=network.target

[Service]
Type=simple
User=root
ExecStart=${INSTALL_DIR}/${BINARY_NAME} --config ${CONFIG_FILE}
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    print_success "Systemd service created. You can manage it with:"
    print_info "  systemctl start b4"
    print_info "  systemctl stop b4"
    print_info "  systemctl enable b4  # To start on boot"

    SYSTEMCTL_CREATED="1"
}

# Create OpenWRT/Entware init script
create_sysv_service() {
    # Determine the correct init.d directory
    INIT_DIR=""

    # Check for Entware/MerlinWRT (uses /opt/etc/init.d)
    if [ -d "/opt/etc/init.d" ] && [ -w "/opt/etc/init.d" ]; then
        INIT_DIR="/opt/etc/init.d"
        print_info "Detected Entware/MerlinWRT system"
    # Check for standard OpenWRT (uses /etc/init.d if writable)
    elif [ -d "/etc/init.d" ] && [ -w "/etc/init.d" ]; then
        INIT_DIR="/etc/init.d"
    # Fallback: try to create /opt/etc/init.d if it doesn't exist
    elif [ -d "/opt/etc" ]; then
        mkdir -p /opt/etc/init.d 2>/dev/null && INIT_DIR="/opt/etc/init.d"
    fi

    # Only proceed if we found a writable init directory
    if [ -n "$INIT_DIR" ]; then
        print_info "Creating init script in $INIT_DIR..."

        if [ "$INIT_DIR" = "/etc/init.d" ]; then
            INIT_SCRIPT_NAME="b4"
        else
            INIT_SCRIPT_NAME="S99b4"
        fi

        INIT_FULL_PATH="${INIT_DIR}/${INIT_SCRIPT_NAME}"

        rm -f "${INIT_DIR}/S99b4" 2>/dev/null || true # remove legacy script

        if [ -f "${INIT_DIR}/rc.func" ]; then
            print_info "rc.func found in $INIT_DIR, using it for init script"

            cat >"${INIT_FULL_PATH}" <<'EOF'
#!/bin/sh

# B4 DPI Bypass Service Init Script

ENABLED=yes
PROCS=b4
ARGS="--config=CONFIG_PLACEHOLDER"
PREARGS="nohup"
DESC="$PROCS"
PATH=/opt/sbin:/opt/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin


kernel_mod_load() {
	KERNEL=$(uname -r)

	connbytes_mod_path=$(find /lib/modules/$(uname -r) -name "xt_connbytes.ko*")
	if [ ! -z "$connbytes_mod_path" ]; then
		insmod "$connbytes_mod_path" >/dev/null 2>&1 && echo "xt_connbytes.ko loaded"
	fi

	nfqueue_mod_path=$(find /lib/modules/$(uname -r) -name "xt_NFQUEUE.ko*")
	if [ ! -z "$nfqueue_mod_path" ]; then
		insmod "$nfqueue_mod_path" >/dev/null 2>&1 && echo "xt_NFQUEUE.ko loaded"
	fi

	(modprobe xt_connbytes >/dev/null 2>&1 && echo "xt_connbytes loaded") || true
	(modprobe xt_NFQUEUE >/dev/null 2>&1 && echo "xt_NFQUEUE loaded") || true
}

if [ "$1" = "start" ] || [ "$1" = "restart" ]
then
    kernel_mod_load
fi

. /opt/etc/init.d/rc.func

EOF

        else
            print_info "Creating standard init.d script"
            cat >"${INIT_FULL_PATH}" <<'EOF'
#!/bin/sh

# B4 DPI Bypass Service Init Script
PROG=PROG_PLACEHOLDER
CONFIG_FILE=CONFIG_PLACEHOLDER
PIDFILE=/var/run/b4.pid

start() {
    echo "Starting b4..."
    if [ -f "$PIDFILE" ] && kill -0 $(cat "$PIDFILE") 2>/dev/null; then
        echo "b4 is already running"
        return 1
    fi

    kernel_mod_load

    nohup $PROG --config $CONFIG_FILE > /var/log/b4.log 2>&1 &
    echo $! > "$PIDFILE"
    sleep 1
    # Verify it's actually running
    if kill -0 $(cat "$PIDFILE") 2>/dev/null; then
        echo "b4 started (PID: $(cat "$PIDFILE"))"
    else
        echo "Warning: b4 may have failed to start, check /var/log/b4.log"
        rm -f "$PIDFILE"
        return 1
    fi
}

stop() {
    echo "Stopping b4..."
    if [ -f "$PIDFILE" ]; then
        kill $(cat "$PIDFILE") 2>/dev/null
        rm -f "$PIDFILE"
        echo "b4 stopped"
    else
        echo "b4 is not running"
    fi
}

kernel_mod_load() {
	KERNEL=$(uname -r)

	connbytes_mod_path=$(find /lib/modules/$(uname -r) -name "xt_connbytes.ko*")
	if [ ! -z "$connbytes_mod_path" ]; then
		insmod "$connbytes_mod_path" >/dev/null 2>&1 && echo "xt_connbytes.ko loaded"
	fi

	nfqueue_mod_path=$(find /lib/modules/$(uname -r) -name "xt_NFQUEUE.ko*")
	if [ ! -z "$nfqueue_mod_path" ]; then
		insmod "$nfqueue_mod_path" >/dev/null 2>&1 && echo "xt_NFQUEUE.ko loaded"
	fi

	(modprobe xt_connbytes>/dev/null 2>&1 && echo "xt_connbytes loaded") || true
	(modprobe xt_NFQUEUE >/dev/null 2>&1 && echo "xt_NFQUEUE loaded") || true
}


case "$1" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    restart)
        stop
        sleep 1
        start
        ;;
    *)
        echo "Usage: $0 {start|stop|restart}"
        exit 1
        ;;
esac
EOF

        fi

        sed "s|PROG_PLACEHOLDER|${INSTALL_DIR}/${BINARY_NAME}|g; s|CONFIG_PLACEHOLDER|${CONFIG_FILE}|g" "${INIT_FULL_PATH}" >"${INIT_FULL_PATH}.tmp"
        mv "${INIT_FULL_PATH}.tmp" "${INIT_FULL_PATH}"
        chmod +x "${INIT_FULL_PATH}"

        print_success "Init script created at ${INIT_FULL_PATH}"
        print_info "You can manage it with:"
        print_info "  ${INIT_FULL_PATH} start"
        print_info "  ${INIT_FULL_PATH} stop"
        print_info "  ${INIT_FULL_PATH} restart"

    else
        print_warning "Could not create init script - no writable init directory found"
    fi
}
