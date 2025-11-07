import React from "react";
import { Grid } from "@mui/material";
import { Description as DescriptionIcon } from "@mui/icons-material";
import SettingSection from "@molecules/common/B4Section";
import SettingSelect from "@atoms/common/B4Select";
import SettingSwitch from "@atoms/common/B4Switch";
import { B4Config, LogLevel } from "@models/Config";

interface LoggingSettingsProps {
  config: B4Config;
  onChange: (field: string, value: number | boolean) => void;
}

const LOG_LEVELS: Array<{ value: LogLevel; label: string }> = [
  { value: LogLevel.ERROR, label: "Error" },
  { value: LogLevel.INFO, label: "Info" },
  { value: LogLevel.TRACE, label: "Trace" },
  { value: LogLevel.DEBUG, label: "Debug" },
] as const;

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
            value={config.system.logging.level}
            options={LOG_LEVELS}
            onChange={(e) =>
              onChange("system.logging.level", Number(e.target.value))
            }
            helperText="Set the verbosity of logging output"
          />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <SettingSwitch
            label="Instant Flush"
            checked={config?.system?.logging?.instaflush}
            onChange={(checked: boolean) =>
              onChange("system.logging.instaflush", Boolean(checked))
            }
            description="Flush logs immediately (may impact performance)"
          />
          <SettingSwitch
            label="Syslog"
            checked={config?.system?.logging?.syslog}
            onChange={(checked: boolean) =>
              onChange("system.logging.syslog", Boolean(checked))
            }
            description="Enable syslog output"
          />
        </Grid>
      </Grid>
    </SettingSection>
  );
};
