import { Grid } from "@mui/material";
import { Science as TestIcon } from "@mui/icons-material";
import SettingSection from "@molecules/common/B4Section";
import B4Slider from "@atoms/common/B4Slider";
import B4TextField from "@atoms/common/B4TextField";
import { B4Config } from "@models/Config";

interface CheckerSettingsProps {
  config: B4Config;
  onChange: (
    field: string,
    value: string | boolean | number | string[]
  ) => void;
}

export const CheckerSettings: React.FC<CheckerSettingsProps> = ({
  config,
  onChange,
}) => {
  return (
    <SettingSection
      title="Testing Configuration"
      description="Configure testing behavior and output"
      icon={<TestIcon />}
    >
      <Grid container spacing={2}>
        <Grid size={{ xs: 12, lg: 6 }}>
          <B4Slider
            label="Discovery Timeout"
            value={config.system.checker.discovery_timeout || 5}
            onChange={(value) =>
              onChange("system.checker.discovery_timeout", value)
            }
            min={3}
            max={30}
            step={1}
            valueSuffix=" sec"
            helperText="Timeout per preset during discovery"
          />
        </Grid>
        <Grid size={{ xs: 12, lg: 6 }}>
          <B4Slider
            label="Config Propagation Delay"
            value={config.system.checker.config_propagate_ms || 1500}
            onChange={(value) =>
              onChange("system.checker.config_propagate_ms", value)
            }
            min={500}
            max={5000}
            step={100}
            valueSuffix=" ms"
            helperText="Delay for config to propagate to workers (increase on slow devices)"
          />
        </Grid>
        <Grid size={{ xs: 12, lg: 6 }}>
          <B4TextField
            label="Reference Domain"
            value={config.system.checker.reference_domain || "max.ru"}
            onChange={(e) =>
              onChange("system.checker.reference_domain", e.target.value)
            }
            placeholder="max.ru"
            helperText="Fast domain to measure your network baseline speed"
          />
        </Grid>
      </Grid>
    </SettingSection>
  );
};
