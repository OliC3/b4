# B4 - Bye Bye Big Bro

## [1.18.4] - 2025-11-25

- CHANGED: update set default `desync` mode to 'off'.
- FIXED: simplify package handling.

## [1.18.3] - 2025-11-25

- FIXED: `Domains` table not updating with new packets due to React state reference issue.
- FIXED: `Domains` table columns layout being cramped with fixed widths.

## [1.18.2] - 2025-11-25

- ADDED: `ClientHello Mutation` support for `IPv6`.
- FIXED: Discovery presets missing default values for `SNI Mutation`, `TCP Window`, and `Desync` settings.

## [1.18.1] - 2025-11-25

- FIXED: Unable to change `ClientHello Mutation` mode in the Faking settings.

## [1.18.0] - 2025-11-24

- IMPROVED: Overall performance (frontend and backend).
- ADDED: `TCP Window Manipulation` (`--tcp-win-mode`) - sends fake packets with manipulated TCP window sizes to confuse stateful DPI. Modes: `oscillate` (cycling window values), `zero` (zero-window probe attack), `random` (randomized windows), `escalate` (gradually increasing windows).
- ADDED: `TCP Desync Attack` (`--tcp-desync-mode`) - injects fake TCP control packets (RST/FIN/ACK) with low TTL and corrupted checksums to desynchronize DPI connection tracking. Modes: `rst`, `fin`, `ack`, `combo`, `full`.
- ADDED: `SNI Mutation` for ClientHello fingerprint evasion - modifies TLS handshake structure to bypass DPI fingerprinting. Modes: `duplicate` (inject fake SNIs), `grease` (add GREASE extensions), `padding` (add padding extension), `reorder` (shuffle extensions), `full` (all mutations combined), `advanced` (TLS 1.3 features like PSK/key_share).

## [1.17.1] - 2025-11-23

- ADDED: `Out-of-Band` (OOB) data handling with configurable position, reverse order, and character (`--frag=oob`).
- ADDED: `Out-of-Band` (OOB) strategies to `B4Discovery`.
- ADDED: `TLS Record Splitting` fragmentation strategy (`--frag=tls`) - splits ClientHello into multiple TLS records to bypass DPI expecting single-record handshakes.
- ADDED: `SACK dropping` (`--tcp-drop-sack`) - strips Selective Acknowledgment options from TCP headers to force full retransmissions and confuse stateful DPI tracking.
- UPDATED: Fake `SNI` payload now uses TLS 1.3 ClientHello structure with `staticcdn.duckduckgo.com`.
- IMPROVED: `SNI` fragmentation for long domains (>30 bytes). Now splits 12 bytes before SNI end instead of middle, ensuring domain suffixes like `googlevideo.com` are properly fragmented across packets.
- IMPROVED: `Matcher` performance with LRU caching for large geosite/geoip categories (70-90% CPU reduction for sets with big data inside).
- IMPROVED: `Geodat` download workflow - files now immediately available in sets manager without restart, config auto-reloads after download.
- IMPROVED: Set `Fragmentation` tab refactored.
- FIXED: Logs level can be switched witout reloading the app.
- FIXED: Config validation bug where Main Set was compared against itself, causing startup failure with `TCP ConnBytesLimit greater than main set` error.
- FIXED: update default fake SNI payload to use new format.
- CHANGED: Renamed `--frag-sni-reverse` to `--frag-reverse` and update related configurations.

## [1.16.1] - 2025-11-20

- ADDED: Asynchronous packet injection for TCP and UDP traffic. Verdict is now sent to kernel immediately, with packet manipulation performed in parallel. Eliminates kernel queue blocking that previously caused video streaming hangs and site loading delays.
- FIXED: Critical performance bottleneck where each QUIC/UDP packet with default configuration (FakeSeqLength: 6) would block the kernel for 6ms minimum. This caused YouTube and other video services to buffer or hang intermittently.
- FIXED: IPv6 QUIC packet processing incorrectly used TCP delay settings instead of UDP delay settings.
- FIXED: New configuration sets created from the Domains page were not saving custom names, defaulting to generic "Set 1/2/3" names instead.
- IMPROVED: Removed unnecessary `1ms` sleep delays when `Seg2Delay` is set to `0`, reducing packet processing latency by up to `6ms` per QUIC packet.

