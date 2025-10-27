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
TEMP_DIR="/tmp/b4_install_$$"

# Colors for output (if terminal supports it)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
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
    printf "${YELLOW}Remove configuration files as well? (y/N): ${NC}"
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

# Download file
download_file() {
    url="$1"
    output="$2"

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

    # Calculate and display checksum for verification
    if command_exists sha256sum; then
        print_info "Calculating checksum..."
        actual_hash=$(sha256sum "$output" | cut -d' ' -f1)
        print_info "SHA256: $actual_hash"
        print_warning "Note: GitHub releases for b4 don't include checksums, please verify manually if needed"
    else
        print_warning "sha256sum not found, cannot calculate checksum"
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

    # Download the archive
    if ! download_file "$download_url" "$archive_path"; then
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

    # Backup existing binary if it exists
    if [ -f "${INSTALL_DIR}/${BINARY_NAME}" ]; then
        print_info "Backing up existing binary..."
        # Create timestamp in POSIX way
        timestamp=$(date '+%Y%m%d_%H%M%S')
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
ExecStart=${INSTALL_DIR}/${BINARY_NAME} --config ${CONFIG_DIR}/b4.json
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
    $PROG --config $CONFIG_FILE > /var/log/b4.log 2>&1 &
    echo $! > "$PIDFILE"
    echo "b4 started"
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

    # Print installation summary
    echo ""
    print_info "Binary installed to: ${INSTALL_DIR}/${BINARY_NAME}"
    print_info "Configuration at: ${CONFIG_DIR}/b4.json"
    echo ""
    print_info "To see all B4 options:"
    print_info "  ${INSTALL_DIR}/${BINARY_NAME} --help"

    echo ""
    print_info "To start B4 now:"
    print_info "  ${INIT_DIR}/S99b4 restart # For OpenWRT/Entware systems"
    print_info "  systemctl start b4                  # For systemd systems"
    echo ""

    # Check PATH
    if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
        print_warning "Note: $INSTALL_DIR is not in your PATH"
        print_info "You may want to add it to your PATH or create a symlink:"
        print_info "  ln -s ${INSTALL_DIR}/${BINARY_NAME} /usr/bin/${BINARY_NAME}"
    fi

    echo "======================================="
    echo "     Installation Complete!"
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
