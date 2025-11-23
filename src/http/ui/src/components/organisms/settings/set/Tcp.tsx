import React from "react";
import { Grid, FormControlLabel, Switch, Typography } from "@mui/material";
import { Dns as DnsIcon } from "@mui/icons-material";
import SettingSection from "@molecules/common/B4Section";
import B4Slider from "@atoms/common/B4Slider";
import { B4SetConfig } from "@models/Config";

interface TcpSettingsProps {
  config: B4SetConfig;
  onChange: (field: string, value: string | number | boolean) => void;
}

export const TcpSettings: React.FC<TcpSettingsProps> = ({
  config,
  onChange,
}) => {
  return (
    <SettingSection
      title="TCP Configuration"
      description="Configure TCP packet handling"
      icon={<DnsIcon />}
    >
      <Grid container spacing={3}>
        <Grid size={{ xs: 12, md: 6 }}>
          <B4Slider
            label="Connection Bytes Limit"
            value={config.tcp.conn_bytes_limit}
            onChange={(value) => onChange("tcp.conn_bytes_limit", value)}
            min={1}
            max={100}
            step={1}
            helperText="Bytes to analyze before applying bypass"
          />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <B4Slider
            label="Segment 2 Delay"
            value={config.tcp.seg2delay}
            onChange={(value) => onChange("tcp.seg2delay", value)}
            min={0}
            max={1000}
            step={10}
            valueSuffix=" ms"
            helperText="Delay between segments"
          />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <FormControlLabel
            control={
              <Switch
                checked={config.tcp.drop_sack || false}
                onChange={(e) => onChange("tcp.drop_sack", e.target.checked)}
                color="primary"
              />
            }
            label={
              <div>
                <Typography variant="body1" fontWeight={500}>
                  Drop SACK Options
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  Strip Selective Acknowledgment from TCP headers to confuse
                  stateful DPI
                </Typography>
              </div>
            }
          />
        </Grid>
        {/* SYN Fake Settings */}
        <Grid size={{ xs: 12, md: 6 }}>
          <FormControlLabel
            control={
              <Switch
                checked={config.tcp.syn_fake || false}
                onChange={(e) => onChange("tcp.syn_fake", e.target.checked)}
                color="primary"
              />
            }
            label={
              <div>
                <Typography variant="body1" fontWeight={500}>
                  SYN Fake Packets
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  Send fake SYN packets during TCP handshake (aggressive)
                </Typography>
              </div>
            }
          />
        </Grid>

        <Grid size={{ xs: 12, md: 6 }}>
          <B4Slider
            label="SYN Fake Payload Length"
            value={config.tcp.syn_fake_len || 0}
            onChange={(value) => onChange("tcp.syn_fake_len", value)}
            min={0}
            max={1200}
            step={64}
            disabled={!config.tcp.syn_fake}
            helperText={
              config.tcp.syn_fake
                ? "Fake payload size (0 = use full fake packet)"
                : "Enable SYN Fake to configure length"
            }
          />
        </Grid>
      </Grid>
    </SettingSection>
  );
};
