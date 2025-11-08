import React from "react";
import { Grid } from "@mui/material";
import { Dns as DnsIcon } from "@mui/icons-material";
import SettingSection from "@molecules/common/B4Section";
import B4Slider from "@atoms/common/B4Slider";
import { B4SetConfig } from "@models/Config";

interface TcpSettingsProps {
  config: B4SetConfig;
  onChange: (field: string, value: string | number) => void;
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
      </Grid>
    </SettingSection>
  );
};
