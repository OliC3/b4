import React, { useState, useEffect } from "react";
import {
  Grid,
  Box,
  Chip,
  IconButton,
  Typography,
  Alert,
  Button,
  List,
  ListItem,
  ListItemText,
  Skeleton,
  Paper,
  Tooltip,
  Tabs,
  Tab,
  Stack,
} from "@mui/material";
import {
  Language as LanguageIcon,
  Add as AddIcon,
  Info as InfoIcon,
  Category as CategoryIcon,
  Domain as DomainIcon,
  Block as BlockIcon,
  Security as SecurityIcon,
} from "@mui/icons-material";
import SettingSection from "@molecules/common/B4Section";
import SettingTextField from "@atoms/common/B4TextField";
import SettingAutocomplete from "@atoms/common/B4Autocomplete";
import { colors, button_primary } from "@design";
import { B4Dialog } from "@molecules/common/B4Dialog";
import { B4SetConfig, GeoConfig } from "@models/Config";

interface DomainSettingsProps {
  config: B4SetConfig & { domain_stats?: DomainStatistics };
  geo: GeoConfig;
  onChange: (field: string, value: string | string[]) => void;
}

interface DomainStatistics {
  // Bypass stats
  manual_domains: number;
  geosite_domains: number;
  total_domains: number;
  category_breakdown?: Record<string, number>;
  geosite_available: boolean;
  // Block stats
  block_manual_domains?: number;
  block_geosite_domains?: number;
  block_total_domains?: number;
  block_category_breakdown?: Record<string, number>;
}

interface CategoryPreview {
  category: string;
  total_domains: number;
  preview_count: number;
  preview: string[];
}

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: Readonly<TabPanelProps>) {
  const { children, value, index, ...other } = props;
  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`domain-tabpanel-${index}`}
      aria-labelledby={`domain-tab-${index}`}
      {...other}
    >
      {value === index && <Box sx={{ pt: 3 }}>{children}</Box>}
    </div>
  );
}

