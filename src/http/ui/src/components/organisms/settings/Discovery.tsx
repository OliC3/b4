import React from "react";
import {
  Box,
  Chip,
  Divider,
  Grid,
  IconButton,
  Typography,
} from "@mui/material";
import { Science as TestIcon, Add as AddIcon } from "@mui/icons-material";
import SettingSection from "@molecules/common/B4Section";
import B4Slider from "@atoms/common/B4Slider";
import B4TextField from "@atoms/common/B4TextField";
import { B4Config } from "@models/Config";
import { colors } from "@design";

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
  const [newDns, setNewDns] = React.useState("");

  const handleAddDns = () => {
    if (newDns.trim()) {
      const current = config.system.checker.reference_dns || [];
      if (!current.includes(newDns.trim())) {
        onChange("system.checker.reference_dns", [...current, newDns.trim()]);
      }
      setNewDns("");
    }
  };

  const handleRemoveDns = (dns: string) => {
    const current = config.system.checker.reference_dns || [];
    onChange(
      "system.checker.reference_dns",
      current.filter((s) => s !== dns)
    );
  };

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
            value={config.system.checker.reference_domain || "yandex.ru"}
            onChange={(e) =>
              onChange("system.checker.reference_domain", e.target.value)
            }
            placeholder="yandex.ru"
            helperText="Fast domain to measure your network baseline speed"
          />
        </Grid>
        <Grid size={{ xs: 12 }}>
          <Divider sx={{ my: 1 }}>
            <Chip label="DNS Configuration" size="small" />
          </Divider>
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <Box sx={{ display: "flex", gap: 1, alignItems: "flex-start" }}>
            <B4TextField
              label="Add DNS Server"
              value={newDns}
              onChange={(e) => setNewDns(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  handleAddDns();
                }
              }}
              placeholder="e.g., 8.8.8.8"
              helperText="Additional DNS servers to test"
            />
            <IconButton
              onClick={handleAddDns}
              sx={{
                bgcolor: colors.accent.secondary,
                color: colors.secondary,
                "&:hover": { bgcolor: colors.accent.secondaryHover },
              }}
            >
              <AddIcon />
            </IconButton>
          </Box>
        </Grid>
        {(config.system.checker.reference_dns?.length ?? 0) > 0 && (
          <Grid size={{ xs: 12, md: 6 }}>
            <Typography variant="subtitle2" gutterBottom>
              Active DNS servers to test:
            </Typography>
            <Box
              sx={{
                display: "flex",
                flexWrap: "wrap",
                gap: 1,
                p: 1,
                border: `1px solid ${colors.border.default}`,
                borderRadius: 1,
                bgcolor: colors.background.paper,
              }}
            >
              {config.system.checker.reference_dns.map((dns) => (
                <Chip
                  key={dns}
                  label={dns}
                  onDelete={() => handleRemoveDns(dns)}
                  size="small"
                  sx={{
                    bgcolor: colors.accent.primary,
                    color: colors.secondary,
                    "& .MuiChip-deleteIcon": {
                      color: colors.secondary,
                    },
                  }}
                />
              ))}
            </Box>
          </Grid>
        )}
      </Grid>
    </SettingSection>
  );
};
