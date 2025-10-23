import React from "react";
import { Grid } from "@mui/material";
import { ToggleOn as ToggleOnIcon } from "@mui/icons-material";
import SettingSection from "../../molecules/common/B4Section";
import SettingSwitch from "../../atoms/common/B4Switch";
import B4Config from "../../../models/Config";

interface FeatureSettingsProps {
  config: B4Config;
  onChange: (field: string, value: boolean) => void;
}

export const FeatureSettings: React.FC<FeatureSettingsProps> = ({
  config,
  onChange,
}) => {
  return (
    <SettingSection
      title="Feature Flags"
      description="Enable or disable advanced features"
      icon={<ToggleOnIcon />}
    >
      <Grid container spacing={2}>
        <Grid size={{ xs: 12, md: 6 }}>
          <SettingSwitch
            label="Generic Segmentation Offload (GSO)"
            checked={config.use_gso}
            onChange={(checked) => onChange("use_gso", checked)}
            description="Enable GSO for better performance"
          />
          <SettingSwitch
            label="Connection Tracking"
            checked={config.use_conntrack}
            onChange={(checked) => onChange("use_conntrack", checked)}
            description="Enable connection tracking (conntrack)"
          />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <SettingSwitch
            label="Skip IPTables Setup"
            checked={config.skip_iptables}
            onChange={(checked) => onChange("skip_iptables", checked)}
            description="Skip automatic iptables rules configuration"
          />
        </Grid>
      </Grid>
    </SettingSection>
  );
};
