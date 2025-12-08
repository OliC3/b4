import {
  Grid,
  Alert,
  Box,
  Typography,
  Chip,
  Stack,
  List,
  ListItemButton,
  ListItemText,
  ListItemIcon,
} from "@mui/material";
import {
  Dns as DnsIcon,
  Check as CheckIcon,
  Security as SecurityIcon,
  Speed as SpeedIcon,
  Block as BlockIcon,
} from "@mui/icons-material";
import SettingSection from "@molecules/common/B4Section";
import SettingTextField from "@atoms/common/B4TextField";
import SettingSwitch from "@atoms/common/B4Switch";
import { B4SetConfig } from "@models/Config";
import { colors } from "@design";

interface DnsSettingsProps {
  config: B4SetConfig;
  ipv6: boolean;
  onChange: (field: string, value: string | boolean) => void;
}

const POPULAR_DNS = [
  {
    ip: "8.8.4.4",
    name: "Google Secondary",
    desc: "Fast, global (~14ms)",
    dnssec: true,
    tags: [],
    warn: true,
  },
  {
    ip: "185.228.168.9",
    name: "CleanBrowsing",
    desc: "Security filter, Seattle (~10ms)",
    dnssec: true,
    tags: ["fast", "security"],
  },
  {
    ip: "95.85.95.85",
    name: "Gcore",
    desc: "Fast CDN, Luxembourg (~11ms)",
    dnssec: true,
    tags: ["fast"],
  },
  {
    ip: "2001:4860:4860::8844",
    name: "Google IPv6",
    desc: "Fast, global (~11ms)",
    dnssec: true,
    tags: ["ipv6"],
    warn: true,
    ipv6: true,
  },
  {
    ip: "195.46.39.105",
    name: "SafeDNS",
    desc: "Frankfurt, content filter (~11ms)",
    dnssec: true,
    tags: ["fast", "filtering"],
  },
  {
    ip: "208.67.222.222",
    name: "OpenDNS",
    desc: "Cisco, customizable (~12ms)",
    dnssec: true,
    tags: ["fast", "filtering"],
  },
  {
    ip: "2606:4700:4700::1001",
    name: "Cloudflare IPv6",
    desc: "Fast, privacy-first (~12ms)",
    dnssec: true,
    tags: ["fast", "privacy", "ipv6"],
    ipv6: true,
  },
  {
    ip: "1.1.1.1",
    name: "Cloudflare",
    desc: "Fastest, privacy-first (~12ms)",
    dnssec: true,
    tags: ["fast", "privacy"],
  },
  {
    ip: "195.46.39.39",
    name: "SafeDNS Primary",
    desc: "Frankfurt (~13ms)",
    dnssec: true,
    tags: ["fast", "filtering"],
  },
  {
    ip: "8.8.8.8",
    name: "Google",
    desc: "Fast, global (~14ms)",
    dnssec: true,
    tags: [],
    warn: true,
  },
  {
    ip: "223.6.6.6",
    name: "Alibaba",
    desc: "China, may be slow/blocked (~15ms)",
    dnssec: false,
    tags: ["china", "poisoned"],
    china: true,
  },
  {
    ip: "45.90.29.77",
    name: "NextDNS",
    desc: "NY, customizable (~16ms)",
    dnssec: true,
    tags: ["fast", "privacy"],
  },
  {
    ip: "208.67.220.220",
    name: "OpenDNS Secondary",
    desc: "Newark (~16ms)",
    dnssec: true,
    tags: ["fast", "filtering"],
  },
  {
    ip: "2620:119:35::35",
    name: "OpenDNS IPv6",
    desc: "Cisco (~16ms)",
    dnssec: true,
    tags: ["ipv6", "filtering"],
    ipv6: true,
  },
  {
    ip: "1.0.0.3",
    name: "Cloudflare Family",
    desc: "Malware + adult filter (~17ms)",
    dnssec: true,
    tags: ["fast", "family"],
  },
  {
    ip: "8.26.56.26",
    name: "Comodo Secure",
    desc: "Malware blocking (~17ms)",
    dnssec: true,
    tags: ["fast", "security"],
  },
  {
    ip: "208.67.222.220",
    name: "OpenDNS Alt",
    desc: "Newark (~17ms)",
    dnssec: true,
    tags: ["filtering"],
  },
  {
    ip: "2620:0:ccc::2",
    name: "OpenDNS IPv6 Alt",
    desc: "NY (~19ms)",
    dnssec: true,
    tags: ["ipv6", "filtering"],
    ipv6: true,
  },
  {
    ip: "8.20.247.10",
    name: "Comodo Secondary",
    desc: "NJ (~20ms)",
    dnssec: true,
    tags: ["security"],
  },
  {
    ip: "223.5.5.5",
    name: "Alibaba Primary",
    desc: "China, may be slow/blocked (~20ms)",
    dnssec: false,
    tags: ["china", "poisoned"],
    china: true,
  },
  {
    ip: "1.1.1.2",
    name: "Cloudflare Malware",
    desc: "Blocks malware (~20ms)",
    dnssec: true,
    tags: ["security"],
  },
  {
    ip: "1.0.0.2",
    name: "Cloudflare Malware Alt",
    desc: "Blocks malware (~20ms)",
    dnssec: true,
    tags: ["security"],
  },
  {
    ip: "149.112.112.13",
    name: "Quad9",
    desc: "SF, malware blocking (~24ms)",
    dnssec: true,
    tags: ["privacy", "security"],
  },
  {
    ip: "64.6.64.6",
    name: "Neustar",
    desc: "NY, reliable (~24ms)",
    dnssec: true,
    tags: ["privacy"],
  },
  {
    ip: "2620:fe::10",
    name: "Quad9 IPv6",
    desc: "SF (~25ms)",
    dnssec: true,
    tags: ["ipv6", "privacy", "security"],
    ipv6: true,
  },
  {
    ip: "2620:0:ccd::2",
    name: "OpenDNS IPv6 Newark",
    desc: "Newark (~25ms)",
    dnssec: true,
    tags: ["ipv6", "filtering"],
    ipv6: true,
  },
  {
    ip: "2400:3200:baba::1",
    name: "Alibaba IPv6",
    desc: "China, may be slow/blocked",
    dnssec: false,
    tags: ["ipv6", "china", "poisoned"],
    ipv6: true,
    china: true,
  },
  {
    ip: "149.112.112.9",
    name: "Quad9 Alt",
    desc: "SF (~43ms)",
    dnssec: true,
    tags: ["privacy", "security"],
  },
  {
    ip: "94.140.14.140",
    name: "AdGuard Family",
    desc: "Blocks ads + adult (~43ms)",
    dnssec: true,
    tags: ["adblock", "family"],
  },
  {
    ip: "9.9.9.10",
    name: "Quad9 No Filter",
    desc: "No malware blocking (~45ms)",
    dnssec: true,
    tags: ["privacy"],
  },
  {
    ip: "9.9.9.9",
    name: "Quad9 Primary",
    desc: "Malware blocking (~51ms)",
    dnssec: true,
    tags: ["privacy", "security"],
  },
  {
    ip: "149.112.112.112",
    name: "Quad9 Secondary",
    desc: "SF (~52ms)",
    dnssec: true,
    tags: ["privacy", "security"],
  },
  {
    ip: "8.26.56.10",
    name: "Comodo Alt",
    desc: "Clifton (~62ms)",
    dnssec: true,
    tags: ["security"],
  },
  {
    ip: "149.112.112.11",
    name: "Quad9 ECS",
    desc: "SF, with ECS (~64ms)",
    dnssec: true,
    tags: ["privacy"],
  },
  {
    ip: "185.228.169.9",
    name: "CleanBrowsing EU",
    desc: "Amsterdam (~80ms)",
    dnssec: true,
    tags: ["security"],
  },
  {
    ip: "205.171.3.66",
    name: "Level3",
    desc: "CenturyLink, Chicago (~108ms)",
    dnssec: true,
    tags: [],
  },
  {
    ip: "2620:fe::fe",
    name: "Quad9 IPv6 Primary",
    desc: "SF (~111ms)",
    dnssec: true,
    tags: ["ipv6", "privacy", "security"],
    ipv6: true,
  },
  {
    ip: "74.82.42.42",
    name: "Hurricane Electric",
    desc: "Reliable backbone (~116ms)",
    dnssec: true,
    tags: ["privacy"],
  },
  {
    ip: "77.88.8.8",
    name: "Yandex",
    desc: "⚠️ Russian, likely poisoned",
    dnssec: true,
    tags: ["poisoned"],
    warn: true,
  },
  {
    ip: "204.194.232.200",
    name: "OpenDNS Miami",
    desc: "Miami (~144ms)",
    dnssec: true,
    tags: ["filtering"],
  },
  {
    ip: "216.146.35.35",
    name: "Oracle Dyn",
    desc: "Ashburn (~149ms)",
    dnssec: true,
    tags: [],
  },
  {
    ip: "2.56.220.2",
    name: "Gcore IPv4 Alt",
    desc: "Luxembourg (~194ms)",
    dnssec: true,
    tags: [],
  },
  {
    ip: "94.140.15.15",
    name: "AdGuard Secondary",
    desc: "Cyprus (~270ms)",
    dnssec: true,
    tags: ["adblock"],
  },
  {
    ip: "8.20.247.20",
    name: "Comodo Slow",
    desc: "Clifton (~316ms)",
    dnssec: true,
    tags: ["security"],
  },
  {
    ip: "2a10:50c0::1:ff",
    name: "AdGuard IPv6",
    desc: "Cyprus (~480ms)",
    dnssec: true,
    tags: ["ipv6", "adblock"],
    ipv6: true,
  },
].sort((a, b) => a.name.localeCompare(b.name));

