# B4 - Bye Bye Big Bro

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
