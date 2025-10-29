#!/bin/sh
#
# B4 Universal Installer Script (POSIX Compliant)
# Automatically detects system architecture and installs the appropriate b4 binary
# Supports OpenWRT, MerlinWRT, and other Linux-based routers with only sh shell
#

set -e

# Configuration
REPO_OWNER="DanielLavrushin"
REPO_NAME="b4"
INSTALL_DIR="/opt/sbin"
BINARY_NAME="b4"
CONFIG_DIR="/opt/etc/b4"
CONFIG_FILE="${CONFIG_DIR}/b4.json"
TEMP_DIR="/tmp/b4_install_$$"

# Geosite sources (pipe-delimited: number|name|url)
GEOSITE_SOURCES="1|Loyalsoldier source|https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geosite.dat
2|RUNET Freedom source|https://raw.githubusercontent.com/runetfreedom/russia-v2ray-rules-dat/release/geosite.dat
3|Nidelon source|https://github.com/Nidelon/ru-block-v2ray-rules/releases/latest/download/geosite.dat
4|DustinWin source|https://github.com/DustinWin/ruleset_geodata/releases/download/mihomo/geosite.dat
5|Chocolate4U source|https://raw.githubusercontent.com/Chocolate4U/Iran-v2ray-rules/release/geosite.dat"

# Colors for output (if terminal supports it)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    CYAN='\033[0;36m'
    NC='\033[0m' # No Color
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

# Helper functions
print_info() {
    printf "${BLUE}[INFO   ]${NC} %s\n" "$1" >&2
}

print_success() {
    printf "${GREEN}[SUCCESS]${NC} %s\n" "$1" >&2
}

print_error() {
    printf "${RED}[ERROR  ]${NC} %s\n" "$1" >&2
}

print_warning() {
    printf "${YELLOW}[WARNING]${NC} %s\n" "$1" >&2
}

# Check if command exists (works on routers without 'command' builtin)
command_exists() {
    # Try which first (common on routers)
    if which "$1" >/dev/null 2>&1; then
        return 0
    fi

    # Fallback: try to run with --help (most commands support this)
    if $1 --help >/dev/null 2>&1; then
        return 0
    fi

    # Command not found
    return 1
}

# Check if running as root (works on minimal routers)
check_root() {
    # Method 1: Check $USER variable
    if [ "$USER" = "root" ]; then
        return 0
    fi

    # Method 2: Try to write to /etc (only root can)
    if touch /etc/.root_test 2>/dev/null; then
        rm -f /etc/.root_test 2>/dev/null
        return 0
    fi

    # Method 3: Check whoami if available
    if which whoami >/dev/null 2>&1 && [ "$(whoami 2>/dev/null)" = "root" ]; then
        return 0
    fi

    # If we get here, probably not root
    print_error "This script must be run as root"
    print_info "Please switch to root user first"
    exit 1
}

