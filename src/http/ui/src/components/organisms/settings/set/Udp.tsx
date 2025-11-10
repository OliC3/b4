import React from "react";
import { Grid, Alert, Divider, Chip, Box } from "@mui/material";
import {
  Dns as DnsIcon,
  Warning as WarningIcon,
  Info as InfoIcon,
} from "@mui/icons-material";
import SettingSection from "@molecules/common/B4Section";
import SettingSelect from "@atoms/common/B4Select";
import SettingTextField from "@atoms/common/B4TextField";
import B4Slider from "@atoms/common/B4Slider";
import B4Switch from "@/components/atoms/common/B4Switch";
import { B4SetConfig } from "@models/Config";

interface UdpSettingsProps {
  config: B4SetConfig;
  onChange: (field: string, value: string | boolean | number) => void;
}

const UDP_MODES = [
  {
    value: "drop",
    label: "Drop",
    description: "Drop matched UDP packets (forces TCP fallback)",
  },
  {
    value: "fake",
    label: "Fake & Fragment",
    description: "Send fake packets and fragment real ones (DPI bypass)",
  },
];

const UDP_QUIC_FILTERS = [
  {
    value: "disabled",
    label: "Disabled",
    description: "Don't process QUIC at all",
  },
  {
    value: "all",
    label: "All QUIC",
    description: "Match all QUIC Initial packets (blind matching)",
  },
  {
    value: "parse",
    label: "Parse SNI",
    description: "Match only QUIC with SNI in domain list (smart matching)",
  },
];

const UDP_FAKING_STRATEGIES = [
  { value: "none", label: "None", description: "No faking strategy" },
  {
    value: "ttl",
    label: "TTL",
    description: "Use low TTL to make packets expire",
  },
  { value: "checksum", label: "Checksum", description: "Corrupt UDP checksum" },
];

