import React, { useState, useEffect } from "react";
import {
  Grid,
  Box,
  Chip,
  IconButton,
  Typography,
  Alert,
  Collapse,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  List,
  ListItem,
  ListItemText,
  Skeleton,
  Paper,
  Tooltip,
} from "@mui/material";
import {
  Language as LanguageIcon,
  Add as AddIcon,
  ExpandMore as ExpandMoreIcon,
  ExpandLess as ExpandLessIcon,
  Info as InfoIcon,
  Category as CategoryIcon,
  Domain as DomainIcon,
} from "@mui/icons-material";
import SettingSection from "../../molecules/common/B4Section";
import SettingTextField from "../../atoms/common/B4TextField";
import SettingAutocomplete from "../../atoms/common/B4Autocomplete";
import { colors } from "../../../Theme";
import B4Config from "../../../models/Config";

interface DomainSettingsProps {
  config: B4Config & { domain_stats?: DomainStatistics };
  onChange: (field: string, value: string | string[]) => void;
}

interface DomainStatistics {
  manual_domains: number;
  geosite_domains: number;
  total_domains: number;
  category_breakdown?: Record<string, number>;
  geosite_available: boolean;
}

interface CategoryPreview {
  category: string;
  total_domains: number;
  preview_count: number;
  preview: string[];
}