# Remove/Uninstall b4
remove_b4() {
    echo ""
    echo "======================================="
    echo "     B4 Uninstaller"
    echo "======================================="
    echo ""

    # Stop the service first
    print_info "Stopping b4 service if running..."

    # Check systemd service
    if [ -f "/etc/systemd/system/b4.service" ]; then
        if which systemctl >/dev/null 2>&1; then
            systemctl stop b4 2>/dev/null || true
            systemctl disable b4 2>/dev/null || true
            print_info "Stopped systemd service"
        fi
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

    # Remove binary
    if [ -f "${INSTALL_DIR}/${BINARY_NAME}" ]; then
        print_info "Removing binary: ${INSTALL_DIR}/${BINARY_NAME}"
        rm -f "${INSTALL_DIR}/${BINARY_NAME}"
        print_success "Binary removed"
    else
        print_warning "Binary not found: ${INSTALL_DIR}/${BINARY_NAME}"
    fi

    # Remove backup binaries
    print_info "Removing backup binaries..."
    rm -f "${INSTALL_DIR}/${BINARY_NAME}.backup."* 2>/dev/null || true

    # Remove service files
    if [ -f "/etc/systemd/system/b4.service" ]; then
        print_info "Removing systemd service..."
        rm -f "/etc/systemd/system/b4.service"
        if which systemctl >/dev/null 2>&1; then
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

    # Ask about configuration
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

# Detect system architecture - ONLY returns arch string
detect_architecture() {
    arch=$(uname -m)
    arch_variant=""

    case "$arch" in
    x86_64 | amd64)
        arch_variant="amd64"
        ;;
    i386 | i486 | i586 | i686)
        arch_variant="386"
        ;;
    aarch64 | arm64)
        arch_variant="arm64"
        ;;
    armv7* | armv7l)
        # Check for hardware floating point support
        if grep -q "vfpv" /proc/cpuinfo 2>/dev/null; then
            arch_variant="armv7"
        else
            arch_variant="armv6"
        fi
        ;;
    armv6*)
        arch_variant="armv6"
        ;;
    armv5*)
        arch_variant="armv5"
        ;;
    arm*)
        # Generic ARM - try to detect version from CPU info
        if [ -f /proc/cpuinfo ]; then
            if grep -q "ARMv7" /proc/cpuinfo; then
                arch_variant="armv7"
            elif grep -q "ARMv6" /proc/cpuinfo; then
                arch_variant="armv6"
            elif grep -q "ARMv5" /proc/cpuinfo; then
                arch_variant="armv5"
            else
                # Default to armv5 for maximum compatibility
                arch_variant="armv5"
            fi
        else
            arch_variant="armv5"
        fi
        ;;
    mips64)
        # Check endianness (POSIX compliant)
        if printf '\001' | od -An -tx1 | grep -q '01'; then
            arch_variant="mips64le"
        else
            arch_variant="mips64"
        fi
        ;;
    mips*)
        # Check if 64-bit capable
        if command_exists getconf && getconf LONG_BIT 2>/dev/null | grep -q "64"; then
            # 64-bit MIPS
            if printf '\001' | od -An -tx1 | grep -q '01'; then
                arch_variant="mips64le"
            else
                arch_variant="mips64"
            fi
        else
            # 32-bit MIPS
            if printf '\001' | od -An -tx1 | grep -q '01'; then
                arch_variant="mipsle"
            else
                arch_variant="mips"
            fi
        fi
        ;;
    ppc64le)
        arch_variant="ppc64le"
        ;;
    ppc64)
        arch_variant="ppc64"
        ;;
    riscv64)
        arch_variant="riscv64"
        ;;
    s390x)
        arch_variant="s390x"
        ;;
    loongarch64)
        arch_variant="loong64"
        ;;
    *)
        print_error "Unsupported architecture: $arch"
        exit 1
        ;;
    esac

    # ONLY output the result to stdout
    echo "$arch_variant"
}

# Get latest release version from GitHub - ONLY returns version string
get_latest_version() {
    api_url="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"
    version=""

    # Try wget first, then curl
    if command_exists wget; then
        version=$(wget -qO- "$api_url" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    elif command_exists curl; then
        version=$(curl -s "$api_url" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    else
        print_error "Neither wget nor curl found. Please install one of them."
        exit 1
    fi

    if [ -z "$version" ]; then
        print_error "Failed to fetch latest version"
        exit 1
    fi

    # ONLY output the result to stdout
    echo "$version"
}

# Verify checksum
verify_checksum() {
    file="$1"
    checksum_url="$2"
    checksum_type="$3" # "sha256" or "md5"

    checksum_file="${file}.${checksum_type}"

    # Get display name for checksum type (POSIX compliant)
    case "$checksum_type" in
    sha256) checksum_display="SHA256" ;;
    md5) checksum_display="MD5" ;;
    *) checksum_display="$checksum_type" ;;
    esac

    # Try to download checksum file
    print_info "Downloading ${checksum_display} checksum..."
    if command_exists wget; then
        if ! wget -q -O "$checksum_file" "$checksum_url" 2>/dev/null; then
            return 1
        fi
    elif command_exists curl; then
        if ! curl -s -L -o "$checksum_file" "$checksum_url" 2>/dev/null; then
            return 1
        fi
    else
        return 1
    fi

    # Check if checksum file was actually downloaded (not a 404 page)
    if [ ! -s "$checksum_file" ]; then
        rm -f "$checksum_file"
        return 1
    fi

    # Extract expected checksum (handle format: "checksum filename")
    expected_checksum=$(cat "$checksum_file" | awk '{print $1}')

    if [ -z "$expected_checksum" ]; then
        print_warning "Could not parse checksum from file"
        rm -f "$checksum_file"
        return 1
    fi

    # Calculate actual checksum
    if [ "$checksum_type" = "sha256" ]; then
        if ! command_exists sha256sum; then
            print_warning "sha256sum not found, skipping SHA256 verification"
            rm -f "$checksum_file"
            return 1
        fi
        actual_checksum=$(sha256sum "$file" | awk '{print $1}')
    elif [ "$checksum_type" = "md5" ]; then
        if ! command_exists md5sum; then
            print_warning "md5sum not found, skipping MD5 verification"
            rm -f "$checksum_file"
            return 1
        fi
        actual_checksum=$(md5sum "$file" | awk '{print $1}')
    else
        return 1
    fi

    # Compare checksums
    if [ "$expected_checksum" = "$actual_checksum" ]; then
        print_success "${checksum_display} checksum verified: $actual_checksum"
        rm -f "$checksum_file"
        return 0
    else
        print_error "${checksum_display} checksum mismatch!"
        print_error "Expected: $expected_checksum"
        print_error "Got:      $actual_checksum"
        print_error "File may be corrupted or tampered with!"
        rm -f "$checksum_file"
        return 2
    fi
}

