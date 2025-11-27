# Check kernel module status
check_kernel_module() {
    module_name="$1"

    # Check if module is loaded
    if lsmod 2>/dev/null | grep -q "^$module_name"; then
        echo "loaded"
        return 0
    fi

    # Skip filesystem check on routers - often hangs
    echo "unknown"
    return 0
}

# Get service status
get_service_status() {
    # Check if b4 process is running
    if ps 2>/dev/null | grep -v grep | grep -v "b4install" | grep -q "b4$\|b4[[:space:]]"; then
        echo "running"
        return 0
    fi

    # Check systemd service
    if [ -f "/etc/systemd/system/b4.service" ] && command_exists systemctl >/dev/null 2>&1; then
        if systemctl is-active --quiet b4 2>/dev/null; then
            echo "running (systemd)"
            return 0
        else
            echo "stopped (systemd)"
            return 0
        fi
    fi

    # Check Entware init
    if [ -f "/opt/etc/init.d/S99b4" ]; then
        echo "configured (entware)"
        return 0
    fi

    # Check standard init
    if [ -f "/etc/init.d/b4" ]; then
        echo "configured (init.d)"
        return 0
    fi

    echo "not installed"
    return 0
}

# Get network interfaces info
get_network_info() {
    # Get primary IP
    primary_ip=""
    if command_exists ip; then
        primary_ip=$(ip -4 route get 1 2>/dev/null | grep -oP 'src \K\S+' | head -1 || true)
    elif command_exists ifconfig; then
        primary_ip=$(ifconfig 2>/dev/null | grep 'inet addr:' | grep -v '127.0.0.1' | head -1 | awk '{print $2}' | cut -d':' -f2 || true)
    fi

    echo "$primary_ip"
}

# Detect firewall backend
detect_firewall_backend() {
    # Check if nft exists and has rules
    if which nft >/dev/null 2>&1; then
        out=$(nft list tables 2>/dev/null || true)
        if [ -n "$out" ]; then
            echo "nftables"
            return 0
        fi
    fi

    # Check for iptables
    if which iptables >/dev/null 2>&1; then
        # Check if iptables-legacy or iptables-nft
        out=$(iptables --version 2>/dev/null || true)
        if echo "$out" | grep -q "nf_tables"; then
            echo "iptables-nft"
        else
            echo "iptables-legacy"
        fi
        return 0
    fi

    echo "none"
    return 0
}

