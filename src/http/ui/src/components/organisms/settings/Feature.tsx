import React from "react";
import { Grid } from "@mui/material";
import { ToggleOn as ToggleOnIcon } from "@mui/icons-material";
import SettingSection from "@molecules/common/B4Section";
import SettingSwitch from "@atoms/common/B4Switch";
import B4Config from "@models/Config";

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
            label="Enable IPv4 Support"
            checked={config.queue.ipv4}
            onChange={(checked) => onChange("queue.ipv4", checked)}
            description="Enable IPv4 support"
          />
          <SettingSwitch
            label="Enable IPv6 Support"
            checked={config.queue.ipv6}
            onChange={(checked) => onChange("queue.ipv6", checked)}
            description="Enable IPv6 support"
          />
          <SettingSwitch
            label="Skip IPTables/NFTables Setup"
            checked={config.system.tables.skip_setup}
            onChange={(checked) =>
              onChange("system.tables.skip_setup", checked)
            }
            description="Skip automatic IPTables/NFTables rules configuration"
          />
        </Grid>
      </Grid>
    </SettingSection>
  );
};
