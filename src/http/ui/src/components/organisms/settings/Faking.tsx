import React from "react";
import { Grid } from "@mui/material";
import { Security as SecurityIcon } from "@mui/icons-material";
import SettingSection from "@molecules/common/B4Section";
import SettingSelect from "@atoms/common/B4Select";
import SettingTextField from "@atoms/common/B4TextField";
import SettingSwitch from "@atoms/common/B4Switch";
import B4Config, { FakingPayloadType } from "@models/Config";

interface FakingSettingsProps {
  config: B4Config;
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
            checked={config.bypass.faking.sni}
            onChange={(checked) => onChange("bypass.faking.sni", checked)}
            description="Send fake SNI packets"
          />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <SettingSelect
            label="Fake Strategy"
            value={config.bypass.faking.strategy}
            options={FAKE_STRATEGIES}
            onChange={(e) =>
              onChange("bypass.faking.strategy", e.target.value as string)
            }
            helperText="Strategy for sending fake packets"
            disabled={!config.bypass.faking.sni}
          />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <SettingSelect
            label="Fake Payload Type"
            value={config.bypass.faking.sni_type}
            options={FAKE_PAYLOAD_TYPES}
            onChange={(e) =>
              onChange("bypass.faking.sni_type", Number(e.target.value))
            }
            helperText="Type of payload to send in fake packets"
            disabled={!config.bypass.faking.sni}
          />
        </Grid>
        <Grid size={{ xs: 12, md: 4 }}>
          <SettingTextField
            label="Fake TTL"
            type="number"
            value={config.bypass.faking.ttl}
            onChange={(e) =>
              onChange("bypass.faking.ttl", Number(e.target.value))
            }
            helperText="TTL for fake packets"
            disabled={!config.bypass.faking.sni}
          />
        </Grid>
        <Grid size={{ xs: 12, md: 4 }}>
          <SettingTextField
            label="Sequence Offset"
            type="number"
            value={config.bypass.faking.seq_offset}
            onChange={(e) =>
              onChange("bypass.faking.seq_offset", Number(e.target.value))
            }
            helperText="Sequence number offset"
            disabled={!config.bypass.faking.sni}
          />
        </Grid>
        <Grid size={{ xs: 12, md: 4 }}>
          <SettingTextField
            label="SNI Sequence Length"
            type="number"
            value={config.bypass.faking.sni_seq_length}
            onChange={(e) =>
              onChange("bypass.faking.sni_seq_length", Number(e.target.value))
            }
            helperText="Length of fake SNI sequence"
            disabled={!config.bypass.faking.sni}
          />
        </Grid>
        {config.bypass.faking.sni_type === FakingPayloadType.CUSTOM && (
          <Grid size={{ xs: 12 }}>
            <SettingTextField
              label="Custom Payload"
              value={config.bypass.faking.custom_payload}
              onChange={(e) =>
                onChange("bypass.faking.custom_payload", e.target.value)
              }
              helperText="Custom payload for fake packets (hex string)"
              disabled={!config.bypass.faking.sni}
              multiline
              rows={2}
            />
          </Grid>
        )}
      </Grid>
    </SettingSection>
  );
};