# Download file and verify checksums
download_file() {
    url="$1"
    output="$2"
    version="$3"
    arch="$4"

    print_info "Downloading from: $url"

    # Download the file
    if command_exists wget; then
        wget -q --show-progress -O "$output" "$url" || {
            print_error "Download failed"
            return 1
        }
    elif command_exists curl; then
        curl -L -# -o "$output" "$url" || {
            print_error "Download failed"
            return 1
        }
    fi

    # Construct checksum URLs
    file_name="${BINARY_NAME}-linux-${arch}.tar.gz"
    sha256_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${version}/${file_name}.sha256"
    md5_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${version}/${file_name}.md5"

    # Try to verify checksums (try SHA256 first, then MD5)
    checksum_verified=0

    if verify_checksum "$output" "$sha256_url" "sha256"; then
        checksum_verified=1
    elif [ $? -eq 2 ]; then
        # Checksum mismatch (not just missing)
        print_error "Download verification failed!"
        return 1
    fi

    if [ $checksum_verified -eq 0 ]; then
        if verify_checksum "$output" "$md5_url" "md5"; then
            checksum_verified=1
        elif [ $? -eq 2 ]; then
            # Checksum mismatch (not just missing)
            print_error "Download verification failed!"
            return 1
        fi
    fi

    if [ $checksum_verified -eq 0 ]; then
        print_warning "No checksums found in release - unable to verify download integrity"
        print_warning "Please verify manually if this is a security concern"

        # Still calculate and display local checksum for manual verification
        if command_exists sha256sum; then
            local_hash=$(sha256sum "$output" | awk '{print $1}')
            print_info "Local SHA256: $local_hash"
        elif command_exists md5sum; then
            local_hash=$(md5sum "$output" | awk '{print $1}')
            print_info "Local MD5: $local_hash"
        fi
    fi

    return 0
}

# Check for required dependencies
check_dependencies() {
    missing_deps=""

    # Check for tar
    if ! command_exists tar; then
        missing_deps="${missing_deps} tar"
    fi

    # Check for download tool
    if ! command_exists wget && ! command_exists curl; then
        missing_deps="${missing_deps} wget/curl"
    fi

    if [ -n "$missing_deps" ]; then
        print_error "Missing required dependencies:$missing_deps"
        print_info "Please install them first"

        # Try to detect package manager and suggest install command
        if command_exists opkg; then
            print_info "For OpenWRT, try: opkg update && opkg install wget tar"
        elif command_exists apt-get; then
            print_info "Try: apt-get update && apt-get install -y wget tar"
        elif command_exists yum; then
            print_info "Try: yum install -y wget tar"
        fi

        exit 1
    fi
}

