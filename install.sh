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
# These will be set dynamically by set_system_paths()
INSTALL_DIR=""
CONFIG_DIR=""
SERVICE_DIR=""
SERVICE_NAME=""
SYSTEM_TYPE=""
BINARY_NAME="b4"
CONFIG_FILE="" # Will be set after CONFIG_DIR is determined
TEMP_DIR="/tmp/b4_install_$$"
QUIET_MODE="0"
GEOSITE_SRC=""
GEOSITE_DST=""

# geodat sources (pipe-delimited: number|name|url)
GEODAT_SOURCES="1|Loyalsoldier source|https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/
2|RUNET Freedom source|https://raw.githubusercontent.com/runetfreedom/russia-v2ray-rules-dat/release/
3|Nidelon source|https://github.com/Nidelon/ru-block-v2ray-rules/releases/latest/download/
4|DustinWin source|https://github.com/DustinWin/ruleset_geodata/releases/download/mihomo/
5|Chocolate4U source|https://raw.githubusercontent.com/Chocolate4U/Iran-v2ray-rules/release/"

# Colors for output (if terminal supports it)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    CYAN='\033[0;36m'
    MAGENTA='\033[0;35m'
    BOLD='\033[1m'
    NC='\033[0m' # No Color
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    CYAN=''
    MAGENTA=''
    BOLD=''
    NC=''
fi

# Helper functions
print_info() {
    if [ "$QUIET_MODE" -eq 0 ]; then
        printf "${BLUE}[INFO   ]${NC} %s\n" "$1" >&2
    fi
}

print_success() {
    if [ "$QUIET_MODE" -eq 0 ]; then
        printf "${GREEN}[SUCCESS]${NC} %s\n" "$1" >&2
    fi
}

print_error() {
    printf "${RED}[ERROR  ]${NC} %s\n" "$1" >&2
}

print_warning() {
    if [ "$QUIET_MODE" -eq 0 ]; then
        printf "${YELLOW}[WARNING]${NC} %s\n" "$1" >&2
    fi
}

print_header() {
    if [ "$QUIET_MODE" -eq 0 ]; then
        printf "\n${MAGENTA}%s${NC}\n" "$1" >&2
    fi
}

print_detail() {
    if [ "$QUIET_MODE" -eq 0 ]; then
        printf "  ${CYAN}%-22s${NC}: %b\n" "$1" "$2" >&2
    fi
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
        fi
    done

    # Remove backup binaries
    print_info "Removing backup binaries..."
    for dir in $POSSIBLE_DIRS; do
        rm -f "$dir/${BINARY_NAME}.backup."* 2>/dev/null || true
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
# Detect system type and set appropriate paths
detect_system_type() {
    # Check for Entware
    if [ -d "/opt/etc/init.d" ] && [ -f "/opt/etc/entware_release" ]; then
        echo "entware"
        return
    fi

    # Check for OpenWRT
    if [ -f "/etc/openwrt_release" ]; then
        echo "openwrt"
        return
    fi

    # Check for MerlinWRT
    if [ -f "/etc/merlinwrt_release" ] || [ -d "/jffs" ]; then
        echo "merlin"
        return
    fi

    # Check for standard systemd-based Linux
    if [ -d "/etc/systemd/system" ] && command_exists systemctl; then
        echo "systemd-linux"
        return
    fi

    # Check for standard init.d Linux
    if [ -d "/etc/init.d" ] && [ ! -f "/etc/openwrt_release" ]; then
        echo "sysv-linux"
        return
    fi

    # Default to generic Linux
    echo "generic-linux"
}

# Set paths based on system type
set_system_paths() {
    SYSTEM_TYPE=$(detect_system_type)

    case "$SYSTEM_TYPE" in
    entware | merlin)
        INSTALL_DIR="/opt/sbin"
        CONFIG_DIR="/opt/etc/b4"
        SERVICE_DIR="/opt/etc/init.d"
        SERVICE_NAME="S99b4"
        ;;
    openwrt)
        # OpenWRT typically uses /usr/sbin or /usr/bin
        if [ -d "/usr/sbin" ]; then
            INSTALL_DIR="/usr/sbin"
        else
            INSTALL_DIR="/usr/bin"
        fi
        CONFIG_DIR="/etc/b4"
        SERVICE_DIR="/etc/init.d"
        SERVICE_NAME="b4"
        ;;
    systemd-linux)
        INSTALL_DIR="/usr/local/bin"
        CONFIG_DIR="/etc/b4"
        SERVICE_DIR="/etc/systemd/system"
        SERVICE_NAME="b4.service"
        ;;
    sysv-linux | generic-linux)
        INSTALL_DIR="/usr/local/bin"
        CONFIG_DIR="/etc/b4"
        SERVICE_DIR="/etc/init.d"
        SERVICE_NAME="b4"
        ;;
    *)
        # Fallback
        INSTALL_DIR="/usr/local/bin"
        CONFIG_DIR="/etc/b4"
        ;;
    esac

    CONFIG_FILE="${CONFIG_DIR}/b4.json"

    print_info "Detected system: $SYSTEM_TYPE"
    print_info "Using install directory: $INSTALL_DIR"
    print_info "Using config directory: $CONFIG_DIR"
}