export function DnsSettings({ config, onChange, ipv6 }: DnsSettingsProps) {
  const dns = config.dns || { enabled: false, target_dns: "" };
  const selectedServer = POPULAR_DNS.find((d) => d.ip === dns.target_dns);

  const handleServerSelect = (ip: string) => {
    onChange("dns.target_dns", ip);
  };

  return (
    <SettingSection
      title="DNS Redirect"
      description="Redirect DNS queries to bypass ISP DNS poisoning"
      icon={<DnsIcon />}
    >
      <Grid container spacing={3}>
        <Grid size={{ xs: 12 }}>
          <Alert severity="info">
            Some ISPs intercept DNS queries (especially to 8.8.8.8) and return
            fake IPs for blocked domains. DNS redirect transparently rewrites
            your DNS queries to use an unpoisoned resolver.
          </Alert>
        </Grid>

        <Grid size={{ xs: 12, md: 6 }}>
          <SettingSwitch
            label="Enable DNS Redirect"
            checked={dns.enabled}
            onChange={(checked: boolean) => onChange("dns.enabled", checked)}
            description="Redirect DNS queries for domains in this set to specified DNS server"
          />
        </Grid>

        {dns.enabled && (
          <>
            {/* Custom IP input */}
            <Grid size={{ xs: 12, md: 6 }}>
              <SettingSwitch
                label="Fragment DNS Queries"
                checked={dns.fragment_query || false}
                onChange={(checked: boolean) =>
                  onChange("dns.fragment_query", checked)
                }
                description="Split DNS packets using IP fragmentation to bypass DPI that pattern-matches domain names in queries"
              />
            </Grid>
            <Grid size={{ xs: 12, md: 6 }}>
              <SettingTextField
                label="DNS Server IP"
                value={dns.target_dns}
                onChange={(e) => onChange("dns.target_dns", e.target.value)}
                placeholder="e.g., 9.9.9.9"
                helperText="Select below or enter custom IP"
              />
            </Grid>

            <Grid size={{ xs: 12, md: 6 }}>
              {selectedServer && (
                <Box
                  sx={{
                    p: 2,
                    bgcolor: colors.background.paper,
                    borderRadius: 1,
                    border: `1px solid ${colors.border.default}`,
                    height: "100%",
                  }}
                >
                  <Stack direction="row" alignItems="center" spacing={1}>
                    <DnsIcon sx={{ color: colors.secondary }} />
                    <Typography variant="subtitle2">
                      {selectedServer.name}
                    </Typography>
                    {selectedServer.dnssec && (
                      <Chip
                        icon={<SecurityIcon />}
                        label="DNSSEC"
                        size="small"
                        sx={{
                          height: 20,
                          fontSize: "0.65rem",
                          bgcolor: `${colors.tertiary}22`,
                          color: colors.tertiary,
                          "& .MuiChip-icon": {
                            fontSize: 12,
                            color: colors.tertiary,
                          },
                        }}
                      />
                    )}
                  </Stack>
                  <Typography variant="caption" color="text.secondary">
                    {selectedServer.desc}
                  </Typography>
                </Box>
              )}
            </Grid>

            {/* DNS server list */}
            <Grid size={{ xs: 12 }}>
              <Typography variant="subtitle2" sx={{ mb: 1 }}>
                Recommended DNS Servers
              </Typography>
              <Box
                sx={{
                  border: `1px solid ${colors.border.default}`,
                  borderRadius: 1,
                  bgcolor: colors.background.paper,
                  maxHeight: 320,
                  overflow: "auto",
                }}
              >
                <List dense disablePadding>
                  {POPULAR_DNS.filter((server) =>
                    ipv6 ? server.ipv6 : !server.ipv6
                  ).map((server) => (
                    <ListItemButton
                      key={server.ip}
                      selected={dns.target_dns === server.ip}
                      onClick={() => handleServerSelect(server.ip)}
                      sx={{
                        borderLeft: server.warn
                          ? `3px solid ${colors.quaternary}`
                          : "3px solid transparent",
                        "&.Mui-selected": {
                          bgcolor: `${colors.secondary}22`,
                          borderLeftColor: colors.secondary,
                          "&:hover": { bgcolor: `${colors.secondary}33` },
                        },
                      }}
                    >
                      <ListItemIcon sx={{ minWidth: 36 }}>
                        {dns.target_dns === server.ip ? (
                          <CheckIcon
                            sx={{ color: colors.secondary, fontSize: 20 }}
                          />
                        ) : server.warn ? (
                          <BlockIcon
                            sx={{ color: colors.secondary, fontSize: 20 }}
                          />
                        ) : (
                          <DnsIcon
                            sx={{ color: colors.text.secondary, fontSize: 20 }}
                          />
                        )}
                      </ListItemIcon>
                      <ListItemText
                        primary={
                          <Stack
                            direction="row"
                            alignItems="center"
                            spacing={1}
                          >
                            <Typography
                              variant="body2"
                              sx={{
                                fontFamily: "monospace",
                                color: server.warn
                                  ? colors.secondary
                                  : "inherit",
                              }}
                            >
                              {server.name}
                            </Typography>
                            <Typography variant="body2" color="text.secondary">
                              {server.ip}
                            </Typography>
                            {server.tags.includes("fast") && (
                              <SpeedIcon
                                sx={{ fontSize: 14, color: colors.secondary }}
                              />
                            )}
                            {server.tags.includes("adblock") && (
                              <BlockIcon
                                sx={{ fontSize: 14, color: colors.secondary }}
                              />
                            )}
                          </Stack>
                        }
                        secondary={server.desc}
                        secondaryTypographyProps={{
                          variant: "caption",
                          sx: {
                            color: server.warn ? colors.secondary : undefined,
                          },
                        }}
                      />
                    </ListItemButton>
                  ))}
                </List>
              </Box>
            </Grid>

            {/* Visual explanation */}
            <Grid size={{ xs: 12 }}>
              <Box
                sx={{
                  p: 2,
                  bgcolor: colors.background.paper,
                  borderRadius: 1,
                  border: `1px solid ${colors.border.default}`,
                }}
              >
                <Typography
                  variant="caption"
                  color="text.secondary"
                  component="div"
                  sx={{ mb: 1 }}
                >
                  HOW IT WORKS
                </Typography>
                <Stack
                  direction="row"
                  alignItems="center"
                  spacing={1}
                  flexWrap="wrap"
                  useFlexGap
                >
                  <Chip
                    label="App"
                    size="small"
                    sx={{ bgcolor: colors.accent.primary }}
                  />
                  <Typography variant="caption">→ DNS query for</Typography>
                  <Chip
                    label="instagram.com"
                    size="small"
                    sx={{
                      bgcolor: colors.accent.secondary,
                      color: colors.secondary,
                    }}
                  />
                  <Typography variant="caption">→</Typography>
                  <Chip
                    label="poisoned DNS"
                    size="small"
                    sx={{
                      bgcolor: colors.quaternary,
                      textDecoration: "line-through",
                    }}
                  />
                  <Typography variant="caption">→</Typography>
                  <Chip
                    label={dns.target_dns || "select DNS"}
                    size="small"
                    sx={{
                      bgcolor: dns.target_dns
                        ? colors.tertiary
                        : colors.accent.primary,
                    }}
                  />
                  <Typography variant="caption">→ Real IP</Typography>
                </Stack>
              </Box>
            </Grid>

            {/* Warnings */}
            {!dns.target_dns && (
              <Grid size={{ xs: 12 }}>
                <Alert severity="warning">
                  Select or enter a DNS server IP to enable redirect.
                </Alert>
              </Grid>
            )}

            {dns.target_dns === "8.8.8.8" && (
              <Grid size={{ xs: 12 }}>
                <Alert severity="warning">
                  Google DNS (8.8.8.8) is commonly poisoned by Russian ISPs.
                  Consider Quad9 (9.9.9.9) or Cloudflare (1.1.1.1) instead.
                </Alert>
              </Grid>
            )}
          </>
        )}
      </Grid>
    </SettingSection>
  );
}
