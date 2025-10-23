import React from "react";
import { Grid } from "@mui/material";
import { Description as DescriptionIcon } from "@mui/icons-material";
import SettingSection from "../../molecules/common/B4Section";
import SettingSelect from "../../atoms/common/B4Select";
import SettingSwitch from "../../atoms/common/B4Switch";
import B4Config from "../../../models/Config";

interface LoggingSettingsProps {
  config: B4Config;
  onChange: (field: string, value: number | boolean) => void;
}

const LOG_LEVELS = [
  { value: -1, label: "Silent" },
  { value: 0, label: "Error" },
  { value: 1, label: "Info" },
  { value: 2, label: "Debug" },
  { value: 3, label: "Trace" },
];

export const LoggingSettings: React.FC<LoggingSettingsProps> = ({
  config,
  onChange,
}) => {
  return (
    <SettingSection
      title="Logging Configuration"
      description="Configure logging behavior and output"
      icon={<DescriptionIcon />}
    >
      <Grid container spacing={2}>
        <Grid size={{ xs: 12, md: 6 }}>
          <SettingSelect
            label="Log Level"
            value={config.logging.level}
            options={LOG_LEVELS}
            onChange={(e) => onChange("level", Number(e.target.value))}
            helperText="Set the verbosity of logging output"
          />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <SettingSwitch
            label="Instant Flush"
            checked={config.logging.instaflush}
            onChange={(checked) => onChange("instaflush", checked)}
            description="Flush logs immediately (may impact performance)"
          />
          <SettingSwitch
            label="Syslog"
            checked={config.logging.syslog}
            onChange={(checked) => onChange("syslog", checked)}
            description="Enable syslog output"
          />
        </Grid>
      </Grid>
    </SettingSection>
  );
};