# Detect system architecture and return appropriate binary variant
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
    armv7)
        arch_variant="armv7"
        ;;
    armv7* | armv7l | armv7-*)
        # Default to armv5 for compatibility, only use armv7 if certain
        arch_variant="armv5"

        # Only use armv7 if we have clear evidence of full support
        if [ -f /proc/cpuinfo ]; then
            # Need BOTH vfpv3+ AND proper architecture confirmation
            if grep -qE "(vfpv[3-9]|vfpv[0-9][0-9])" /proc/cpuinfo 2>/dev/null &&
                grep -qE "CPU architecture:\s*7" /proc/cpuinfo 2>/dev/null; then
                arch_variant="armv7"
                print_info "Full ARMv7 support detected, using armv7 binary"
            else
                print_warning "armv7l detected but using armv5 for compatibility (safer for routers)"
            fi
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
            # Look for CPU architecture line first (most reliable)
            if grep -qE "CPU architecture:\s*7" /proc/cpuinfo; then
                arch_variant="armv7"
            elif grep -qE "CPU architecture:\s*6" /proc/cpuinfo; then
                arch_variant="armv6"
            elif grep -qE "CPU architecture:\s*5" /proc/cpuinfo; then
                arch_variant="armv5"
            # Fallback to searching for ARM version strings
            elif grep -qi "ARMv7" /proc/cpuinfo; then
                arch_variant="armv7"
            elif grep -qi "ARMv6" /proc/cpuinfo; then
                arch_variant="armv6"
            elif grep -qi "ARMv5" /proc/cpuinfo; then
                arch_variant="armv5"
            else
                # Default to armv5 for maximum compatibility
                arch_variant="armv5"
            fi
        else
            # No cpuinfo available, default to safest option
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

    checksum_file="${file}.sha256"

    # Try to download checksum file
    print_info "Downloading SHA256 checksum..."
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
    if ! command_exists sha256sum; then
        print_warning "sha256sum not found, skipping SHA256 verification"
        rm -f "$checksum_file"
        return 1
    fi
    actual_checksum=$(sha256sum "$file" | awk '{print $1}')

    # Compare checksums
    if [ "$expected_checksum" = "$actual_checksum" ]; then
        print_success "SHA256 checksum verified: $actual_checksum"
        rm -f "$checksum_file"
        return 0
    else
        print_error "SHA256 checksum mismatch!"
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
        if [ "$QUIET_MODE" = "0" ]; then
            wget_opts="-q --show-progress"
        else
            wget_opts="-q"
        fi
        wget $wget_opts -O "$output" "$url" || {
            print_error "Download failed"
            return 1
        }
    elif command_exists curl; then
        curl -L -# -o "$output" "$url" || {
            print_error "Download failed"
            return 1
        }
    fi

    # Construct checksum URL
    file_name="${BINARY_NAME}-linux-${arch}.tar.gz"
    sha256_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${version}/${file_name}.sha256"

    # Try to verify SHA256 checksum
    if verify_checksum "$output" "$sha256_url"; then
        return 0
    elif [ $? -eq 2 ]; then
        # Checksum mismatch (not just missing)
        print_error "Download verification failed!"
        return 1
    else
        # Checksum file not found
        print_warning "No checksum file found in release - unable to verify download integrity"
        print_warning "Please verify manually if this is a security concern"

        # Still calculate and display local checksum for manual verification
        if command_exists sha256sum; then
            local_hash=$(sha256sum "$output" | awk '{print $1}')
            print_info "Local SHA256: $local_hash"
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

    rm -f "/opt/etc/init.d/S99b4" 2>/dev/null || true # remove legacy script
    rm -f "/etc/init.d/b4" 2>/dev/null || true        # remove legacy script
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
        rm -f "${INSTALL_DIR}/${BINARY_NAME}.backup.*" 2>/dev/null || true
        print_success "b4 installed successfully!"
    else
        print_warning "Binary installed but version check failed"
    fi
}