# Create necessary directories
setup_directories() {
    print_info "Creating directories..."

    # Create install directory if it doesn't exist
    if [ ! -d "$INSTALL_DIR" ]; then
        mkdir -p "$INSTALL_DIR" || {
            print_error "Failed to create install directory: $INSTALL_DIR"
            exit 1
        }
    fi

    # Create config directory
    if [ ! -d "$CONFIG_DIR" ]; then
        mkdir -p "$CONFIG_DIR" || {
            print_error "Failed to create config directory: $CONFIG_DIR"
            exit 1
        }
    fi

    # Create temp directory
    mkdir -p "$TEMP_DIR" || {
        print_error "Failed to create temp directory"
        exit 1
    }
}

# Clean up temporary files
cleanup() {
    if [ -d "$TEMP_DIR" ]; then
        rm -rf "$TEMP_DIR"
    fi
}

# Set up trap for cleanup
trap cleanup EXIT INT TERM

# Check if process is running (POSIX compliant, no pidof)
is_process_running() {
    process_name="$1"
    # Match exact binary name, not the installer script
    if ps 2>/dev/null | grep -v grep | grep -v "b4install" | grep -q "^.*${process_name}$\|^.*${process_name}[[:space:]]"; then
        return 0
    else
        return 1
    fi
}

# Stop process (POSIX compliant)
stop_process() {
    process_name="$1"
    if is_process_running "$process_name"; then
        print_info "Stopping existing $process_name process..."
        # Try pkill if available, otherwise use ps + kill
        if command_exists pkill; then
            pkill "^${process_name}$" 2>/dev/null || true
        else
            # BusyBox way: find and kill by name, exclude installer script
            ps | grep -v grep | grep -v "b4install" | grep "${process_name}$\|${process_name}[[:space:]]" | awk '{print $1}' | while read pid; do
                if [ -n "$pid" ]; then
                    kill "$pid" 2>/dev/null || true
                fi
            done
        fi
        sleep 2
    fi
}

# Install b4 binary
install_b4() {
    arch="$1"
    version="$2"

    # Construct download URL
    file_name="${BINARY_NAME}-linux-${arch}.tar.gz"
    download_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${version}/${file_name}"
    archive_path="${TEMP_DIR}/${file_name}"

    # Download the archive with checksum verification
    if ! download_file "$download_url" "$archive_path" "$version" "$arch"; then
        print_error "Failed to download b4 for architecture: $arch"
        exit 1
    fi

    # Extract the binary
    print_info "Extracting archive..."
    cd "$TEMP_DIR"
    tar -xzf "$archive_path" || {
        print_error "Failed to extract archive"
        exit 1
    }

    # Check if binary exists
    if [ ! -f "${BINARY_NAME}" ]; then
        print_error "Binary not found in archive"
        exit 1
    fi

    # Stop existing b4 if running
    stop_process "$BINARY_NAME"

    # Create timestamp in POSIX way
    timestamp=$(date '+%Y%m%d_%H%M%S')

    # Backup existing binary if it exists
    if [ -f "${INSTALL_DIR}/${BINARY_NAME}" ]; then
        print_info "Backing up existing binary..."
        mv "${INSTALL_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}.backup.${timestamp}"
    fi

    # Install the new binary
    print_info "Installing b4 to ${INSTALL_DIR}..."
    cp "${BINARY_NAME}" "${INSTALL_DIR}/" || {
        print_error "Failed to copy binary to install directory"
        exit 1
    }

    # Set executable permissions
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}" || {
        print_error "Failed to set executable permissions"
        exit 1
    }

    # Verify installation
    if "${INSTALL_DIR}/${BINARY_NAME}" --version >/dev/null 2>&1; then
        rm -f "${INSTALL_DIR}/${BINARY_NAME}.backup.${timestamp}"
        print_success "b4 installed successfully!"
    else
        print_warning "Binary installed but version check failed"
    fi
}

# Create systemd service file (for systems with systemd)
create_systemd_service() {
    # Check if systemd directory exists and systemctl is available
    if [ -d "/etc/systemd/system" ] && which systemctl >/dev/null 2>&1; then
        print_info "Creating systemd service..."

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
    fi
}

