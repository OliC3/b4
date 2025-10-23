import React from "react";
import { Grid } from "@mui/material";
import { Settings as SettingsIcon } from "@mui/icons-material";
import SettingSection from "../../molecules/common/B4Section";
import SettingTextField from "../../atoms/common/B4TextField";
import B4Config from "../../../models/Config";

interface NetworkSettingsProps {
  config: B4Config;
  onChange: (field: string, value: number) => void;
}

export const NetworkSettings: React.FC<NetworkSettingsProps> = ({
  config,
  onChange,
}) => {
  return (
    <SettingSection
      title="Network Configuration"
      description="Configure netfilter queue and network processing parameters"
      icon={<SettingsIcon />}
    >
      <Grid container spacing={2}>
        <Grid size={{ xs: 12, md: 6 }}>
          <SettingTextField
            label="Queue Start Number"
            type="number"
            value={config.queue_start_num}
            onChange={(e) => onChange("queueStartNum", Number(e.target.value))}
            helperText="Netfilter queue number to use (0-65535)"
          />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <SettingTextField
            label="Worker Threads"
            type="number"
            value={config.threads}
            onChange={(e) => onChange("threads", Number(e.target.value))}
            helperText="Number of worker threads (minimum 1)"
          />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <SettingTextField
            label="Packet Mark"
            type="number"
            value={config.mark}
            onChange={(e) => onChange("mark", Number(e.target.value))}
            helperText="Packet mark value for iptables"
          />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <SettingTextField
            label="Connection Bytes Limit"
            type="number"
            value={config.conn_bytes_limit}
            onChange={(e) => onChange("connBytesLimit", Number(e.target.value))}
            helperText="Connection bytes limit"
          />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <SettingTextField
            label="Segment 2 Delay (ms)"
            type="number"
            value={config.seg2delay}
            onChange={(e) => onChange("seg2Delay", Number(e.target.value))}
            helperText="Delay between segments in milliseconds"
          />
        </Grid>
      </Grid>
    </SettingSection>
  );
};
