remove_b4() {
    echo ""
    echo "======================================="
    echo "     B4 Uninstaller"
    echo "======================================="
    echo ""

    # Detect system to get proper paths
    set_system_paths

    # Stop the service first
    print_info "Stopping b4 service if running..."

    # Check systemd service FIRST
    if [ -f "/etc/systemd/system/b4.service" ] && command_exists systemctl; then
        systemctl stop b4 2>/dev/null || true
        systemctl disable b4 2>/dev/null || true
        print_info "Stopped systemd service"
    fi

    # Check Entware init script
    if [ -f "/opt/etc/init.d/S99b4" ]; then
        /opt/etc/init.d/S99b4 stop 2>/dev/null || true
        print_info "Stopped Entware service"
    fi

    # Check standard init script
    if [ -f "/etc/init.d/b4" ]; then
        /etc/init.d/b4 stop 2>/dev/null || true
        print_info "Stopped init service"
    fi

    # Kill any remaining b4 processes
    if ps 2>/dev/null | grep -v grep | grep -v "b4install" | grep -q "b4$\|b4[[:space:]]"; then
        print_info "Killing remaining b4 processes..."
        ps | grep -v grep | grep -v "b4install" | grep "b4$\|b4[[:space:]]" | awk '{print $1}' | while read pid; do
            if [ -n "$pid" ]; then
                kill "$pid" 2>/dev/null || true
            fi
        done
        sleep 1
    fi

    # Remove binary from all possible locations
    POSSIBLE_DIRS="/opt/sbin /usr/local/bin /usr/bin /usr/sbin"
    for dir in $POSSIBLE_DIRS; do
        if [ -f "$dir/$BINARY_NAME" ]; then
            print_info "Removing binary: $dir/$BINARY_NAME"
            rm -f "$dir/$BINARY_NAME"
            print_success "Binary removed from $dir"

            # Remove any backup files
            rm -f "$dir/"${BINARY_NAME}.backup.* 2>/dev/null || true

        fi
    done

    # Remove service files
    if [ -f "/etc/systemd/system/b4.service" ]; then
        print_info "Removing systemd service..."
        rm -f "/etc/systemd/system/b4.service"
        if command_exists systemctl; then
            systemctl daemon-reload 2>/dev/null || true
        fi
        print_success "Systemd service removed"
    fi

    if [ -f "/opt/etc/init.d/S99b4" ]; then
        print_info "Removing Entware init script..."
        rm -f "/opt/etc/init.d/S99b4"
        print_success "Entware init script removed"
    fi

    if [ -f "/etc/init.d/b4" ]; then
        print_info "Removing init script..."
        rm -f "/etc/init.d/b4"
        print_success "Init script removed"
    fi

    # Remove symlinks
    if [ -L "/usr/bin/${BINARY_NAME}" ]; then
        print_info "Removing symlink: /usr/bin/${BINARY_NAME}"
        rm -f "/usr/bin/${BINARY_NAME}"
    fi

    # Ask about configuration ONCE
    printf "${CYAN}Remove configuration files as well? (y/N): ${NC}"
    read answer
    case "$answer" in
    [yY] | [yY][eE][sS])
        print_info "Removing configuration directory: $CONFIG_DIR"
        rm -rf "$CONFIG_DIR"
        print_success "Configuration removed"
        ;;
    *)
        print_info "Configuration preserved at: $CONFIG_DIR"
        ;;
    esac

    # Remove log files
    rm -f /var/log/b4.log 2>/dev/null || true
    rm -f /var/run/b4.pid 2>/dev/null || true

    echo ""
    print_success "B4 has been uninstalled successfully!"
    echo ""

    exit 0
}
