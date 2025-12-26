import { useState, useEffect } from "react";
import {
  Grid,
  Box,
  Typography,
  Button,
  List,
  ListItem,
  ListItemText,
  Skeleton,
  Tooltip,
  Stack,
} from "@mui/material";
import {
  DomainIcon,
  CategoryIcon,
  InfoIcon,
  ClearIcon,
  IpIcon,
} from "@b4.icons";

import {
  B4TextField,
  B4Section,
  B4Dialog,
  B4Alert,
  B4Tabs,
  B4Tab,
  B4ChipList,
  B4PlusButton,
} from "@b4.elements";
import SettingAutocomplete from "@common/B4Autocomplete";
import { B4SetConfig, GeoConfig } from "@models/config";
import { SetStats } from "./Manager";

interface TargetSettingsProps {
  config: B4SetConfig;
  geo: GeoConfig;
  stats?: SetStats;
  onChange: (field: string, value: string | string[]) => void;
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

export const TargetSettings = ({
  config,
  onChange,
  geo,
  stats,
}: TargetSettingsProps) => {
  const [tabValue, setTabValue] = useState(0);
  const [newBypassDomain, setNewBypassDomain] = useState("");
  const [newBypassIP, setNewBypassIP] = useState("");
  const [newBypassCategory, setNewBypassCategory] = useState("");
  const [availableCategories, setAvailableCategories] = useState<string[]>([]);
  const [loadingCategories, setLoadingCategories] = useState(false);

  const [newBypassGeoIPCategory, setNewBypassGeoIPCategory] = useState("");
  const [availableGeoIPCategories, setAvailableGeoIPCategories] = useState<
    string[]
  >([]);
  const [loadingGeoIPCategories, setLoadingGeoIPCategories] = useState(false);

  const [previewDialog, setPreviewDialog] = useState<{
    open: boolean;
    category: string;
    data?: CategoryPreview;
    loading: boolean;
  }>({ open: false, category: "", loading: false });

  useEffect(() => {
    if (geo.sitedat_path) {
      void loadAvailableSiteCategories();
    }
    if (geo.ipdat_path) {
      void loadAvailableGeoIPCategories();
    }
  }, [geo.sitedat_path, geo.ipdat_path]);

  const loadAvailableSiteCategories = async () => {
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

  const loadAvailableGeoIPCategories = async () => {
    setLoadingGeoIPCategories(true);
    try {
      const response = await fetch("/api/geoip");
      if (response.ok) {
        const data = (await response.json()) as { tags: string[] };
        setAvailableGeoIPCategories(data.tags || []);
      }
    } catch (error) {
      console.error("Failed to load geoip categories:", error);
    } finally {
      setLoadingGeoIPCategories(false);
    }
  };

  const handleAddBypassDomain = () => {
    const value = newBypassDomain.trim();
    if (!value) return;

    const domainRange = value.split(/[\s,|]+/).filter(Boolean);
    const existing = new Set(config.targets.sni_domains);
    const next = [...config.targets.sni_domains];

    for (const raw of domainRange) {
      const domain = raw.trim();
      if (domain && !existing.has(domain)) {
        existing.add(domain);
        next.push(domain);
      }
    }

    onChange("targets.sni_domains", next);
    setNewBypassDomain("");
  };

  const handleAddBypassIP = () => {
    const value = newBypassIP.trim();
    if (!value) return;

    const ipRange = value.split(/[\s,|]+/).filter(Boolean);
    const existing = new Set(config.targets.ip);
    const next = [...config.targets.ip];

    for (const raw of ipRange) {
      const ip = raw.trim();
      if (ip && !existing.has(ip)) {
        existing.add(ip);
        next.push(ip);
      }
    }

    onChange("targets.ip", next);
    setNewBypassIP("");
  };

  const handleClearAllBypassIPs = () => {
    onChange("targets.ip", []);
  };

  const handleRemoveBypassIP = (ip: string) => {
    onChange(
      "targets.ip",
      config.targets.ip.filter((d) => d !== ip)
    );
  };

  const handleAddBypassGeoIPCategory = (category: string) => {
    if (category && !config.targets.geoip_categories.includes(category)) {
      onChange("targets.geoip_categories", [
        ...config.targets.geoip_categories,
        category,
      ]);
      setNewBypassGeoIPCategory("");
    }
  };

  const handleRemoveBypassGeoIPCategory = (category: string) => {
    onChange(
      "targets.geoip_categories",
      config.targets.geoip_categories.filter((c) => c !== category)
    );
  };

  const handleRemoveBypassDomain = (domain: string) => {
    onChange(
      "targets.sni_domains",
      config.targets.sni_domains.filter((d) => d !== domain)
    );
  };

  const handleAddBypassCategory = (category: string) => {
    if (category && !config.targets.geosite_categories.includes(category)) {
      onChange("targets.geosite_categories", [
        ...config.targets.geosite_categories,
        category,
      ]);
      setNewBypassCategory("");
    }
  };

  const handleRemoveBypassCategory = (category: string) => {
    onChange(
      "targets.geosite_categories",
      config.targets.geosite_categories.filter((c) => c !== category)
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

  const renderCategoryLabel = (
    category: string,
    breakdown?: Record<string, number>
  ) => {
    const count = breakdown?.[category];
    return (
      <Box sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
        <span>{category}</span>
        {count && (
          <Typography
            component="span"
            variant="caption"
            sx={{
              bgcolor: "action.selected",
              px: 0.5,
              borderRadius: 1,
              fontWeight: 600,
            }}
          >
            {count}
          </Typography>
        )}
      </Box>
    );
  };

  return (
    <>
      <Stack spacing={3}>
        <B4Section
          title="Domain Filtering Configuration"
          description="Configure domain matching for DPI bypass and blocking"
          icon={<DomainIcon />}
        >
          <Box sx={{ borderBottom: 1, borderColor: "divider", mb: 0 }}>
            <B4Tabs
              value={tabValue}
              onChange={(_, newValue: number) => setTabValue(newValue)}
            >
              <B4Tab icon={<DomainIcon />} label="Bypass Domains" inline />
              <B4Tab icon={<IpIcon />} label="Bypass IPs" inline />
            </B4Tabs>
          </Box>

          {/* DPI Bypass Tab */}
          <TabPanel value={tabValue} index={0}>
            <B4Alert severity="info" sx={{ m: 0 }}>
              Domains in this list will use DPI bypass techniques
              (fragmentation, faking) when matched.
            </B4Alert>

            <Grid container spacing={2}>
              {/* Manual Bypass Domains */}
              <Grid size={{ sm: 12, md: 6 }}>
                <Box>
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
                    <Tooltip title="Add specific domains to bypass DPI.">
                      <InfoIcon fontSize="small" color="action" />
                    </Tooltip>
                  </Typography>
                  <Box
                    sx={{ display: "flex", gap: 1, alignItems: "flex-start" }}
                  >
                    <B4TextField
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
                    <B4PlusButton
                      onClick={handleAddBypassDomain}
                      disabled={!newBypassDomain.trim()}
                    />
                  </Box>
                  <Box sx={{ mt: 2 }}>
                    <B4ChipList
                      items={config.targets.sni_domains}
                      getKey={(d) => d}
                      getLabel={(d) => d}
                      onDelete={handleRemoveBypassDomain}
                      title="Active manually added domains"
                      emptyMessage="No bypass domains added"
                      showEmpty
                    />
                  </Box>
                </Box>
              </Grid>

              {/* GeoSite Categories */}
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

                    <Box sx={{ mt: 2 }}>
                      <B4ChipList
                        title="Active Bypass Categories"
                        items={config.targets.geosite_categories}
                        getKey={(c) => c}
                        getLabel={(c) =>
                          renderCategoryLabel(
                            c,
                            stats?.geosite_category_breakdown
                          )
                        }
                        onDelete={handleRemoveBypassCategory}
                        onClick={(c) => void previewCategory(c)}
                      />
                    </Box>
                  </Box>
                </Grid>
              )}
            </Grid>
          </TabPanel>

          {/* Bypass IPs Tab */}
          <TabPanel value={tabValue} index={1}>
            <B4Alert>
              IP ranges in these categories will use DPI bypass techniques
              (fragmentation, faking) when matched.
            </B4Alert>

            <Grid container spacing={2}>
              {/* Manual Bypass IPs */}
              <Grid size={{ sm: 12, md: 6 }}>
                <Box>
                  <Typography
                    variant="h6"
                    sx={{
                      display: "flex",
                      alignItems: "center",
                      gap: 1,
                      mb: 2,
                    }}
                  >
                    <DomainIcon /> Manual Bypass IPs
                    <Tooltip title="Add specific ip/cidr to bypass DPI.">
                      <InfoIcon fontSize="small" color="action" />
                    </Tooltip>
                  </Typography>
                  <Box
                    sx={{ display: "flex", gap: 1, alignItems: "flex-start" }}
                  >
                    <B4TextField
                      label="Add Bypass IP/CIDR"
                      value={newBypassIP}
                      onChange={(e) => setNewBypassIP(e.target.value)}
                      onKeyDown={(e) => {
                        if (
                          e.key === "Enter" ||
                          e.key === "Tab" ||
                          e.key === ","
                        ) {
                          e.preventDefault();
                          handleAddBypassIP();
                        }
                      }}
                      helperText="e.g. 192.168.1.1, 10.0.0.0/8"
                      placeholder="192.168.1.1"
                    />
                    <B4PlusButton
                      onClick={handleAddBypassIP}
                      disabled={!newBypassIP}
                    />
                  </Box>
                  <Box sx={{ mt: 2 }}>
                    <Box
                      sx={{
                        display: "flex",
                        justifyContent: "space-between",
                        alignItems: "center",
                        mb: 1,
                      }}
                    >
                      <Typography variant="subtitle2">
                        Active manually added IPs ({config.targets.ip.length})
                      </Typography>
                      {config.targets.ip.length > 0 && (
                        <Button
                          size="small"
                          onClick={handleClearAllBypassIPs}
                          startIcon={<ClearIcon />}
                        >
                          Clear All
                        </Button>
                      )}
                    </Box>
                    <B4ChipList
                      items={config.targets.ip}
                      getKey={(ip) => ip}
                      getLabel={(ip) => ip}
                      onDelete={handleRemoveBypassIP}
                      emptyMessage="No bypass ip added"
                      showEmpty
                      maxHeight={200}
                    />
                  </Box>
                </Box>
              </Grid>

              {/* GeoIP Categories */}
              {geo.ipdat_path && availableGeoIPCategories.length > 0 && (
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
                      <CategoryIcon /> Bypass GeoIP Categories
                      <Tooltip title="Load predefined IP lists from GeoIP database for DPI bypass">
                        <InfoIcon fontSize="small" color="action" />
                      </Tooltip>
                    </Typography>

                    <SettingAutocomplete
                      label="Add Bypass GeoIP Category"
                      value={newBypassGeoIPCategory}
                      options={availableGeoIPCategories}
                      onChange={setNewBypassGeoIPCategory}
                      onSelect={handleAddBypassGeoIPCategory}
                      loading={loadingGeoIPCategories}
                      placeholder="Select or type GeoIP category"
                      helperText={`${availableGeoIPCategories.length} GeoIP categories available`}
                    />

                    <Box sx={{ mt: 2 }}>
                      <B4ChipList
                        items={config.targets.geoip_categories}
                        getKey={(c) => c}
                        getLabel={(c) =>
                          renderCategoryLabel(
                            c,
                            stats?.geoip_category_breakdown
                          )
                        }
                        onDelete={handleRemoveBypassGeoIPCategory}
                        title="Active Bypass GeoIP Categories"
                      />
                    </Box>
                  </Box>
                </Grid>
              )}
            </Grid>
          </TabPanel>
        </B4Section>
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
          >
            Close
          </Button>
        }
      >
        {previewDialog.loading ? (
          <Box sx={{ p: 2 }}>
            <Skeleton variant="text" />
            <Skeleton variant="text" />
            <Skeleton variant="text" />
          </Box>
        ) : previewDialog.data ? (
          <>
            <B4Alert severity="info" sx={{ mb: 2 }}>
              Total domains in category: {previewDialog.data.total_domains}
              {previewDialog.data.total_domains >
                previewDialog.data.preview_count &&
                ` (showing first ${previewDialog.data.preview_count})`}
            </B4Alert>
            <List dense sx={{ maxHeight: 600, overflow: "auto" }}>
              {previewDialog.data.preview.map((domain) => (
                <ListItem key={domain}>
                  <ListItemText primary={domain} />
                </ListItem>
              ))}
            </List>
          </>
        ) : (
          <B4Alert severity="error">Failed to load category preview</B4Alert>
        )}
      </B4Dialog>
    </>
  );
};
