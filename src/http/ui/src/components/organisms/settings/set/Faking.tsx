import React from "react";
import { Grid } from "@mui/material";
import { Security as SecurityIcon } from "@mui/icons-material";
import SettingSection from "@molecules/common/B4Section";
import SettingSelect from "@atoms/common/B4Select";
import SettingTextField from "@atoms/common/B4TextField";
import SettingSwitch from "@atoms/common/B4Switch";
import B4Slider from "@atoms/common/B4Slider";
import { B4SetConfig, FakingPayloadType } from "@models/Config";

interface FakingSettingsProps {
  config: B4SetConfig;
  onChange: (field: string, value: string | boolean | number) => void;
}

const FAKE_STRATEGIES = [
  { value: "ttl", label: "TTL" },
  { value: "randseq", label: "Random Sequence" },
  { value: "pastseq", label: "Past Sequence" },
  { value: "tcp_check", label: "TCP Check" },
  { value: "md5sum", label: "MD5 Sum" },
];

const FAKE_PAYLOAD_TYPES = [
  { value: 0, label: "Random" },
  { value: 1, label: "Custom" },
  { value: 2, label: "Default" },
];

export const FakingSettings: React.FC<FakingSettingsProps> = ({
  config,
  onChange,
}) => {
  return (
    <SettingSection
      title="Fake SNI Configuration"
      description="Configure fake SNI packets to confuse DPI"
      icon={<SecurityIcon />}
    >
      <Grid container spacing={2}>
        <Grid size={{ xs: 12 }}>
          <SettingSwitch
            label="Enable Fake SNI"
            checked={config.faking.sni}
            onChange={(checked) => onChange("faking.sni", checked)}
            description="Send fake SNI packets"
          />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <SettingSelect
            label="Fake Strategy"
            value={config.faking.strategy}
            options={FAKE_STRATEGIES}
            onChange={(e) =>
              onChange("faking.strategy", e.target.value as string)
            }
            helperText="Strategy for sending fake packets"
            disabled={!config.faking.sni}
          />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <SettingSelect
            label="Fake Payload Type"
            value={config.faking.sni_type}
            options={FAKE_PAYLOAD_TYPES}
            onChange={(e) =>
              onChange("faking.sni_type", Number(e.target.value))
            }
            helperText="Type of payload to send in fake packets"
            disabled={!config.faking.sni}
          />
        </Grid>
        <Grid size={{ xs: 12, md: 4 }}>
          <B4Slider
            label="Fake TTL"
            value={config.faking.ttl}
            onChange={(value) => onChange("faking.ttl", value)}
            min={1}
            max={64}
            step={1}
            helperText="TTL for fake packets"
            disabled={!config.faking.sni}
          />
        </Grid>
        <Grid size={{ xs: 12, md: 4 }}>
          <SettingTextField
            label="Sequence Offset"
            type="number"
            value={config.faking.seq_offset}
            onChange={(e) =>
              onChange("faking.seq_offset", Number(e.target.value))
            }
            helperText="Sequence number offset"
            disabled={!config.faking.sni}
          />
        </Grid>
        <Grid size={{ xs: 12, md: 4 }}>
          <B4Slider
            label="SNI Sequence Length"
            value={config.faking.sni_seq_length}
            onChange={(value) => onChange("faking.sni_seq_length", value)}
            min={1}
            max={20}
            step={1}
            helperText="Length of fake SNI sequence"
            disabled={!config.faking.sni}
          />
        </Grid>
        {config.faking.sni_type === FakingPayloadType.CUSTOM && (
          <Grid size={{ xs: 12 }}>
            <SettingTextField
              label="Custom Payload"
              value={config.faking.custom_payload}
              onChange={(e) =>
                onChange("faking.custom_payload", e.target.value)
              }
              helperText="Custom payload for fake packets (hex string)"
              disabled={!config.faking.sni}
              multiline
              rows={2}
            />
          </Grid>
        )}
      </Grid>
    </SettingSection>
  );
};