# System information display function
show_system_info() {

    set_system_paths

    echo ""
    echo "======================================="
    echo "       B4 System Information"
    echo "======================================="

    # System Information
    print_header "System Information"

    # OS Detection
    os_type="Unknown"
    if [ -f /etc/openwrt_release ]; then
        os_type="OpenWRT"
        os_version=$(grep 'DISTRIB_RELEASE' /etc/openwrt_release | cut -d'=' -f2 | tr -d "'\"" || true)
    elif [ -f /etc/merlinwrt_release ]; then
        os_type="MerlinWRT"
        os_version=$(cat /etc/merlinwrt_release 2>/dev/null || true)
    elif [ -f /etc/entware_release ]; then
        os_type="Entware"
        os_version=$(cat /etc/entware_release 2>/dev/null || true)
    elif [ -f /etc/os-release ]; then
        os_type=$(grep '^NAME=' /etc/os-release | cut -d'=' -f2 | tr -d '"' || echo "Linux")
        os_version=$(grep '^VERSION=' /etc/os-release | cut -d'=' -f2 | tr -d '"' || true)
    else
        os_type="Linux"
    fi

    print_detail "Operating System" "$os_type ${os_version}"
    print_detail "Kernel Version" "$(uname -r)"
    print_detail "Architecture (raw)" "$(uname -m)"
    print_detail "Architecture (b4)" "$(detect_architecture)"
    print_detail "Hostname" "$(hostname 2>/dev/null || echo 'unknown')"

    # CPU Info
    if [ -f /proc/cpuinfo ]; then
        cpu_model=$(grep 'model name' /proc/cpuinfo 2>/dev/null | head -1 | cut -d':' -f2 | sed 's/^ *//' || true)
        if [ -z "$cpu_model" ]; then
            cpu_model=$(grep 'Processor' /proc/cpuinfo 2>/dev/null | head -1 | cut -d':' -f2 | sed 's/^ *//' || true)
        fi
        if [ -n "$cpu_model" ]; then
            print_detail "CPU Model" "$cpu_model"
        fi

        cpu_cores=$(grep -c '^processor' /proc/cpuinfo 2>/dev/null || echo "1")
        print_detail "CPU Cores" "$cpu_cores"
    fi

    # Memory Info
    if [ -f /proc/meminfo ]; then
        mem_total=$(grep '^MemTotal:' /proc/meminfo | awk '{printf "%.0f MB", $2/1024}')
        mem_free=$(grep '^MemFree:' /proc/meminfo | awk '{printf "%.0f MB", $2/1024}')
        print_detail "Memory" "$mem_total (Free: $mem_free)"
    fi

    # Network Info
    # primary_ip=$(get_network_info)
    # if [ -n "$primary_ip" ]; then
    #     print_detail "Primary IP" "$primary_ip"
    # fi

    # B4 Installation Status
    print_header "B4 Status"

    # Check if b4 is installed
    if [ -f "${INSTALL_DIR}/${BINARY_NAME}" ]; then
        print_detail "Binary Location" "${INSTALL_DIR}/${BINARY_NAME}"

        # Get version if possible
        if "${INSTALL_DIR}/${BINARY_NAME}" --version >/dev/null 2>&1; then
            b4_version=$("${INSTALL_DIR}/${BINARY_NAME}" --version 2>&1 | head -1)
            print_detail "Installed Version" "$b4_version"
        else
            print_detail "Installed Version" "Unknown (binary present)"
        fi

        # Check service status
        service_status=$(get_service_status)
        if echo "$service_status" | grep -q "running"; then
            print_detail "Service Status" "${GREEN}$service_status${NC}"
        else
            print_detail "Service Status" "${YELLOW}$service_status${NC}"
        fi
    else
        print_detail "Installation Status" "${RED}Not installed${NC}"
    fi

    # Check for config file
    if [ -f "$CONFIG_FILE" ]; then
        print_detail "Config File" "$CONFIG_FILE"

        # Check config content if jq is available
        if command_exists jq; then
            queue_num=$(jq -r '.queue_start_num // 537' "$CONFIG_FILE" 2>/dev/null || echo "537")
            threads=$(jq -r '.threads // 4' "$CONFIG_FILE" 2>/dev/null || echo "4")
            web_port=$(jq -r '.web_server.port // 0' "$CONFIG_FILE" 2>/dev/null || echo "0")
            print_detail "Queue Number" "$queue_num"
            print_detail "Worker Threads" "$threads"
            if [ "$web_port" != "0" ]; then
                print_detail "Web UI Port" "$web_port"
            fi
        fi
    else
        print_detail "Config File" "${YELLOW}Not found${NC}"
    fi

    # Check for geosite data
    if [ -f "$CONFIG_FILE" ] && command_exists jq; then
        geosite_path=$(jq -r '.system.geo.geosite_path // empty' "$CONFIG_FILE" 2>/dev/null)
        if [ -n "$geosite_path" ] && [ "$geosite_path" != "null" ] && [ -f "$geosite_path" ]; then
            geosite_size=$(du -h "$geosite_path" 2>/dev/null | cut -f1)
            print_detail "Geosite Data" "$geosite_path ($geosite_size)"
        fi
    fi

    # Service Manager Detection
    print_header "Service Management"

    if [ -f "/etc/systemd/system/b4.service" ] && which systemctl >/dev/null 2>&1; then
        print_detail "Service Manager" "systemd"
        print_detail "Service File" "/etc/systemd/system/b4.service"
    elif [ -f "/opt/etc/init.d/S99b4" ]; then
        print_detail "Service Manager" "Entware init"
        print_detail "Service File" "/opt/etc/init.d/S99b4"
    elif [ -f "/etc/init.d/b4" ]; then
        print_detail "Service Manager" "SysV init"
        print_detail "Service File" "/etc/init.d/b4"
    else
        print_detail "Service Manager" "${YELLOW}None configured${NC}"
    fi

    # Firewall/Netfilter Status
    print_header "Firewall & Netfilter"

    firewall_backend=$(detect_firewall_backend)
    print_detail "Firewall Backend" "$firewall_backend"

    # Check for iptables
    if which iptables >/dev/null 2>&1; then
        iptables_version=$(iptables --version 2>&1 | head -1 | awk '{print $2}' | tr -d 'v')
        print_detail "iptables" "${GREEN}Available${NC} (v$iptables_version)"

        # Check for b4 rules in iptables
        if iptables -t mangle -L B4 -n 2>/dev/null | grep -q NFQUEUE; then
            print_detail "iptables Rules" "${GREEN}Active${NC}"
        fi
    else
        print_detail "iptables" "${YELLOW}Not found${NC}"
    fi

    # Check for nftables
    if which nft >/dev/null 2>&1; then
        nft_version=$(nft --version 2>&1 | awk '{print $2}' | tr -d 'v')
        print_detail "nftables" "${GREEN}Available${NC} (v$nft_version)"

        # Check for b4 rules in nftables
        if nft list table inet b4_mangle 2>/dev/null | grep -q queue; then
            print_detail "nftables Rules" "${GREEN}Active${NC}"
        fi
    else
        print_detail "nftables" "${YELLOW}Not found${NC}"
    fi

    # Check for ip6tables
    if which ip6tables >/dev/null 2>&1; then
        print_detail "ip6tables" "${GREEN}Available${NC}"
    else
        print_detail "ip6tables" "${YELLOW}Not found${NC}"
    fi

    # Check netfilter queue status
    if [ -f /proc/net/netfilter/nfnetlink_queue ]; then
        nfqueue_info=$(cat /proc/net/netfilter/nfnetlink_queue 2>/dev/null | grep -v "^#" | head -1 || true)
        if [ -n "$nfqueue_info" ]; then
            print_detail "NFQueue Status" "${GREEN}Available${NC}"
        else
            print_detail "NFQueue Status" "${YELLOW}Available (no queues)${NC}"
        fi
    else
        print_detail "NFQueue Status" "${RED}Not available${NC}"
    fi

    # Kernel Modules
    print_header "Kernel Modules"

    # Netfilter modules
    modules="nf_conntrack xt_connbytes xt_NFQUEUE nf_tables nft_queue nft_ct"
    for mod in $modules; do
        status=$(check_kernel_module "$mod" || true)
        case "$status" in
        loaded)
            print_detail "$mod" "${GREEN}Loaded${NC}"
            ;;
        available)
            print_detail "$mod" "${CYAN}Available${NC}"
            ;;
        unknown)
            print_detail "$mod" "${YELLOW}Not found${NC}"
            ;;
        esac
    done

    # Check conntrack settings
    if [ -f /proc/sys/net/netfilter/nf_conntrack_checksum ]; then
        checksum=$(cat /proc/sys/net/netfilter/nf_conntrack_checksum 2>/dev/null || echo "1")
        if [ "$checksum" = "0" ]; then
            print_detail "conntrack_checksum" "${GREEN}Disabled (good)${NC}"
        else
            print_detail "conntrack_checksum" "${YELLOW}Enabled${NC}"
        fi
    fi

    if [ -f /proc/sys/net/netfilter/nf_conntrack_tcp_be_liberal ]; then
        liberal=$(cat /proc/sys/net/netfilter/nf_conntrack_tcp_be_liberal 2>/dev/null || echo "0")
        if [ "$liberal" = "1" ]; then
            print_detail "tcp_be_liberal" "${GREEN}Enabled (good)${NC}"
        else
            print_detail "tcp_be_liberal" "${YELLOW}Disabled${NC}"
        fi
    fi

    # Dependencies Check
    print_header "Dependencies"

    deps="wget curl tar jq sha256sum nohup"
    for dep in $deps; do
        if command_exists "$dep"; then
            print_detail "$dep" "${GREEN}Available${NC}"
        else
            print_detail "$dep" "${YELLOW}Not found${NC}"
        fi
    done

    # Package Manager Detection
    print_header "Package Management"

    if command_exists opkg; then
        print_detail "Package Manager" "opkg (OpenWRT/Entware)"
    elif command_exists apt-get; then
        print_detail "Package Manager" "apt (Debian/Ubuntu)"
    elif command_exists yum; then
        print_detail "Package Manager" "yum (RedHat/CentOS)"
    elif command_exists apk; then
        print_detail "Package Manager" "apk (Alpine)"
    else
        print_detail "Package Manager" "${YELLOW}None detected${NC}"
    fi

    # Recommendations
    print_header "Recommendations"

    recommendations=0

    # Check if running as root
    if [ "$USER" != "root" ] && ! (touch /etc/.root_test 2>/dev/null && rm -f /etc/.root_test 2>/dev/null); then
        printf "  ${YELLOW}⚠${NC}  Run this script as root for installation"
        recommendations=$((recommendations + 1))
    fi

    # Check for missing critical dependencies
    if ! command_exists wget && ! command_exists curl; then
        printf "  ${RED}✗${NC}  Install wget or curl for downloading"
        recommendations=$((recommendations + 1))
    fi

    if ! command_exists tar; then
        printf "  ${RED}✗${NC}  Install tar for extracting archives"
        recommendations=$((recommendations + 1))
    fi

    # Check for missing kernel modules
    if [ "$(check_kernel_module nf_conntrack)" = "missing" ]; then
        printf "  ${YELLOW}⚠${NC}  nf_conntrack module not found - may need kernel rebuild"
        recommendations=$((recommendations + 1))
    fi

    # Check firewall
    if [ "$firewall_backend" = "none" ]; then
        printf "  ${RED}✗${NC}  No firewall (iptables/nftables) detected"
        recommendations=$((recommendations + 1))
    fi

    # Check if b4 is installed but not running
    if [ -f "${INSTALL_DIR}/${BINARY_NAME}" ]; then
        service_status=$(get_service_status)
        if ! echo "$service_status" | grep -q "running"; then
            printf "  ${YELLOW}⚠${NC}  B4 is installed but not running"
            if [ -f "/etc/systemd/system/b4.service" ]; then
                printf "      Try: systemctl start b4"
            elif [ -f "/opt/etc/init.d/S99b4" ]; then
                printf "      Try: /opt/etc/init.d/S99b4 start"
            elif [ -f "/etc/init.d/b4" ]; then
                printf "      Try: /etc/init.d/b4 start"
            fi
            recommendations=$((recommendations + 1))
        fi
    fi

    if [ $recommendations -eq 0 ]; then
        printf "  ${GREEN}✓${NC}  System appears ready for B4"
    fi

    echo ""

}
