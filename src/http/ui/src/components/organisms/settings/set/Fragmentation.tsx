import React from "react";
import { Grid } from "@mui/material";
import { CallSplit as CallSplitIcon } from "@mui/icons-material";
import SettingSection from "@molecules/common/B4Section";
import SettingSelect from "@atoms/common/B4Select";
import SettingSwitch from "@atoms/common/B4Switch";
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
  return (
    <SettingSection
      title="Fragmentation Strategy"
      description="Configure packet fragmentation for DPI circumvention"
      icon={<CallSplitIcon />}
    >
      <Grid container spacing={3}>
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
            helperText="Position where to fragment SNI"
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
      </Grid>
    </SettingSection>
  );
};