# Create OpenWRT/Entware init script
create_openwrt_init() {
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

    # Only proceed if we found a writable init directory and opkg exists
    if [ -n "$INIT_DIR" ] && which opkg >/dev/null 2>&1; then
        print_info "Creating init script in $INIT_DIR..."

        cat >"${INIT_DIR}/S99b4" <<'EOF'
#!/bin/sh

# B4 DPI Bypass Service Init Script
PROG=/opt/sbin/b4
CONFIG_FILE=/opt/etc/b4/b4.json
PIDFILE=/var/run/b4.pid

start() {
    echo "Starting b4..."
    if [ -f "$PIDFILE" ] && kill -0 $(cat "$PIDFILE") 2>/dev/null; then
        echo "b4 is already running"
        return 1
    fi

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

        chmod +x "${INIT_DIR}/S99b4"
        print_success "Init script created at ${INIT_DIR}/S99b4"
        print_info "You can manage it with:"
        print_info "  ${INIT_DIR}/S99b4 start"
        print_info "  ${INIT_DIR}/S99b4 stop"
        print_info "  ${INIT_DIR}/S99b4 restart"

        # For Entware systems, remind about rc.unslung
        if [ "$INIT_DIR" = "/opt/etc/init.d" ]; then
            print_info "To start on boot, the script will run automatically via Entware"
        fi
    else
        print_warning "Could not create init script - no writable init directory found"
    fi
}

# Get geosite path from config using jq if available
get_geosite_path_from_config() {
    if [ -f "$CONFIG_FILE" ] && command_exists jq; then
        geosite_path=$(jq -r '.domains.geosite_path // empty' "$CONFIG_FILE" 2>/dev/null)
        if [ -n "$geosite_path" ] && [ "$geosite_path" != "null" ]; then
            # Extract directory from path
            echo "$(dirname "$geosite_path")"
            return 0
        fi
    fi
    return 1
}

# Display geosite source menu and get user choice
select_geosite_source() {
    echo "" >&2
    echo "=======================================" >&2
    echo "  Select Geosite Data Source" >&2
    echo "=======================================" >&2
    echo "" >&2

    # Display sources (using POSIX-compliant iteration)
    echo "$GEOSITE_SOURCES" | while IFS='|' read -r num name url; do
        if [ -n "$num" ]; then
            printf "  ${GREEN}%s${NC}) %s\n" "$num" "$name" >&2
        fi
    done

    echo "" >&2
    printf "${CYAN}Select source (1-5) or 'q' to skip: ${NC}" >&2
    read choice

    case "$choice" in
    [qQ] | [qQ][uU][iI][tT])
        return 1
        ;;
    [1-5])
        # Extract URL for selected choice (POSIX-compliant)
        selected_url=$(echo "$GEOSITE_SOURCES" | grep "^${choice}|" | cut -d'|' -f3)
        if [ -n "$selected_url" ]; then
            echo "$selected_url"
            return 0
        else
            print_error "Invalid selection"
            return 1
        fi
        ;;
    *)
        print_error "Invalid selection"
        return 1
        ;;
    esac
}

# Download geosite file
download_geosite() {
    geosite_url="$1"
    save_dir="$2"
    geosite_file="${save_dir}/geosite.dat"

    print_info "Downloading geosite.dat from: $geosite_url"

    # Create directory if it doesn't exist
    if [ ! -d "$save_dir" ]; then
        mkdir -p "$save_dir" || {
            print_error "Failed to create directory: $save_dir"
            return 1
        }
    fi

    # Download the file
    if command_exists wget; then
        wget -q --show-progress -O "$geosite_file" "$geosite_url" || {
            print_error "Download failed"
            return 1
        }
    elif command_exists curl; then
        curl -L -# -o "$geosite_file" "$geosite_url" || {
            print_error "Download failed"
            return 1
        }
    else
        print_error "Neither wget nor curl found"
        return 1
    fi

    # Verify file was downloaded and is not empty
    if [ ! -f "$geosite_file" ] || [ ! -s "$geosite_file" ]; then
        print_error "Downloaded file is empty or missing"
        rm -f "$geosite_file"
        return 1
    fi

    print_success "Geosite file downloaded to: $geosite_file"
    return 0
}