# Create systemd service file (for systems with systemd)
create_systemd_service() {
    # Only create if systemd is actually available and functioning
    if ! [ -d "/etc/systemd/system" ] || ! command_exists systemctl; then
        print_warning "Systemd not available, skipping service creation"
        return
    fi

    # Check if systemd is actually running (not just installed)
    if ! systemctl list-units >/dev/null 2>&1; then
        print_warning "Systemd not running, skipping service creation"
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

	(modprobe xt_connbytes --first-time >/dev/null 2>&1 && echo "xt_connbytes loaded") || true
	(modprobe xt_NFQUEUE --first-time >/dev/null 2>&1 && echo "xt_NFQUEUE loaded") || true
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

        chmod +x "${INIT_FULL_PATH}"
        sed -i "s|PROG_PLACEHOLDER|${INSTALL_DIR}/${BINARY_NAME}|g" "${INIT_FULL_PATH}"
        sed -i "s|CONFIG_PLACEHOLDER|${CONFIG_FILE}|g" "${INIT_FULL_PATH}"

        print_success "Init script created at ${INIT_FULL_PATH}"
        print_info "You can manage it with:"
        print_info "  ${INIT_FULL_PATH} start"
        print_info "  ${INIT_FULL_PATH} stop"
        print_info "  ${INIT_FULL_PATH} restart"

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
select_geo_source() {
    echo "" >&2
    echo "=======================================" >&2
    echo "  Select Geosite Data Source" >&2
    echo "=======================================" >&2
    echo "" >&2

    # Display sources (using POSIX-compliant iteration)
    echo "$GEODAT_SOURCES" | while IFS='|' read -r num name url; do
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
        selected_url=$(echo "$GEODAT_SOURCES" | grep "^${choice}|" | cut -d'|' -f3)
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

download_geodat() {
    geosite_url="$1/geosite.dat"
    geoip_url="$1/geoip.dat"

    save_dir="$2"
    geosite_file="${save_dir}/geosite.dat"
    geoip_file="${save_dir}/geoip.dat"

    print_info "Downloading $3 from: $geosite_url"

    # Create directory if it doesn't exist
    if [ ! -d "$save_dir" ]; then
        mkdir -p "$save_dir" || {
            print_error "Failed to create directory: $save_dir"
            return 1
        }
    fi

    # Download the file
    if command_exists wget; then
        if [ "$QUIET_MODE" = "0" ]; then
            wget_opts="-q --show-progress"
        else
            wget_opts="-q"
        fi
        wget $wget_opts -O "$geosite_file" "$geosite_url" || {
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

    # Download the file
    if command_exists wget; then
        if [ "$QUIET_MODE" = "0" ]; then
            wget_opts="-q --show-progress"
        else
            wget_opts="-q"
        fi
        wget $wget_opts -O "$geoip_file" "$geoip_url" || {
            print_error "Download failed"
            return 1
        }
    elif command_exists curl; then
        curl -L -# -o "$geoip_file" "$geoip_url" || {
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

    # Verify file was downloaded and is not empty
    if [ ! -f "$geoip_file" ] || [ ! -s "$geoip_file" ]; then
        print_error "Downloaded file is empty or missing"
        rm -f "$geoip_file"
        return 1
    fi

    print_success "Geosite file downloaded to: $geosite_file"
    print_success "GeoIP file downloaded to: $geoip_file"
    return 0
}

# Update config file with geosite path
update_config_geodat_path() {
    geosite_file="$1"
    geoip_file="$2"

    # Try to update with jq if available
    if command_exists jq; then
        print_info "Updating config file..."

        if [ ! -f "$CONFIG_FILE" ]; then
            jq -n --arg path "$geosite_file" --arg geosite_url "$geosite_url" --arg geoip_path "$geoip_file" --arg geoip_url "$geoip_url" '{
           domains: {
               geosite_path: $path,
               geosite_url: $geosite_url,
               geoip_path: $geoip_path,
               geoip_url: $geoip_url
           }
       }' >"$CONFIG_FILE"
            return 0
        fi

        # Create temporary file
        temp_file="${CONFIG_FILE}.tmp"

        # Update or add geosite_path
        if jq ".domains.geosite_path = \"$geosite_file\" | .domains.geosite_url = \"$geosite_url\" | .domains.geoip_path = \"$geoip_file\" | .domains.geoip_url = \"$geoip_url\" " "$CONFIG_FILE" >"$temp_file" 2>/dev/null; then
            mv "$temp_file" "$CONFIG_FILE" || {
                print_error "Failed to update config file"
                rm -f "$temp_file"
                return 0
            }
            print_success "Config updated with geosite path: $geosite_file"
            print_success "Config updated with geosite URL: $geosite_url"
            print_success "Config updated with geoip path: $geoip_file"
            print_success "Config updated with geoip URL: $geoip_url"
            return 0
        else
            print_error "Failed to parse config with jq"
            rm -f "$temp_file"
            return 0
        fi
    else
        print_warning "jq not found - cannot automatically update config"
        print_info "Please manually add to your config file:"
        print_info "  \"domains\": {"
        print_info "    \"geosite_path\": \"$geosite_file\""
        print_info "  }"
        echo ""
        print_info "Or remember to update Geosite Path in the B4 Web UI by accessing Settings -> Domains."
        return 0
    fi
}

# Setup geosite data
setup_geosite() {
    echo ""
    echo "======================================="
    echo "  Geosite Data Setup"
    echo "======================================="
    echo ""

    if [ -z "$GEOSITE_SRC" ] && [ -z "$GEOSITE_DST" ]; then
        # Ask if user wants to download geosite
        printf "${CYAN}Do you want to download geosite.dat file? (y/N): ${NC}"
        read answer
    else
        answer="y"
    fi

    case "$answer" in
    [yY] | [yY][eE][sS])
        # Select source
        if [ -z "$GEOSITE_SRC" ]; then
            geosite_url=$(select_geo_source)
            if [ $? -ne 0 ] || [ -z "$geosite_url" ]; then
                print_info "Geosite setup skipped"
                return 0
            fi
        else
            geosite_url="$GEOSITE_SRC"
            print_info "Using geosite source: $geosite_url"
        fi

        if [ -z "$GEOSITE_DST" ]; then

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
            read geosite_dst_dir

            # Use default if empty
            if [ -z "$geosite_dst_dir" ]; then
                geosite_dst_dir="$default_dir"
            fi
        else
            geosite_dst_dir="$GEOSITE_DST"
            print_info "Using geodat destination: $geosite_dst_dir"
        fi

        # Download geosite file
        download_geodat "$geosite_url" "$geosite_dst_dir"
        geosite_file="${geosite_dst_dir}/geosite.dat"
        geoip_file="${geosite_dst_dir}/geoip.dat"

        # Update config
        update_config_geodat_path "$geosite_file" "$geoip_file"

        print_success "Geosite setup completed!"
        return 0

        ;;
    *)
        print_info "Geosite setup skipped"
        ;;
    esac

    echo ""
    return 0
}

