# B4 - Bye Bye Big Bro

![GitHub Release](https://img.shields.io/github/v/release/daniellavrushin/b4?logoColor=violet)
![GitHub Release Date](https://img.shields.io/github/release-date/daniellavrushin/b4)
![GitHub commits since latest release](https://img.shields.io/github/commits-since/daniellavrushin/b4/latest)
![GitHub Downloads (specific asset, latest release)](https://img.shields.io/github/downloads/daniellavrushin/b4/latest/total)
![image](https://img.shields.io/github/downloads/DanielLavrushin/b4/total?label=total%20downloads)

**B4** is a high-performance, next-generation network packet processor designed to circumvent Deep Packet Inspection (DPI) systems. Built with Go and featuring a modern React-based web interface, B4 provides intelligent traffic obfuscation to maintain privacy and unrestricted access to content.

![alt text](image.png)

## Overview

**B4** operates at the kernel level using Linux `netfilter` queues to intercept and process network packets in real-time. It analyzes TLS/QUIC traffic patterns and applies sophisticated evasion techniques to bypass DPI systems used by ISPs and network administrators.

## How It Works

- Packet Interception: Captures packets via netfilter queue before they leave your device
- SNI Detection: Identifies TLS ClientHello and QUIC Initial packets containing Server Name Indication
- Smart Evasion: Applies configurable fragmentation and obfuscation techniques
- Transparent Routing: Sends modified packets that appear normal to endpoints but confuse DPI systems

## Key Features

### Advanced DPI Bypass

- Multi-Strategy Evasion: TCP/IP fragmentation, fake packet injection, sequence manipulation
- Protocol Support: TLS (HTTPS) and QUIC (HTTP/3) traffic processing
- SNI-Based Targeting: Selective processing based on domain names

### Intelligent Domain Filtering

- Geodata Integration: Support for geosite.dat/geoip.dat community database files (v2ray-rules format)
- Category-Based Filtering: Filter by service categories (youtube, netflix, google, etc.)
- Manual Domain Lists: Add custom domains for processing

### Modern Web Interface

- Real-Time Dashboard: Live metrics, connection monitoring, and system health
- Interactive Configuration: Web-based settings management on-the-fly
- Live Logs: WebSocket-powered log streaming with filtering
- Domain Analysis: See intercepted connections and add domains on-the-fly

### Performance & Compatibility

- Multi-Threaded Processing: Configurable worker threads for high throughput
- Cross-Platform: Linux systems, OpenWRT routers, MerlinWRT, Entware environments
- Low Overhead: Efficient packet processing with minimal latency
  -IPv4/IPv6 Support: Dual-stack networking

## Prerequisites

- Linux kernel with netfilter support
- Root/CAP_NET_ADMIN privileges
- Go 1.24+ (for building)
- Node.js 18+ and pnpm (for building web UI)

## Installation

### To install B4

```bash
wget -O ~/b4install.sh https://raw.githubusercontent.com/DanielLavrushin/b4/main/install.sh && chmod +x ~/b4install.sh && ~/b4install.sh
```

### Installer Options

The B4 installer script (`b4install.sh`) supports various flags for different use cases:

### Basic Usage

```bash
# Install latest version
./b4install.sh

# Install specific version
./b4install.sh v1.8.0

# Quiet mode (minimal output)
./b4install.sh --quiet
```

### Available Flags

#### Installation Flags

| Flag                | Description                    | Example                                    |
| ------------------- | ------------------------------ | ------------------------------------------ |
| `VERSION`           | Install specific version       | `./b4install.sh v1.4.0`                    |
| `--quiet`, `-q`     | Suppress output except errors  | `./b4install.sh --quiet`                   |
| `--geosite-src=URL` | Specify geosite.dat source URL | `./b4install.sh --geosite-src=https://...` |
| `--geosite-dst=DIR` | Directory to save geosite.dat  | `./b4install.sh --geosite-dst=/opt/etc/b4` |

#### Management Flags

| Flag                            | Description              | Example                   |
| ------------------------------- | ------------------------ | ------------------------- |
| `--update`, `-u`                | Update to latest version | `./b4install.sh --update` |
| `--remove`, `--uninstall`, `-r` | Uninstall B4             | `./b4install.sh --remove` |
| `--help`, `-h`                  | Show help message        | `./b4install.sh --help`   |

### Update Process

The `--update` flag performs an automatic update:

```bash
./b4install.sh --update
```

### Uninstallation

```bash
# Interactive uninstall (asks about config)
wget -O ~/b4install.sh https://raw.githubusercontent.com/DanielLavrushin/b4/main/install.sh && chmod +x ~/b4install.sh && ~/b4install.sh --remove

# The uninstaller:
# - Stops all B4 processes
# - Removes binary and backups
# - Removes service files (systemd/init)
# - Optionally keeps or removes configuration
# - Cleans up iptables/nftables rules
```

### Service Manager Detection

The installer intelligently detects and configures the appropriate service manager:

| System Type    | Service Manager | Start Command                 |
| -------------- | --------------- | ----------------------------- |
| Standard Linux | systemd         | `systemctl start b4`          |
| OpenWRT        | procd init      | `/etc/init.d/b4 start`        |
| Entware/Merlin | Entware init    | `/opt/etc/init.d/S99b4 start` |

### Troubleshooting Installation

#### Installation Fails

```bash
# Check for missing dependencies
opkg update && opkg install wget tar  # OpenWRT
apt-get install wget tar              # Debian/Ubuntu
```

## Quick Start

### entware/merlinwrt

```bash
/opt/etc/init.d/S99b4 restart
```

### Service Management

```bash
sudo systemctl start b4
sudo systemctl enable b4 #to restart on reboot
```

## Configuration

### Command-Line Flags

#### Network Configuration

- `--queue-num` - Netfilter queue number (default: 537)
- `--threads` - Number of worker threads (default: 4)
- `--mark` - Packet mark value (default: 32768)
- `--connbytes-limit` - Connection bytes limit (default: 19)

#### Geodata Filtering

- `--geosite` - Path to geosite.dat file
- `--geo-categories` - Categories to process (e.g., youtube,facebook)
- `--sni-domains` - Comma-separated list of domains (can be used as additional list together with `--geo-categories`)

#### TCP Fragmentation

- `--frag` - Fragmentation strategy: tcp/ip/none (default: tcp)
- `--frag-sni-reverse` - Reverse fragment order (default: true)
- `--frag-middle-sni` - Fragment in middle of SNI (default: true)
- `--frag-sni-pos` - SNI fragment position (default: 1)

#### Fake SNI Configuration

- `--fake-sni` - Enable fake SNI packets (default: true)
- `--fake-ttl` - TTL for fake packets (default: 8)
- `--fake-strategy` - Strategy: ttl/randseq/pastseq/tcp_check/md5sum (default: pastseq)
- `--fake-seq-offset` - Sequence offset for fake packets (default: 10000)
- `--fake-sni-len` - Length of fake SNI sequence (default: 1)
- `--fake-sni-type` - Payload type: 0=random, 1=custom, 2=default

#### UDP/QUIC Configuration

- `--udp-mode` - UDP handling: drop/fake (default: drop)
- `--udp-fake-seq-len` - UDP fake packet sequence length (default: 6)
- `--udp-fake-len` - UDP fake packet size in bytes (default: 64)
- `--udp-faking-strategy` - Strategy: none/ttl/checksum (default: none)
- `--udp-dport-min` - Minimum UDP destination port (default: 0)
- `--udp-dport-max` - Maximum UDP destination port (default: 0)
- `--udp-filter-quic` - QUIC filtering: disabled/all/parse (default: parse)

#### System Configuration

- `--skip-tables` - Skip iptables/nftables rules setup
- `--seg2delay` - Delay between segments in ms (default: 0)

#### Logging

- `--verbose` - Verbosity level: debug/trace/info/silent (default: info)
- `--instaflush` - Flush logs immediately (default: true)
- `--syslog` - Enable syslog output

#### Web Server

- `--web-port` - Port for web interface (0 disables) (default: 0)

### Configuration File

B4 can save and load configuration from `b4.json`:

```json
{
  "queue_start_num": 537,
  "threads": 4,
  "web_server": {
    "port": 7000
  },
  "logging": {
    "level": "info",
    "instaflush": true,
    "syslog": false
  }
}
```

## Web Interface

Access the web interface at `http://your-router-ip:7000` (configure port with `--web-port`).

Features:

- **Domains**: Real-time table of intercepted connections with protocol, SNI, source/destination
- **Logs**: Live streaming logs with filtering and search
- **Settings**: Configuration management (under development)

## Advanced Usage

### Custom Fake Packet Strategies

```bash
# TTL manipulation (fake packets expire before reaching destination)
sudo b4 --fake-strategy ttl --fake-ttl 5

# Random sequence numbers (confuse state tracking)
sudo b4 --fake-strategy randseq

# Past sequence numbers (appear to be retransmissions)
sudo b4 --fake-strategy pastseq --fake-seq-offset 8192

# TCP checksum corruption
sudo b4 --fake-strategy tcp_check
```

## Troubleshooting

### No packets being processed

```bach
# Check iptables rules
sudo iptables -t mangle -vnL --line-numbers

# Verify nfqueue status
cat /proc/net/netfilter/nfnetlink_queue

# Check B4 is running with correct permissions (root)
ps aux | grep b4
```

### Web interface not accessible

- Check if port is open: `sudo netstat -tlnp | grep 7000`
- Verify `--web-port` is set
- Check firewall rules

### High CPU usage

```bash
# Reduce worker threads
sudo b4 --threads 2

# Limit to specific domains
sudo b4 --sni-domains specific-domain.com
```

## Credits

B4 builds upon research and techniques from:

- [youtubeUnblock](https://github.com/Waujito/youtubeUnblock)
- [GoodbyeDPI](https://github.com/ValdikSS/GoodbyeDPI)
- [zapret](https://github.com/bol-van/zapret)

## License

This project is provided as-is for educational purposes. Users are responsible for compliance with local laws and regulations.

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## Disclaimer

This tool is intended for bypassing censorship and protecting privacy in regions where internet access is restricted. It should not be used for **illegal activities**. `The authors are not responsible for misuse of this software`.
