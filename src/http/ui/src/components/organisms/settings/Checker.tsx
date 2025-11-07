import React, { useState } from "react";
import { Box, Chip, Grid, IconButton, Typography } from "@mui/material";
import {
  Science as TestIcon,
  Domain as DomainIcon,
  Add as AddIcon,
} from "@mui/icons-material";
import SettingSection from "@molecules/common/B4Section";
import SettingTextField from "@atoms/common/B4TextField";
import { colors } from "@design";
import B4Config from "@models/Config";

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
  const [newDomain, setNewDomain] = useState("");

  const handleAddDomain = () => {
    if (newDomain.trim()) {
      onChange("system.checker.domains", [
        ...config.system.checker.domains,
        newDomain.trim(),
      ]);
      setNewDomain("");
    }
  };

  const handleRemoveDomain = (domain: string) => {
    onChange(
      "system.checker.domains",
      config.system.checker.domains.filter((d: string) => d !== domain)
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
          <SettingTextField
            label="Max Concurrent Tests"
            type="number"
            value={config.system.checker.max_concurrent}
            onChange={(e) =>
              onChange("system.checker.max_concurrent", Number(e.target.value))
            }
            helperText="Maximum number of concurrent tests"
          />
        </Grid>
        <Grid size={{ xs: 12, lg: 6 }}>
          <SettingTextField
            label="Test Timeout (seconds)"
            type="number"
            value={config.system.checker.timeout}
            onChange={(e) =>
              onChange("system.checker.timeout", Number(e.target.value))
            }
            helperText="Domain request timeout in seconds"
          />
        </Grid>
        <Grid size={{ sm: 12, md: 6 }}>
          <Typography
            variant="h6"
            sx={{
              display: "flex",
              alignItems: "center",
              gap: 1,
              mb: 2,
            }}
          >
            <DomainIcon /> Domains to Test
          </Typography>
          <Box sx={{ display: "flex", gap: 1, alignItems: "flex-start" }}>
            <SettingTextField
              label="Add Domain"
              value={newDomain}
              onChange={(e) => setNewDomain(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === "Tab" || e.key === ",") {
                  e.preventDefault();
                  handleAddDomain();
                }
              }}
              helperText="e.g., youtube.com, google.com"
              placeholder="example.com"
            />
            <IconButton
              onClick={handleAddDomain}
              sx={{
                bgcolor: colors.accent.secondary,
                color: colors.secondary,
                "&:hover": {
                  bgcolor: colors.accent.secondaryHover,
                },
              }}
            >
              <AddIcon />
            </IconButton>
          </Box>
        </Grid>
        <Grid size={{ sm: 12, md: 6 }}>
          <Box sx={{ mt: 2 }}>
            <Typography variant="subtitle2" gutterBottom>
              Active domains to test
            </Typography>
            <Box
              sx={{
                display: "flex",
                flexWrap: "wrap",
                gap: 1,
                maxHeight: 200,
                overflowY: "auto",
                p: 1,
                border:
                  config.system.checker.domains.length > 0
                    ? `1px solid ${colors.border.default}`
                    : "none",
                borderRadius: 1,
              }}
            >
              {config.system.checker.domains.length === 0 ? (
                <Typography variant="body2" color="text.secondary">
                  No domains added
                </Typography>
              ) : (
                config.system.checker.domains.map((domain: string) => (
                  <Chip
                    key={domain}
                    label={domain}
                    onDelete={() => handleRemoveDomain(domain)}
                    size="small"
                    sx={{
                      bgcolor: colors.accent.primary,
                      color: colors.secondary,
                      "& .MuiChip-deleteIcon": {
                        color: colors.secondary,
                      },
                    }}
                  />
                ))
              )}
            </Box>
          </Box>
        </Grid>
      </Grid>
    </SettingSection>
  );
};