# Print web interface access information
print_web_interface_info() {

    local web_port
    if [ -f "$CONFIG_FILE" ] && command_exists jq; then
        web_port=$(jq -r '.web_server.port // 7000' "$CONFIG_FILE" 2>/dev/null)
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

    #  get args
    VERSION=""
    for arg in "$@"; do
        case "$arg" in
        v* | V*)
            VERSION="$arg"
            print_info "Using specified version: $VERSION"
            ;;
        --quiet | -q)
            QUIET_MODE=1
            ;;
        --geosite-src=*)
            GEOSITE_SRC="${arg#*=}"
            ;;
        --geosite-dst=*)
            GEOSITE_DST="${arg#*=}"
            ;;
        esac
    done

    # Detect system and set paths
    set_system_paths

    if [ "$QUIET_MODE" = "0" ]; then
        echo ""
        echo "======================================="
        echo "     B4 Universal Installer"
        echo "======================================="
        echo ""
    fi

    # Check if running as root
    check_root

    # Check dependencies
    check_dependencies

    # Detect architecture
    print_info "Detecting system architecture..."
    ARCH=$(detect_architecture)
    print_info "Raw architecture: $(uname -m)"
    print_success "Architecture detected: $ARCH"

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
    if [ "$SYSTEMCTL_CREATED" != "1" ]; then
        create_sysv_service
    fi

    if [ "$QUIET_MODE" = "0" ]; then
        setup_geosite

        # Print installation summary
        echo ""
        print_info "Binary installed to: ${INSTALL_DIR}/${BINARY_NAME}"
        print_info "Configuration at: ${CONFIG_FILE}"
        echo ""
        print_info "To see all B4 options:"
        print_info "  ${INSTALL_DIR}/${BINARY_NAME} --help"

        # Check PATH
        if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
            print_warning "Note: $INSTALL_DIR is not in your PATH"
            print_info "You may want to add it to your PATH or create a symlink:"
            print_info "  ln -s ${INSTALL_DIR}/${BINARY_NAME} /usr/bin/${BINARY_NAME}"
        fi

        print_web_interface_info

        echo ""
        print_success "Installation finished successfully!"
        echo ""
        printf "${CYAN}Start B4 service now? (Y/n): ${NC}"
        read answer

        if [ -z "$answer" ]; then
            answer="y"
        fi

        case "$answer" in
        [nN] | [nN][oO])
            print_info "Service not started. Start manually when ready."
            ;;
        *)
            if [ -f "/etc/systemd/system/b4.service" ] && command_exists systemctl; then
                systemctl restart b4 2>/dev/null && print_success "Service started"
            elif [ -f "/opt/etc/init.d/S99b4" ]; then
                /opt/etc/init.d/S99b4 restart 2>/dev/null && print_success "Service started"
            elif [ -f "/etc/init.d/b4" ]; then
                /etc/init.d/b4 restart 2>/dev/null && print_success "Service started"
            fi
            ;;
        esac
        echo ""

        echo "======================================="
        echo "       Installation Complete!"
        echo "======================================="
    fi

}

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
        GEOSITE_SRC=$(jq -r '.domains.geosite_url // empty' "$CONFIG_FILE" 2>/dev/null)
        GEOSITE_DST=$(jq -r '.domains.geosite_path // empty' "$CONFIG_FILE" 2>/dev/null | xargs dirname 2>/dev/null)
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
    if case "$SERVICE_MANAGER" in
        systemd) systemctl is-active --quiet b4 ;;
        entware | init) ps | grep -v grep | grep -q "b4$\|b4[[:space:]]" ;;
        esac then
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
    if [ -f "/etc/systemd/system/b4.service" ] && which systemctl >/dev/null 2>&1; then
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
        geosite_path=$(jq -r '.domains.geosite_path // empty' "$CONFIG_FILE" 2>/dev/null)
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
        --update | -u)
            check_root
            perform_update "$@"
            exit 0
            ;;
        --info | -i | --sysinfo)
            show_system_info
            exit 0
            ;;
        --help | -h)
            echo "Usage: $0 [OPTIONS] [VERSION]"
            echo ""
            echo "Options:"
            echo "  --sysinfo, -i     Show system information and b4 status"
            echo "  --remove, -r      Uninstall b4 from the system"
            echo "  --update, -u      Update b4 to latest version"
            echo "  --help, -h        Show this help message"
            echo "  --quiet, -q       Suppress output except for errors"
            echo "  --geosite-src URL Specify geosite.dat source URL"
            echo "  --geosite-dst DIR Specify directory to save geosite.dat"
            echo "  VERSION           Install specific version (e.g., v1.4.0)"
            echo ""
            echo "Examples:"
            echo "  $0                Install latest version"
            echo "  $0 v1.4.0         Install version 1.4.0"
            echo "  $0 --sysinfo      Show system diagnostics"
            echo "  $0 --update       Update to latest version"
            echo "  $0 --remove       Uninstall b4"
            exit 0
            ;;
        esac
    done

    # No remove/update flag found, proceed with installation
    main_install "$@"
}
# Run main function
main "$@"