export const DomainSettings: React.FC<DomainSettingsProps> = ({
  config,
  onChange,
}) => {
  const [newDomain, setNewDomain] = useState("");
  const [newCategory, setNewCategory] = useState("");
  const [availableCategories, setAvailableCategories] = useState<string[]>([]);
  const [loadingCategories, setLoadingCategories] = useState(false);
  const [previewDialog, setPreviewDialog] = useState<{
    open: boolean;
    category: string;
    data?: CategoryPreview;
    loading: boolean;
  }>({ open: false, category: "", loading: false });
  const [showAdvanced, setShowAdvanced] = useState(false);

  // Fetch available geosite categories when geosite path is set
  useEffect(() => {
    if (config.domains.geosite_path) {
      loadAvailableCategories();
    }
  }, [config.domains.geosite_path]);

  const loadAvailableCategories = async () => {
    setLoadingCategories(true);
    try {
      const response = await fetch("/api/geosite");
      if (response.ok) {
        const data = await response.json();
        setAvailableCategories(data.tags || []);
      }
    } catch (error) {
      console.error("Failed to load geosite categories:", error);
    } finally {
      setLoadingCategories(false);
    }
  };

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

  const handleAddCategory = (category: string) => {
    if (category && !config.domains.geosite_categories.includes(category)) {
      onChange("domains.geosite_categories", [
        ...config.domains.geosite_categories,
        category,
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

  const previewCategory = async (category: string) => {
    setPreviewDialog({ open: true, category, loading: true });
    try {
      const response = await fetch(
        `/api/geosite/category?tag=${encodeURIComponent(category)}`
      );
      if (response.ok) {
        const data = await response.json();
        setPreviewDialog((prev) => ({ ...prev, data, loading: false }));
      }
    } catch (error) {
      console.error("Failed to preview category:", error);
      setPreviewDialog((prev) => ({ ...prev, loading: false }));
    }
  };

  const stats = config.domain_stats;

  return (
    <>
      <SettingSection
        title="Domain Filtering Configuration"
        description="Configure domain matching for DPI bypass"
        icon={<LanguageIcon />}
      >
        {/* Statistics Dashboard */}
        {stats && (
          <Paper
            elevation={0}
            sx={{
              p: 2,
              mb: 3,
              bgcolor: colors.background.paper,
              border: `1px solid ${colors.border.default}`,
            }}
          >
            <Typography variant="subtitle2" color="text.secondary" gutterBottom>
              Domain Statistics
            </Typography>
            <Grid container spacing={2}>
              <Grid size={{ xs: 12, sm: 4 }}>
                <Box sx={{ textAlign: "center" }}>
                  <Typography variant="h4" color="primary">
                    {stats.manual_domains || 0}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    Manual Domains
                  </Typography>
                </Box>
              </Grid>
              <Grid size={{ xs: 12, sm: 4 }}>
                <Box sx={{ textAlign: "center" }}>
                  <Typography variant="h4" color="secondary">
                    {stats.geosite_domains || 0}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    GeoSite Domains
                  </Typography>
                </Box>
              </Grid>
              <Grid size={{ xs: 12, sm: 4 }}>
                <Box sx={{ textAlign: "center" }}>
                  <Typography variant="h4" color="accent.primary">
                    {stats.total_domains || 0}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    Total Active
                  </Typography>
                </Box>
              </Grid>
            </Grid>
          </Paper>
        )}

        <Grid container spacing={3}>
          {/* Manual Domains Section */}
          <Grid size={{ xs: 12 }}>
            <Box sx={{ mb: 2 }}>
              <Typography
                variant="h6"
                sx={{
                  display: "flex",
                  alignItems: "center",
                  gap: 1,
                  mb: 2,
                }}
              >
                <DomainIcon /> Manual Domains
                <Tooltip title="Add specific domains to match. These take priority over GeoSite categories.">
                  <InfoIcon fontSize="small" color="action" />
                </Tooltip>
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
                  helperText="e.g., youtube.com, *.google.com"
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
              <Box
                sx={{
                  mt: 2,
                  display: "flex",
                  flexWrap: "wrap",
                  gap: 1,
                  maxHeight: 200,
                  overflowY: "auto",
                  p: 1,
                  border:
                    config.domains.sni_domains.length > 0
                      ? `1px solid ${colors.border.default}`
                      : "none",
                  borderRadius: 1,
                }}
              >
                {config.domains.sni_domains.length === 0 ? (
                  <Typography variant="body2" color="text.secondary">
                    No manual domains added
                  </Typography>
                ) : (
                  config.domains.sni_domains.map((domain) => (
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

          {/* GeoSite Configuration */}
          <Grid size={{ xs: 12 }}>
            <Box sx={{ mb: 2 }}>
              <Typography
                variant="h6"
                sx={{
                  display: "flex",
                  alignItems: "center",
                  gap: 1,
                  mb: 2,
                }}
              >
                <CategoryIcon /> GeoSite Categories
                <Tooltip title="Load predefined domain lists from GeoSite database">
                  <InfoIcon fontSize="small" color="action" />
                </Tooltip>
              </Typography>

              {/* GeoSite Path */}
              <Grid container spacing={2}>
                <Grid size={{ xs: 12, md: 6 }}>
                  <SettingTextField
                    label="GeoSite Database Path"
                    value={config.domains.geosite_path}
                    onChange={(e) =>
                      onChange("domains.geosite_path", e.target.value)
                    }
                    helperText="Path to geosite.dat file"
                    placeholder="/path/to/geosite.dat"
                  />
                </Grid>
                <Grid size={{ xs: 12, md: 6 }}>
                  {config.domains.geosite_path && (
                    <SettingAutocomplete
                      label="Add Category"
                      value={newCategory}
                      options={availableCategories}
                      onChange={(value) => setNewCategory(value)}
                      onSelect={handleAddCategory}
                      loading={loadingCategories}
                      placeholder="Select or type category"
                      helperText={`${availableCategories.length} categories available`}
                    />
                  )}
                </Grid>
              </Grid>

              {/* Active Categories */}
              {config.domains.geosite_categories.length > 0 && (
                <Box sx={{ mt: 3 }}>
                  <Typography variant="subtitle2" gutterBottom>
                    Active Categories
                  </Typography>
                  <Box
                    sx={{
                      display: "flex",
                      flexWrap: "wrap",
                      gap: 1,
                      p: 2,
                      border: `1px solid ${colors.border.default}`,
                      borderRadius: 1,
                      bgcolor: colors.background.paper,
                    }}
                  >
                    {config.domains.geosite_categories.map((category) => (
                      <Chip
                        key={category}
                        label={
                          <Box
                            sx={{
                              display: "flex",
                              alignItems: "center",
                              gap: 0.5,
                            }}
                          >
                            <span>{category}</span>
                            {stats?.category_breakdown?.[category] && (
                              <Typography
                                component="span"
                                variant="caption"
                                sx={{
                                  cursor: "pointer",
                                  bgcolor: "action.selected",
                                  px: 0.5,
                                  borderRadius: 0.5,
                                }}
                                onClick={(e) => {
                                  e.stopPropagation();
                                  previewCategory(category);
                                }}
                              >
                                {stats.category_breakdown[category]}
                              </Typography>
                            )}
                          </Box>
                        }
                        onDelete={() => handleRemoveCategory(category)}
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
                </Box>
              )}

              {!config.domains.geosite_path && (
                <Alert severity="info" sx={{ mt: 2 }}>
                  Configure GeoSite path to enable category-based domain
                  filtering
                </Alert>
              )}
            </Box>
          </Grid>

          {/* Advanced Options */}
          <Grid size={{ xs: 12 }}>
            <Button
              onClick={() => setShowAdvanced(!showAdvanced)}
              startIcon={showAdvanced ? <ExpandLessIcon /> : <ExpandMoreIcon />}
              sx={{ mb: 2 }}
            >
              Advanced Options
            </Button>
            <Collapse in={showAdvanced}>
              <Grid container spacing={2}>
                <Grid size={{ xs: 12, md: 6 }}>
                  <SettingTextField
                    label="GeoIP Path"
                    value={config.domains.geoip_path}
                    onChange={(e) =>
                      onChange("domains.geoip_path", e.target.value)
                    }
                    helperText="Path to geoip.dat file (optional)"
                  />
                </Grid>
                {/* Add more advanced options here if needed */}
              </Grid>
            </Collapse>
          </Grid>
        </Grid>
      </SettingSection>

      {/* Preview Dialog */}
      <Dialog
        open={previewDialog.open}
        onClose={() =>
          setPreviewDialog({ open: false, category: "", loading: false })
        }
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle>Category Preview: {previewDialog.category}</DialogTitle>
        <DialogContent>
          {previewDialog.loading ? (
            <Box sx={{ p: 2 }}>
              <Skeleton variant="text" />
              <Skeleton variant="text" />
              <Skeleton variant="text" />
            </Box>
          ) : previewDialog.data ? (
            <>
              <Alert severity="info" sx={{ mb: 2 }}>
                Total domains in category: {previewDialog.data.total_domains}
                {previewDialog.data.total_domains >
                  previewDialog.data.preview_count &&
                  ` (showing first ${previewDialog.data.preview_count})`}
              </Alert>
              <List dense sx={{ maxHeight: 300, overflow: "auto" }}>
                {previewDialog.data.preview.map((domain) => (
                  <ListItem key={domain}>
                    <ListItemText primary={domain} />
                  </ListItem>
                ))}
              </List>
            </>
          ) : (
            <Alert severity="error">Failed to load category preview</Alert>
          )}
        </DialogContent>
        <DialogActions>
          <Button
            onClick={() =>
              setPreviewDialog({ open: false, category: "", loading: false })
            }
          >
            Close
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
};
