import React from "react";
import { Grid } from "@mui/material";
import { Dns as DnsIcon } from "@mui/icons-material";
import SettingSection from "../../molecules/common/B4Section";
import SettingSelect from "../../atoms/common/B4Select";
import SettingTextField from "../../atoms/common/B4TextField";
import B4Config from "../../../models/Config";

interface UDPSettingsProps {
  config: B4Config;
  onChange: (field: string, value: string | number) => void;
}

const UDP_MODES = [
  { value: "drop", label: "Drop" },
  { value: "fake", label: "Fake" },
];

const UDP_FAKING_STRATEGIES = [
  { value: "none", label: "None" },
  { value: "ttl", label: "TTL" },
  { value: "checksum", label: "Checksum" },
];

const UDP_QUIC_FILTERS = [
  { value: "disabled", label: "Disabled" },
  { value: "all", label: "All" },
  { value: "parse", label: "Parse" },
];

export const UDPSettings: React.FC<UDPSettingsProps> = ({
  config,
  onChange,
}) => {
  return (
    <SettingSection
      title="UDP Configuration"
      description="Configure UDP packet handling and QUIC filtering"
      icon={<DnsIcon />}
    >
      <Grid container spacing={2}>
        <Grid size={{ xs: 12, md: 6 }}>
          <SettingSelect
            label="UDP Mode"
            value={config.udp.mode}
            options={UDP_MODES}
            onChange={(e) => onChange("udp.mode", e.target.value as string)}
            helperText="UDP packet handling strategy"
          />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <SettingSelect
            label="QUIC Filter"
            value={config.udp.filter_quic}
            options={UDP_QUIC_FILTERS}
            onChange={(e) =>
              onChange("udp.filter_quic", e.target.value as string)
            }
            helperText="QUIC traffic filtering mode"
          />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <SettingSelect
            label="Faking Strategy"
            value={config.udp.faking_strategy}
            options={UDP_FAKING_STRATEGIES}
            onChange={(e) =>
              onChange("udp.faking_strategy", e.target.value as string)
            }
            helperText="Strategy for fake UDP packets"
          />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <SettingTextField
            label="Fake Packet Size"
            type="number"
            value={config.udp.fake_len}
            onChange={(e) => onChange("udp.fake_len", Number(e.target.value))}
            helperText="Size of fake UDP packets in bytes"
          />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <SettingTextField
            label="Fake Sequence Length"
            type="number"
            value={config.udp.fake_seq_length}
            onChange={(e) =>
              onChange("udp.fake_seq_length", Number(e.target.value))
            }
            helperText="Number of fake packets to send"
          />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <Grid container spacing={1}>
            <Grid size={6}>
              <SettingTextField
                label="Dest Port Min"
                type="number"
                value={config.udp.dport_min}
                onChange={(e) =>
                  onChange("udp.dport_min", Number(e.target.value))
                }
                helperText="Minimum destination port"
              />
            </Grid>
            <Grid size={6}>
              <SettingTextField
                label="Dest Port Max"
                type="number"
                value={config.udp.dport_max}
                onChange={(e) =>
                  onChange("udp.dport_max", Number(e.target.value))
                }
                helperText="Maximum destination port"
              />
            </Grid>
          </Grid>
        </Grid>
      </Grid>
    </SettingSection>
  );
};