# Update config file with geosite path
update_config_geosite_path() {
    geosite_file="$1"

    if [ ! -f "$CONFIG_FILE" ]; then
        print_warning "Config file not found: $CONFIG_FILE"
        print_info "You'll need to manually add geosite_path to your config"
        print_info "Set domains.geosite_path to: $geosite_file"
        return 1
    fi

    # Try to update with jq if available
    if command_exists jq; then
        print_info "Updating config file..."

        # Create temporary file
        temp_file="${CONFIG_FILE}.tmp"

        # Update or add geosite_path
        if jq ".domains.geosite_path = \"$geosite_file\"" "$CONFIG_FILE" >"$temp_file" 2>/dev/null; then
            mv "$temp_file" "$CONFIG_FILE" || {
                print_error "Failed to update config file"
                rm -f "$temp_file"
                return 1
            }
            print_success "Config updated with geosite path: $geosite_file"
            return 0
        else
            print_error "Failed to parse config with jq"
            rm -f "$temp_file"
            return 1
        fi
    else
        print_warning "jq not found - cannot automatically update config"
        print_info "Please manually add to your config file:"
        print_info "  \"domains\": {"
        print_info "    \"geosite_path\": \"$geosite_file\""
        print_info "  }"
        echo ""
        print_info "Or remember to update Geosite Path in the B4 Web UI by accessing Settings -> Domains."
        return 1
    fi
}

# Setup geosite data
setup_geosite() {
    echo ""
    echo "======================================="
    echo "  Geosite Data Setup"
    echo "======================================="
    echo ""

    # Ask if user wants to download geosite
    printf "${CYAN}Do you want to download geosite.dat file? (y/N): ${NC}"
    read answer

    case "$answer" in
    [yY] | [yY][eE][sS])
        # Select source
        geosite_url=$(select_geosite_source)
        if [ $? -ne 0 ] || [ -z "$geosite_url" ]; then
            print_info "Geosite setup skipped"
            return 0
        fi

        # Get save directory
        default_dir="$CONFIG_DIR"

        # Try to get existing path from config
        existing_dir=$(get_geosite_path_from_config || true)
        if [ -n "$existing_dir" ]; then
            default_dir="$existing_dir"
            print_info "Found existing geosite path in config: $default_dir"
        fi

        echo ""
        printf "${CYAN}Save directory [${default_dir}]: ${NC}"
        read save_dir

        # Use default if empty
        if [ -z "$save_dir" ]; then
            save_dir="$default_dir"
        fi

        # Download geosite file
        if download_geosite "$geosite_url" "$save_dir"; then
            geosite_file="${save_dir}/geosite.dat"

            # Update config
            update_config_geosite_path "$geosite_file"

            print_success "Geosite setup completed!"
        else
            print_error "Failed to download geosite file"
        fi
        ;;
    *)
        print_info "Geosite setup skipped"
        ;;
    esac

    echo ""
}

# Print web interface access information
print_web_interface_info() {

    local web_port
    if [ -f "$CONFIG_FILE" ] && command_exists jq; then
        web_port=$(jq -r '.web_server.port // empty' "$CONFIG_FILE" 2>/dev/null)
    else
        web_port="7000"
    fi

    echo ""
    echo "======================================="
    echo "  Web Interface Access"
    echo "======================================="
    echo ""

    # Get LAN IP (br0 interface on routers)
    lan_ip=""

    # Try to get IP from br0 specifically (most common on routers)
    if command_exists ip; then
        lan_ip=$(ip -4 addr show br0 2>/dev/null | grep 'inet ' | awk '{print $2}' | cut -d'/' -f1)
    fi

    # Fallback to ifconfig
    if [ -z "$lan_ip" ] && command_exists ifconfig; then
        lan_ip=$(ifconfig br0 2>/dev/null | grep 'inet addr:' | awk '{print $2}' | cut -d':' -f2)
    fi

    # If br0 didn't work, look for any 192.168.x.x address
    if [ -z "$lan_ip" ]; then
        if command_exists ip; then
            lan_ip=$(ip -4 addr show 2>/dev/null | grep 'inet 192.168' | head -n1 | awk '{print $2}' | cut -d'/' -f1)
        elif command_exists ifconfig; then
            lan_ip=$(ifconfig 2>/dev/null | grep 'inet addr:192.168' | head -n1 | awk '{print $2}' | cut -d':' -f2)
        fi
    fi

    # Print local/LAN access
    if [ -n "$lan_ip" ]; then
        print_info "Local network access (LAN):"
        printf "        ${GREEN}http://%s:%s${NC}\n" "$lan_ip" "$web_port"
        printf "        (remember to start the service first)\n"
        echo ""
    fi

    # Get external IP
    print_info "Checking external IP address..."
    external_ip=""

    if command_exists curl; then
        external_ip=$(curl -s --max-time 3 ifconfig.me 2>/dev/null || true)
    elif command_exists wget; then
        external_ip=$(wget -qO- --timeout=3 ifconfig.me 2>/dev/null || true)
    fi

    # Print external access if different from LAN
    if [ -n "$external_ip" ] && [ "$external_ip" != "$lan_ip" ]; then
        print_info "External access (WAN, if port ${web_port} is forwarded):"
        printf "        ${GREEN}http://%s:%s${NC}\n" "$external_ip" "$web_port"
        print_warning "Note: Ensure port ${web_port} is open in your firewall"
    fi

    echo ""
}