export const DomainSettings: React.FC<DomainSettingsProps> = ({
  config,
  onChange,
  geo,
}) => {
  const [tabValue, setTabValue] = useState(0);
  const [newBypassDomain, setNewBypassDomain] = useState("");
  const [newBlockDomain, setNewBlockDomain] = useState("");
  const [newBypassCategory, setNewBypassCategory] = useState("");
  const [newBlockCategory, setNewBlockCategory] = useState("");
  const [availableCategories, setAvailableCategories] = useState<string[]>([]);
  const [loadingCategories, setLoadingCategories] = useState(false);

  const [previewDialog, setPreviewDialog] = useState<{
    open: boolean;
    category: string;
    data?: CategoryPreview;
    loading: boolean;
  }>({ open: false, category: "", loading: false });

  useEffect(() => {
    if (geo.sitedat_path) {
      void loadAvailableCategories();
    }
  }, [geo.sitedat_path]);

  const loadAvailableCategories = async () => {
    setLoadingCategories(true);
    try {
      const response = await fetch("/api/geosite");
      if (response.ok) {
        const data = (await response.json()) as { tags: string[] };
        setAvailableCategories(data.tags || []);
      }
    } catch (error) {
      console.error("Failed to load geosite categories:", error);
    } finally {
      setLoadingCategories(false);
    }
  };

  // Bypass domain handlers
  const handleAddBypassDomain = () => {
    if (newBypassDomain.trim()) {
      onChange("domains.sni_domains", [
        ...config.domains.sni_domains,
        newBypassDomain.trim(),
        config.id,
      ]);
      setNewBypassDomain("");
    }
  };

  const handleRemoveBypassDomain = (domain: string) => {
    onChange(
      "domains.sni_domains",
      config.domains.sni_domains.filter((d) => d !== domain)
    );
  };

  const handleAddBypassCategory = (category: string) => {
    if (category && !config.domains.geosite_categories.includes(category)) {
      onChange("domains.geosite_categories", [
        ...config.domains.geosite_categories,
        category,
      ]);
      setNewBypassCategory("");
    }
  };

  const handleRemoveBypassCategory = (category: string) => {
    onChange(
      "domains.geosite_categories",
      config.domains.geosite_categories.filter((c) => c !== category)
    );
  };

  // Block domain handlers
  const handleAddBlockDomain = () => {
    if (newBlockDomain.trim()) {
      const blockDomains = config.domains.block_domains || [];
      onChange("domains.block_domains", [
        ...blockDomains,
        newBlockDomain.trim(),
      ]);
      setNewBlockDomain("");
    }
  };

  const handleRemoveBlockDomain = (domain: string) => {
    onChange(
      "domains.block_domains",
      (config.domains.block_domains || []).filter((d) => d !== domain)
    );
  };

  const handleAddBlockCategory = (category: string) => {
    const blockCategories = config.domains.block_geosite_categories || [];
    if (category && !blockCategories.includes(category)) {
      onChange("domains.block_geosite_categories", [
        ...blockCategories,
        category,
      ]);
      setNewBlockCategory("");
    }
  };

  const handleRemoveBlockCategory = (category: string) => {
    onChange(
      "domains.block_geosite_categories",
      (config.domains.block_geosite_categories || []).filter(
        (c) => c !== category
      )
    );
  };

  const previewCategory = async (category: string) => {
    setPreviewDialog({ open: true, category, loading: true });
    try {
      const response = await fetch(
        `/api/geosite/category?tag=${encodeURIComponent(category)}`
      );
      if (response.ok) {
        const data = (await response.json()) as CategoryPreview;
        setPreviewDialog((prev) => ({ ...prev, data, loading: false }));
      }
    } catch (error) {
      console.error("Failed to preview category:", error);
      setPreviewDialog((prev) => ({ ...prev, loading: false }));
    }
  };

  const stats = config.domain_stats;
  const blockDomains = config.domains.block_domains || [];
  const blockCategories = config.domains.block_geosite_categories || [];

  // Calculate totals for tab badges
  const bypassTotal = stats?.total_domains || 0;
  const blockTotal =
    stats?.block_total_domains ||
    0 ||
    blockDomains.length + (stats?.block_geosite_domains || 0);

  return (
    <>
      <Stack spacing={3}>
        <SettingSection
          title="Domain Filtering Configuration"
          description="Configure domain matching for DPI bypass and blocking"
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
              <Typography
                variant="subtitle2"
                color="text.secondary"
                gutterBottom
              >
                Overall Domain Statistics
              </Typography>
              <Grid container spacing={2}>
                <Grid size={{ xs: 12, sm: 4 }}>
                  <Box sx={{ textAlign: "center" }}>
                    <Typography variant="h4" color="primary">
                      {bypassTotal}
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      DPI Bypass
                    </Typography>
                  </Box>
                </Grid>
                <Grid size={{ xs: 12, sm: 4 }}>
                  <Box sx={{ textAlign: "center" }}>
                    <Typography variant="h4" color="error">
                      {blockTotal}
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      Blocked
                    </Typography>
                  </Box>
                </Grid>
                <Grid size={{ xs: 12, sm: 4 }}>
                  <Box sx={{ textAlign: "center" }}>
                    <Typography variant="h4" color="secondary">
                      {bypassTotal + blockTotal}
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      Total Validated
                    </Typography>
                  </Box>
                </Grid>
              </Grid>
            </Paper>
          )}

          <Box sx={{ borderBottom: 1, borderColor: "divider", mb: 0 }}>
            <Tabs
              value={tabValue}
              onChange={(_, newValue: number) => setTabValue(newValue)}
              sx={{
                borderBottom: `1px solid ${colors.border.light}`,
                "& .MuiTab-root": {
                  color: colors.text.secondary,
                  textTransform: "none",
                  minHeight: 48,
                  "&.Mui-selected": {
                    color: colors.secondary,
                  },
                },
                "& .MuiTabs-indicator": {
                  bgcolor: colors.secondary,
                },
              }}
            >
              <Tab
                icon={<SecurityIcon />}
                iconPosition="start"
                label={
                  <Box sx={{ display: "flex", alignItems: "center", gap: 2.5 }}>
                    <span>Bypass Domains</span>
                  </Box>
                }
              />
              <Tab
                icon={<BlockIcon />}
                iconPosition="start"
                label={
                  <Box sx={{ display: "flex", alignItems: "center", gap: 2.5 }}>
                    <span>Block Domains</span>
                  </Box>
                }
              />
            </Tabs>
          </Box>
          {/* DPI Bypass Tab */}
          <TabPanel value={tabValue} index={0}>
            <Alert severity="info" sx={{ mb: 2 }}>
              Domains in this list will use DPI bypass techniques
              (fragmentation, faking) when matched.
            </Alert>

            <Grid container spacing={2}>
              {/* Manual Bypass Domains */}
              <Grid size={{ sm: 12, md: 6 }}>
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
                    <DomainIcon /> Manual Bypass Domains
                    <Tooltip title="Add specific domains to bypass DPI. These take priority over GeoSite categories.">
                      <InfoIcon fontSize="small" color="action" />
                    </Tooltip>
                  </Typography>
                  <Box
                    sx={{ display: "flex", gap: 1, alignItems: "flex-start" }}
                  >
                    <SettingTextField
                      label="Add Bypass Domain"
                      value={newBypassDomain}
                      onChange={(e) => setNewBypassDomain(e.target.value)}
                      onKeyDown={(e) => {
                        if (
                          e.key === "Enter" ||
                          e.key === "Tab" ||
                          e.key === ","
                        ) {
                          e.preventDefault();
                          handleAddBypassDomain();
                        }
                      }}
                      helperText="e.g., youtube.com, google.com"
                      placeholder="example.com"
                    />
                    <IconButton
                      onClick={handleAddBypassDomain}
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
                  <Box sx={{ mt: 2 }}>
                    <Typography variant="subtitle2" gutterBottom>
                      Active manually added domains
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
                      {config.domains.sni_domains.length === 0 ? (
                        <Typography variant="body2" color="text.secondary">
                          No bypass domains added
                        </Typography>
                      ) : (
                        config.domains.sni_domains.map((domain) => (
                          <Chip
                            key={domain}
                            label={domain}
                            onDelete={() => handleRemoveBypassDomain(domain)}
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
                </Box>
              </Grid>

              {geo.sitedat_path && availableCategories.length > 0 && (
                <Grid size={{ sm: 12, md: 6 }}>
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
                      <CategoryIcon /> Bypass GeoSite Categories
                      <Tooltip title="Load predefined domain lists from GeoSite database for DPI bypass">
                        <InfoIcon fontSize="small" color="action" />
                      </Tooltip>
                    </Typography>

                    <SettingAutocomplete
                      label="Add Bypass Category"
                      value={newBypassCategory}
                      options={availableCategories}
                      onChange={setNewBypassCategory}
                      onSelect={handleAddBypassCategory}
                      loading={loadingCategories}
                      placeholder="Select or type category"
                      helperText={`${availableCategories.length} categories available`}
                    />

                    {config.domains.geosite_categories.length > 0 && (
                      <Box sx={{ mt: 2 }}>
                        <Typography variant="subtitle2" gutterBottom>
                          Active Bypass Categories
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
                          {config.domains.geosite_categories.map((category) => (
                            <Chip
                              size="small"
                              key={category}
                              label={
                                <Box
                                  sx={{
                                    display: "flex",
                                    alignItems: "center",
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
                                        ml: 0.5,
                                        borderRadius: 1,
                                      }}
                                      onClick={(e) => {
                                        e.stopPropagation();
                                        void previewCategory(category);
                                      }}
                                    >
                                      {stats.category_breakdown[category]}
                                    </Typography>
                                  )}
                                </Box>
                              }
                              onDelete={() =>
                                handleRemoveBypassCategory(category)
                              }
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
                  </Box>
                </Grid>
              )}
            </Grid>
          </TabPanel>
          {/* Block List Tab */}
          <TabPanel value={tabValue} index={1}>
            <Alert severity="warning" sx={{ mb: 2 }}>
              <strong>Warning:</strong> This feature is under development and
              does not work yet.
            </Alert>
            <Alert severity="info" sx={{ mb: 2 }}>
              Domains in this list will be completely blocked - all packets will
              be dropped.
            </Alert>

            <Grid container spacing={3}>
              {/* Manual Block Domains */}
              <Grid size={{ sm: 12, md: 6 }}>
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
                    <BlockIcon /> Manual Block Domains
                    <Tooltip title="Add specific domains to block completely. No packets will pass through.">
                      <InfoIcon fontSize="small" color="action" />
                    </Tooltip>
                  </Typography>
                  <Box
                    sx={{ display: "flex", gap: 1, alignItems: "flex-start" }}
                  >
                    <SettingTextField
                      label="Add Block Domain"
                      value={newBlockDomain}
                      onChange={(e) => setNewBlockDomain(e.target.value)}
                      onKeyDown={(e) => {
                        if (
                          e.key === "Enter" ||
                          e.key === "Tab" ||
                          e.key === ","
                        ) {
                          e.preventDefault();
                          handleAddBlockDomain();
                        }
                      }}
                      helperText="e.g., ads.example.com, tracker.com"
                      placeholder="blocked-site.com"
                    />
                    <IconButton
                      onClick={handleAddBlockDomain}
                      sx={{
                        bgcolor: "error.main",
                        color: "white",
                        "&:hover": {
                          bgcolor: "error.dark",
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
                        blockDomains.length > 0
                          ? `1px solid ${colors.border.default}`
                          : "none",
                      borderRadius: 1,
                    }}
                  >
                    {blockDomains.length === 0 ? (
                      <Typography variant="body2" color="text.secondary">
                        No block domains added
                      </Typography>
                    ) : (
                      blockDomains.map((domain) => (
                        <Chip
                          key={domain}
                          label={domain}
                          onDelete={() => handleRemoveBlockDomain(domain)}
                          size="small"
                          color="error"
                        />
                      ))
                    )}
                  </Box>
                </Box>
              </Grid>

              {geo.sitedat_path && availableCategories.length > 0 && (
                <Grid size={{ sm: 12, md: 6 }}>
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
                      <CategoryIcon /> Block GeoSite Categories
                      <Tooltip title="Load predefined domain lists from GeoSite database to block">
                        <InfoIcon fontSize="small" color="action" />
                      </Tooltip>
                    </Typography>

                    <SettingAutocomplete
                      label="Add Block Category"
                      value={newBlockCategory}
                      options={availableCategories}
                      onChange={setNewBlockCategory}
                      onSelect={handleAddBlockCategory}
                      loading={loadingCategories}
                      placeholder="Select category to block"
                      helperText={`${availableCategories.length} categories available`}
                    />

                    {blockCategories.length > 0 && (
                      <Box sx={{ mt: 3 }}>
                        <Typography variant="subtitle2" gutterBottom>
                          Active Block Categories
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
                          {blockCategories.map((category) => (
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
                                  {stats?.block_category_breakdown?.[
                                    category
                                  ] && (
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
                                        void previewCategory(category);
                                      }}
                                    >
                                      {stats.block_category_breakdown[category]}
                                    </Typography>
                                  )}
                                </Box>
                              }
                              onDelete={() =>
                                handleRemoveBlockCategory(category)
                              }
                              color="error"
                            />
                          ))}
                        </Box>
                      </Box>
                    )}
                  </Box>
                </Grid>
              )}
            </Grid>
          </TabPanel>
        </SettingSection>
      </Stack>

      {/* Preview Dialog */}
      <B4Dialog
        title={`${previewDialog.category.toUpperCase()}`}
        subtitle="Category Preview"
        icon={<CategoryIcon />}
        open={previewDialog.open}
        onClose={() =>
          setPreviewDialog({ open: false, category: "", loading: false })
        }
        actions={
          <Button
            variant="contained"
            onClick={() =>
              setPreviewDialog({ open: false, category: "", loading: false })
            }
            sx={{
              ...button_primary,
            }}
          >
            Close
          </Button>
        }
      >
        <>
          {(() => {
            if (previewDialog.loading) {
              return (
                <Box sx={{ p: 2 }}>
                  <Skeleton variant="text" />
                  <Skeleton variant="text" />
                  <Skeleton variant="text" />
                </Box>
              );
            } else if (previewDialog.data) {
              return (
                <>
                  <Alert severity="info" sx={{ mb: 2 }}>
                    Total domains in category:{" "}
                    {previewDialog.data.total_domains}
                    {previewDialog.data.total_domains >
                      previewDialog.data.preview_count &&
                      ` (showing first ${previewDialog.data.preview_count})`}
                  </Alert>
                  <List dense sx={{ maxHeight: 600, overflow: "auto" }}>
                    {previewDialog.data.preview.map((domain) => (
                      <ListItem key={domain}>
                        <ListItemText primary={domain} />
                      </ListItem>
                    ))}
                  </List>
                </>
              );
            } else {
              return (
                <Alert severity="error">Failed to load category preview</Alert>
              );
            }
          })()}
        </>
      </B4Dialog>
    </>
  );
};
