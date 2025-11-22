import React from "react";
import { Grid, Alert, Divider, Chip, Typography } from "@mui/material";
import { CallSplit as CallSplitIcon } from "@mui/icons-material";
import SettingSection from "@molecules/common/B4Section";
import SettingSelect from "@atoms/common/B4Select";
import SettingSwitch from "@atoms/common/B4Switch";
import B4TextField from "@atoms/common/B4TextField";
import B4Slider from "@atoms/common/B4Slider";
import { B4SetConfig, FragmentationStrategy } from "@models/Config";

interface FragmentationSettingsProps {
  config: B4SetConfig;
  onChange: (field: string, value: string | boolean | number) => void;
}

const fragmentationOptions: { label: string; value: FragmentationStrategy }[] =
  [
    { label: "TCP Fragmentation", value: "tcp" },
    { label: "IP Fragmentation", value: "ip" },
    { label: "No Fragmentation", value: "none" },
  ];

export const FragmentationSettings: React.FC<FragmentationSettingsProps> = ({
  config,
  onChange,
}) => {
  const hasOOB = (config.fragmentation.oob_position || 0) > 0;

  return (
    <SettingSection
      title="Fragmentation & OOB Strategy"
      description="Configure packet fragmentation and Out-of-Band data for DPI circumvention"
      icon={<CallSplitIcon />}
    >
      <Grid container spacing={3}>
        {/* Section 1: Basic Fragmentation */}
        <Grid size={{ xs: 12 }}>
          <Divider sx={{ mb: 2 }}>
            <Chip label="Fragmentation Settings" size="small" />
          </Divider>
        </Grid>

        <Grid size={{ xs: 12, md: 6 }}>
          <SettingSelect
            label="Fragment Strategy"
            value={config.fragmentation.strategy}
            options={fragmentationOptions}
            onChange={(e) =>
              onChange("fragmentation.strategy", e.target.value as string)
            }
            helperText="Choose fragmentation method"
          />
        </Grid>

        <Grid size={{ xs: 12, md: 6 }}>
          <B4Slider
            label="SNI Fragment Position"
            value={config.fragmentation.sni_position}
            onChange={(value) => onChange("fragmentation.sni_position", value)}
            min={0}
            max={10}
            step={1}
            helperText="Position where to fragment SNI (0=auto)"
          />
        </Grid>

        <Grid size={{ xs: 12, md: 6 }}>
          <SettingSwitch
            label="Reverse Fragment Order"
            checked={config.fragmentation.sni_reverse}
            onChange={(checked) =>
              onChange("fragmentation.sni_reverse", checked)
            }
            description="Send fragments in reverse order"
          />
        </Grid>

        <Grid size={{ xs: 12, md: 6 }}>
          <SettingSwitch
            label="Fragment in Middle of SNI"
            checked={config.fragmentation.middle_sni}
            onChange={(checked) =>
              onChange("fragmentation.middle_sni", checked)
            }
            description="Fragment in the middle of the SNI field"
          />
        </Grid>

        {/* Section 2: OOB (Out-of-Band) Settings */}
        {/* OOB Settings - simplified to match SNI pattern */}
        <Grid size={{ xs: 12 }}>
          <Divider sx={{ my: 3 }}>
            <Chip label="OOB (Out-of-Band) Settings" size="small" />
          </Divider>
        </Grid>

        <Grid size={{ xs: 12, md: 4 }}>
          <B4Slider
            label="OOB Position"
            value={config.fragmentation.oob_position || 0}
            onChange={(value) => onChange("fragmentation.oob_position", value)}
            min={0}
            max={10}
            step={1}
            helperText="Split position with URG flag (0=disabled)"
            valueSuffix={config.fragmentation.oob_position > 0 ? " bytes" : ""}
          />
        </Grid>

        <Grid size={{ xs: 12, md: 4 }}>
          <SettingSwitch
            label="OOB Reverse Order"
            checked={config.fragmentation.oob_reverse || false}
            onChange={(checked) =>
              onChange("fragmentation.oob_reverse", checked)
            }
            description="Send OOB segments in reverse (like -q flag)"
            disabled={!hasOOB}
          />
        </Grid>

        <Grid size={{ xs: 12, md: 4 }}>
          <B4TextField
            label="OOB Character"
            value={config.fragmentation.oob_char || "x"}
            onChange={(e) => {
              const char = e.target.value.slice(0, 1);
              onChange("fragmentation.oob_char", char || "x");
            }}
            placeholder="x"
            helperText="Character to append as OOB data"
            inputProps={{ maxLength: 1 }}
            disabled={!hasOOB}
          />
        </Grid>

        {hasOOB && (
          <Grid size={{ xs: 12 }}>
            <Alert severity="success">
              <Typography variant="body2">
                <strong>OOB Active!</strong> Position:{" "}
                {config.fragmentation.oob_position} byte(s)
                {config.fragmentation.oob_reverse
                  ? " (reverse order)"
                  : " (normal order)"}
              </Typography>
            </Alert>
          </Grid>
        )}
      </Grid>
    </SettingSection>
  );
};