# Main installation process
main_install() {
    echo ""
    echo "======================================="
    echo "     B4 Universal Installer"
    echo "======================================="
    echo ""

    # Check if running as root
    check_root

    # Check dependencies
    check_dependencies

    # Detect architecture
    print_info "Detecting system architecture..."
    ARCH=$(detect_architecture)
    print_info "Raw architecture: $(uname -m)"
    print_success "Architecture detected: $ARCH"

    # Get latest version or use provided version
    VERSION=""
    for arg in "$@"; do
        case "$arg" in
        v* | V*)
            VERSION="$arg"
            print_info "Using specified version: $VERSION"
            ;;
        esac
    done

    if [ -z "$VERSION" ]; then
        print_info "Fetching latest release information..."
        VERSION=$(get_latest_version)
        print_success "Latest version: $VERSION"
    fi

    # Setup directories
    setup_directories

    # Install b4
    install_b4 "$ARCH" "$VERSION"

    # Create service files
    create_systemd_service
    create_openwrt_init

    # Setup geosite (new feature)
    setup_geosite

    # Print installation summary
    echo ""
    print_info "Binary installed to: ${INSTALL_DIR}/${BINARY_NAME}"
    print_info "Configuration at: ${CONFIG_FILE}"
    echo ""
    print_info "To see all B4 options:"
    print_info "  ${INSTALL_DIR}/${BINARY_NAME} --help"

    echo ""
    print_info "To start B4 now:"
    if [ -n "$INIT_DIR" ]; then
        print_info "  ${INIT_DIR}/S99b4 start         # For OpenWRT/Entware systems"
    fi
    if [ -d "/etc/systemd/system" ]; then
        print_info "  systemctl start b4              # For systemd systems"
    fi
    echo ""

    # Check PATH
    if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
        print_warning "Note: $INSTALL_DIR is not in your PATH"
        print_info "You may want to add it to your PATH or create a symlink:"
        print_info "  ln -s ${INSTALL_DIR}/${BINARY_NAME} /usr/bin/${BINARY_NAME}"
    fi

    print_web_interface_info

    echo "======================================="
    echo "       Installation Complete!"
    echo "======================================="
}

# Main function - parse arguments
main() {
    # Check for remove flag first
    for arg in "$@"; do
        case "$arg" in
        --remove | --uninstall | -r)
            check_root
            remove_b4
            exit 0
            ;;
        --help | -h)
            echo "Usage: $0 [OPTIONS] [VERSION]"
            echo ""
            echo "Options:"
            echo "  --remove, -r     Uninstall b4 from the system"
            echo "  --help, -h       Show this help message"
            echo "  VERSION          Install specific version (e.g., v1.4.0)"
            echo ""
            echo "Examples:"
            echo "  $0               Install latest version"
            echo "  $0 v1.4.0        Install version 1.4.0"
            echo "  $0 --remove      Uninstall b4"
            exit 0
            ;;
        esac
    done

    # No remove flag found, proceed with installation
    main_install "$@"
}

# Run main function
main "$@"