## [1.16.0] - 2025-11-17

- ADDED: Configuration sets can now be enabled/disabled without deletion.
- ADDED: Clear button next to the IP/CIDR list in the set configuration.
- ADDED: Download `GeoSite`/`GeoIP` database files directly from Settings UI with preset sources.
- IMPROVED: Redesigned `/test` page UX - domains are now managed directly on the test page.
- IMPROVED: Refactor `Discovery` presets generation logic and add new test strategies.
- FIXED: Resolved severe performance bottleneck on `/domains` page when adding ASN filters (caused by expensive ASN lookup operations executing on every render).
- REMOVED: Test domain configuration from `Settings` - domains are now managed exclusively on the Test page.

## [1.15.0] - 2025-11-16

- ADDED: SYN fake packet functionality for advanced DPI bypass. Sends fake SYN packets with configurable payload length to confuse DPI systems before the real connection is established. Configure via `--tcp-syn-fake` and `--tcp-syn-fake-len` flags, or through the TCP settings in Web UI.
- ADDED: IP information enrichment via `IPInfo` API integration. When IPInfo token is configured in Settings → API, click on any destination IP in `/domains` monitoring page to view detailed geolocation, ASN, organization, and network information.
- ADDED: `RIPE` [Stat integration](https://www.ripe.net/) for network intelligence. View ASN prefix announcements and detailed network information directly from the Web UI. Helps identify IP ranges for precise targeting.
- ADDED: Configuration set import/export functionality. Share working bypass configurations between devices or users by exporting sets as JSON files. Import proven configurations with one click to quickly replicate successful setups across multiple installations.
- IMPROVED: Discovery test results now include individual configuration cards per domain instead of single recommended configuration, making it easier to analyze which specific settings work best for each target domain.

## [1.14.0] - 2025-11-13

- ADDED: Select target configuration set when adding domains or IP/CIDR addresses from `/domains` monitoring page. Allows precise control over which configuration set receives the new entry.
- ADDED: One-click configuration adoption from `Discovery` test results. Apply the best-performing configuration directly to your configuration list without manual copying.
- CHANGED: Complete overhaul of `Discovery` testing service with improved reliability and performance. Now they should work as expected.
- FIXED: Memory leaks and overall memory management improvements for better long-term stability.
- FIXED: Update process through the Web UI.

## [1.13.0] - 2025-11-10

- ADDED: Click on destination IP addresses in `/domains` monitoring page to add them to configuration. Modal allows adding either exact IP or CIDR notation for broader site coverage. This does not require to reload or restart B4, works on the fly.
- ADDED: Toggle switch in `/domains` monitoring page to view all packets or only those with identified SNI/domain. Useful for monitoring and debugging `UDP` traffic.
- ADDED: Geodat domain/IP counters at configuration sets.
- ADDED: a new tab under `/test` menu. `Discovery` test results now show individual configuration cards per domain instead of a single recommended configuration, making it easier to see what works best for each specific domain.
- CHANGED: `UDP` port filtering now uses a single flexible field instead of separate "from" and "to" fields. Supports comma-separated ports and ranges (e.g., `80,443,2000-3000`).
- CHANGED: Packages count badge in `/domains` menu now only counts packets processed by B4 targets.
- CHANGED: Replaced `--udp-dport-min` and `--udp-dport-max` flags with single `--udp-dport-filter` flag for flexible port filtering.
- CHANGED: Refactored UDP/QUIC packet handling and UDP-related UI tab in the set configuration.
- FIXED: `UDP` entries are now logged even when UDP packets are configured to be ignored in the configuration
- FIXED: `UI` crash when using filter in /domains monitoring page.
- FIXED: Manually added domains no longer require service restart when geodat files are not configured.
- FIXED: Test Suites now correctly report success when DPI is bypassed regardless of `HTTP status code`. Any HTTP response (including non-200 codes) indicates successful DPI circumvention.

## [1.12.0] - 2025-11-09

- ADDED: Configuration Sets - fine-grained bypass control for different targets
  - Create multiple configuration sets, each with independent TCP/UDP/fragmentation/faking settings
  - Target packets by SNI domain, destination IP/CIDR ranges, or UDP port ranges
- ADDED: `geoip.dat` support.

## [1.11.0] - 2025-11-05

- ADDED: DPI Bypass Test feature to verify that circumvention is working. The feature tests configured domains and measures download speeds to ensure B4 is functioning correctly. Visit the `/test` page to run tests and `/settings/checker` to configure test settings (define which domains to test, etc.).
- ADDED: New feature to reset B4 settings to their defaults. The reset button is located in the `Core` tab on the `Settings` page.
- CHANGED: Moved `RESTART B4 BUTTON` to the `Core` tab on the Settings page (under the `Core Controls` section).
- IMPROVED: Enhanced `flowState` struct to track `SNI` detection and processing status.
- FIXED: Service restart functionality in the UI for different service managers (`Entware`/`OpenWRT`/`systemctl`).
- FIXED: Pause shortcut (pressing down the `P` key on the domains and logs pages) interfering with search input.

## [1.10.1] - 2025-11-03

- IMPROVED: Intermittent connection failures where blocked sites would randomly fail to load in certain browsers (`Safari`, `Firefox`, `Chrome`). Connections _should_ now be more stable and reliable across all browsers by optimizing packet fragmentation strategy.

## [1.10.0] - 2025-11-02

- ADDED: Automatic `iptables`/`nftables` rules restoration. B4 now automatically detects this and restores itself without requiring a manual restart.
- ADDED: New `--tables-monitor-interval` setting to control how often B4 checks if its rules are still active (default: `10` seconds). Set to `0` to disable automatic monitoring.

## [1.9.2] - 2025-11-02

- IMPROVED: Increase TTL and buffer limit for flow state management.
- IMPROVED: enhance SNI character validation.

## [1.9.1] - 2025-11-02

- FIXED: Return back missing `geosite path` field to the settings.

## [1.9.0] - 2025-11-02

- ADDED: Hotkeys to the `/domains` and `/logs` page. Press `ctrl+x` or `Delete` keys to clear the entries. Press `p` or `Pause` to pause the stram.
- ADDED: Parse regex entries from the geosite files.
- ADDED: Connection bytes limit configuration for TCP and UDP in network settings
- FIXED: Wrong total number of total domains in the settings.

## [1.8.0] - 2025-11-01

- ADDED: `nftables` support.
- CHANGED: `--skip-iptables` and `--clear-iptables` renamed to `--skip-tables` and `--clear-tables`.
- IMPROVED: TCP flow handling by fragmenting packets after SNI detection.

## [1.7.0] - 2025-10-31

- ADDED: 'RESTART SERVICE` Button in the Settings to perform the B4 restart from the Web UI.
- ADDED: Add `quiet` mode and `geosite` source/destination options to installer script. Use `b4install.sh --help` to get more information.
- ADDED: Sort Domains by clicking the columns.
- ADDED: Update a new version from the Web Interface.
- REMOVED: iptables `OUTPUT` rule.

## [1.6.0] - 2025-10-29

- FIXED: `Dashboard` works again.
- REMOVED: `--conntrack` and `-gso` flags since they both are not used in the project.
- IMPROVED: Installation script now handles a geosite file setup.

## [1.5.0] - 2025-10-28

- ADDED: `--clear-iptables` argument to perform a cleanup of iptable rules.
- ADDED: `IPv6` support.
- ADDED: `--ipv4` (default is `true`) and `--ipv6` (default is `false`) arguments to control protocol versions.
- IMPROVED: Handling of geodata domains.
