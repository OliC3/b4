import React from "react";
import { Grid, Box, Chip, IconButton } from "@mui/material";
import { Language as LanguageIcon, Add as AddIcon } from "@mui/icons-material";
import SettingSection from "../../molecules/common/B4Section";
import SettingTextField from "../../atoms/common/B4TextField";
import { colors } from "../../../Theme";
import B4Config from "../../../models/Config";

interface DomainSettingsProps {
  config: B4Config;
  onChange: (field: string, value: string | string[]) => void;
}

export const DomainSettings: React.FC<DomainSettingsProps> = ({
  config,
  onChange,
}) => {
  const [newDomain, setNewDomain] = React.useState("");
  const [newCategory, setNewCategory] = React.useState("");

  const handleAddDomain = () => {
    if (newDomain.trim()) {
      onChange("domains.sni_domains", [
        ...config.domains.sni_domains,
        newDomain.trim(),
      ]);
      setNewDomain("");
    }
  };

  const handleRemoveDomain = (domain: string) => {
    onChange(
      "domains.sni_domains",
      config.domains.sni_domains.filter((d) => d !== domain)
    );
  };

  const handleAddCategory = () => {
    if (newCategory.trim()) {
      onChange("domains.geosite_categories", [
        ...config.domains.geosite_categories,
        newCategory.trim(),
      ]);
      setNewCategory("");
    }
  };

  const handleRemoveCategory = (category: string) => {
    onChange(
      "domains.geosite_categories",
      config.domains.geosite_categories.filter((c) => c !== category)
    );
  };

  return (
    <SettingSection
      title="Domain & GeoData Configuration"
      description="Configure SNI domain filtering and geodata sources"
      icon={<LanguageIcon />}
    >
      <Grid container spacing={2}>
        <Grid size={{ xs: 12 }}>
          <Box sx={{ display: "flex", gap: 1, alignItems: "flex-start" }}>
            <SettingTextField
              label="Add SNI Domain"
              value={newDomain}
              onChange={(e) => setNewDomain(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === "Tab" || e.key === ",") {
                  e.preventDefault();
                  handleAddDomain();
                }
              }}
              helperText="Enter domain and press Enter or click +"
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
          <Box
            sx={{
              mt: 2,
              display: "flex",
              flexWrap: "wrap",
              gap: 1,
            }}
          >
            {config.domains.sni_domains.map((domain) => (
              <Chip
                key={domain}
                label={domain}
                onDelete={() => handleRemoveDomain(domain)}
                sx={{
                  bgcolor: colors.accent.primary,
                  color: colors.secondary,
                  borderColor: colors.primary,
                  borderWidth: 1,
                  borderStyle: "solid",
                  p: 1,
                }}
              />
            ))}
          </Box>
        </Grid>

        <Grid size={{ xs: 12, md: 6 }}>
          <SettingTextField
            label="GeoSite Path"
            value={config.domains.geosite_path}
            onChange={(e) => onChange("domains.geosite_path", e.target.value)}
            helperText="Path to geosite.dat file"
          />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <SettingTextField
            label="GeoIP Path"
            value={config.domains.geoip_path}
            onChange={(e) => onChange("domains.geoip_path", e.target.value)}
            helperText="Path to geoip.dat file"
          />
        </Grid>

        <Grid size={{ xs: 12 }}>
          <Box sx={{ display: "flex", gap: 1, alignItems: "flex-start" }}>
            <SettingTextField
              label="Add Geo Category"
              value={newCategory}
              onChange={(e) => setNewCategory(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === "Tab" || e.key === ",") {
                  e.preventDefault();
                  handleAddCategory();
                }
              }}
              helperText="e.g., youtube, facebook, amazon"
            />
            <IconButton
              onClick={handleAddCategory}
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
          <Box sx={{ mt: 2, display: "flex", flexWrap: "wrap", gap: 1 }}>
            {config.domains.geosite_categories.map((category) => (
              <Chip
                key={category}
                label={category}
                onDelete={() => handleRemoveCategory(category)}
                sx={{
                  bgcolor: colors.accent.primary,
                  color: colors.secondary,
                  borderColor: colors.primary,
                  borderWidth: 1,
                  borderStyle: "solid",
                }}
              />
            ))}
          </Box>
        </Grid>
      </Grid>
    </SettingSection>
  );
};