export const UdpSettings: React.FC<UdpSettingsProps> = ({
  config,
  onChange,
}) => {
  // Detect configuration state
  const isQuicEnabled = config.udp.filter_quic !== "disabled";
  const hasPortFilter =
    config.udp.dport_filter && config.udp.dport_filter.trim() !== "";
  const hasDomainsConfigured =
    config.targets?.sni_domains?.length > 0 ||
    config.targets?.geosite_categories?.length > 0;

  // Will any UDP traffic be processed?
  const willProcessUdp = isQuicEnabled || hasPortFilter;

  // Should we show action settings?
  const showActionSettings = willProcessUdp;

  // Should we show fake-specific settings?
  const isFakeMode = config.udp.mode === "fake";
  const showFakeSettings = showActionSettings && isFakeMode;

  // Warnings
  const showParseWarning =
    config.udp.filter_quic === "parse" && !hasDomainsConfigured;
  const showNoProcessingWarning = !willProcessUdp;

  return (
    <SettingSection
      title="UDP & QUIC Configuration"
      description="Configure UDP packet processing and QUIC filtering"
      icon={<DnsIcon />}
    >
      <Grid container spacing={3}>
        {/* Section 1: Matching Rules */}
        <Grid size={{ xs: 12 }}>
          <Divider sx={{ mb: 2 }}>
            <Chip label="What UDP Traffic to Process" size="small" />
          </Divider>
        </Grid>

        {/* QUIC Filter */}
        <Grid size={{ xs: 12, md: 6 }}>
          <SettingSelect
            label="QUIC Filter"
            value={config.udp.filter_quic}
            options={UDP_QUIC_FILTERS}
            onChange={(e) =>
              onChange("udp.filter_quic", e.target.value as string)
            }
            helperText={
              UDP_QUIC_FILTERS.find((o) => o.value === config.udp.filter_quic)
                ?.description
            }
          />
        </Grid>

        {/* Port Filter */}
        <Grid size={{ xs: 12, md: 6 }}>
          <SettingTextField
            label="Port Filter"
            value={config.udp.dport_filter}
            onChange={(e) => onChange("udp.dport_filter", e.target.value)}
            placeholder="e.g., 5000-6000,8000"
            helperText="Match specific UDP ports (VoIP, gaming, etc.) - leave empty to disable"
          />
        </Grid>

        {/* STUN Filter */}
        <Grid size={{ xs: 12, md: 6 }}>
          <B4Switch
            label="Filter STUN Packets"
            checked={config.udp.filter_stun}
            onChange={(checked) => onChange("udp.filter_stun", checked)}
            description="Ignore STUN packets (recommended for voice/video calls)"
          />
        </Grid>

        {/* Parse mode warning */}
        {showParseWarning && (
          <Grid size={{ xs: 12 }}>
            <Alert severity="warning" icon={<WarningIcon />}>
              <strong>Parse mode requires domains:</strong> Add domains in the
              Targets section for SNI matching to work. Without domains, no QUIC
              traffic will be processed.
            </Alert>
          </Grid>
        )}

        {/* No processing warning */}
        {showNoProcessingWarning && (
          <Grid size={{ xs: 12 }}>
            <Alert severity="info" icon={<InfoIcon />}>
              <strong>UDP processing disabled:</strong> Enable QUIC filtering or
              add port filters to process UDP traffic. Currently, all UDP
              packets will pass through unchanged.
            </Alert>
          </Grid>
        )}

        {/* Section 2: Action Settings (only if traffic will be processed) */}
        {showActionSettings && (
          <>
            <Grid size={{ xs: 12 }}>
              <Divider sx={{ my: 2 }}>
                <Chip label="How to Handle Matched Traffic" size="small" />
              </Divider>
            </Grid>

            {/* UDP Mode */}
            <Grid size={{ xs: 12, md: 6 }}>
              <SettingSelect
                label="Action Mode"
                value={config.udp.mode}
                options={UDP_MODES}
                onChange={(e) => onChange("udp.mode", e.target.value as string)}
                helperText={
                  UDP_MODES.find((o) => o.value === config.udp.mode)
                    ?.description
                }
              />
            </Grid>

            {/* Connection Packets Limit */}
            <Grid size={{ xs: 12, md: 6 }}>
              <B4Slider
                label="Connection Packets Limit"
                value={config.udp.conn_bytes_limit}
                onChange={(value) => onChange("udp.conn_bytes_limit", value)}
                min={1}
                max={20}
                step={1}
                helperText="Process only first N packets per connection (recommended: 5-8)"
              />
            </Grid>

            {/* Info about current mode */}
            <Grid size={{ xs: 12 }}>
              <Alert severity="info" icon={<InfoIcon />}>
                {isFakeMode ? (
                  <>
                    <strong>Fake mode:</strong> Matched UDP packets will be
                    preceded by fake packets and fragmented to bypass DPI
                    systems. Configure fake packet settings below.
                  </>
                ) : (
                  <>
                    <strong>Drop mode:</strong> Matched UDP packets will be
                    dropped, forcing the application to fall back to TCP (e.g.,
                    QUIC → HTTPS).
                  </>
                )}
              </Alert>
            </Grid>
          </>
        )}

        {/* Section 3: Fake Mode Settings (only if fake mode is enabled) */}
        {showFakeSettings && (
          <>
            <Grid size={{ xs: 12 }}>
              <Divider sx={{ my: 2 }}>
                <Chip label="Fake Packet Configuration" size="small" />
              </Divider>
            </Grid>

            <Grid size={{ xs: 12, md: 6 }}>
              <SettingSelect
                label="Faking Strategy"
                value={config.udp.faking_strategy}
                options={UDP_FAKING_STRATEGIES}
                onChange={(e) =>
                  onChange("udp.faking_strategy", e.target.value as string)
                }
                helperText={
                  UDP_FAKING_STRATEGIES.find(
                    (o) => o.value === config.udp.faking_strategy
                  )?.description
                }
              />
            </Grid>

            <Grid size={{ xs: 12, md: 6 }}>
              <B4Slider
                label="Fake Packet Count"
                value={config.udp.fake_seq_length}
                onChange={(value) => onChange("udp.fake_seq_length", value)}
                min={1}
                max={20}
                step={1}
                helperText="Number of fake packets sent before real packet"
              />
            </Grid>

            <Grid size={{ xs: 12, md: 6 }}>
              <B4Slider
                label="Fake Packet Size"
                value={config.udp.fake_len}
                onChange={(value) => onChange("udp.fake_len", value)}
                min={32}
                max={1500}
                step={8}
                valueSuffix=" bytes"
                helperText="Size of each fake UDP packet payload"
              />
            </Grid>
          </>
        )}

        {/* Overall explanation */}
        <Grid size={{ xs: 12 }}>
          <Box
            sx={{ mt: 2, p: 2, bgcolor: "background.paper", borderRadius: 1 }}
          >
            <Alert
              severity="info"
              icon={<InfoIcon />}
              sx={{ bgcolor: "transparent" }}
            >
              <strong>How UDP processing works:</strong>
              <br />
              1. Packets are matched by QUIC filter, port filter, or IP targets
              <br />
              2. STUN packets are excluded if STUN filtering is enabled
              <br />
              3. Matched packets are handled according to the action mode
              <br />
              4. Only the first N packets of each connection are processed
            </Alert>
          </Box>
        </Grid>
      </Grid>
    </SettingSection>
  );
};
