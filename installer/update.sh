# Perform update - stops service, updates, and restarts
perform_update() {
    QUIET_MODE=1 # Force quiet mode during updates

    echo ""
    echo "======================================="
    echo "     B4 Update Process"
    echo "======================================="
    echo ""

    # Find existing installation
    FOUND_BINARY=""
    for dir in /opt/sbin /usr/local/bin /usr/bin /usr/sbin; do
        if [ -f "$dir/$BINARY_NAME" ]; then
            FOUND_BINARY="$dir/$BINARY_NAME"
            INSTALL_DIR="$dir"
            break
        fi
    done

    if [ -z "$FOUND_BINARY" ]; then
        print_error "B4 binary not found. Please install first."
        exit 1
    fi

    # Find existing config
    for dir in /opt/etc/b4 /etc/b4; do
        if [ -f "$dir/b4.json" ]; then
            CONFIG_DIR="$dir"
            CONFIG_FILE="$dir/b4.json"
            break
        fi
    done

    print_info "Found existing installation at: $INSTALL_DIR"
    print_info "Using config at: $CONFIG_DIR"

    # Detect service manager
    SERVICE_MANAGER=""
    RESTART_CMD=""

    if [ -f "/etc/systemd/system/b4.service" ] && command_exists systemctl; then
        SERVICE_MANAGER="systemd"
        RESTART_CMD="systemctl restart b4"
    elif [ -f "/opt/etc/init.d/S99b4" ]; then
        SERVICE_MANAGER="entware"
        RESTART_CMD="/opt/etc/init.d/S99b4 restart"
    elif [ -f "/etc/init.d/b4" ]; then
        SERVICE_MANAGER="init"
        RESTART_CMD="/etc/init.d/b4 restart"
    else
        print_error "No service manager detected. Cannot perform automatic update."
        exit 1
    fi

    print_info "Detected service manager: $SERVICE_MANAGER"

    # Extract geosite settings from config if available
    GEOSITE_SRC=""
    GEOSITE_DST=""
    if [ -f "$CONFIG_FILE" ] && command_exists jq; then
        GEOSITE_SRC=$(jq -r '.system.geo.geosite_url // empty' "$CONFIG_FILE" 2>/dev/null)
        geosite_path=$(jq -r '.system.geo.geosite_path // empty' "$CONFIG_FILE" 2>/dev/null)
        if [ -n "$geosite_path" ] && [ "$geosite_path" != "null" ]; then
            GEOSITE_DST=$(dirname "$geosite_path")
        fi
    fi

    # Stop the service
    print_info "Stopping b4 service..."
    case "$SERVICE_MANAGER" in
    systemd)
        systemctl stop b4 2>/dev/null || true
        ;;
    entware)
        /opt/etc/init.d/S99b4 stop 2>/dev/null || true
        ;;
    init)
        /etc/init.d/b4 stop 2>/dev/null || true
        ;;
    esac

    sleep 2
    print_success "Service stopped"

    # Perform the update (call main_install with quiet mode)
    print_info "Installing latest version..."
    main_install "$@"

    # Start the service
    print_info "Starting b4 service..."
    case "$SERVICE_MANAGER" in
    systemd)
        systemctl start b4
        ;;
    entware)
        /opt/etc/init.d/S99b4 start
        ;;
    init)
        /etc/init.d/b4 start
        ;;
    esac

    sleep 2

    # Verify service is running
    service_running=0
    case "$SERVICE_MANAGER" in
    systemd) systemctl is-active --quiet b4 && service_running=1 ;;
    entware | init) ps | grep -v grep | grep -q "b4$\|b4[[:space:]]" && service_running=1 ;;
    esac

    if [ "$service_running" = "1" ]; then
        print_success "Service started successfully"
        echo ""
        echo "======================================="
        echo "     Update Complete!"
        echo "======================================="
        echo ""
    else
        print_error "Service may not have started correctly"
        print_info "Check logs: journalctl -u b4 -f (systemd) or /var/log/b4.log"
        exit 1
    fi
}
