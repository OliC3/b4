import { useEffect, useState, useMemo } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import {
  Container,
  Box,
  Stack,
  Button,
  Alert,
  Snackbar,
  CircularProgress,
  Typography,
  Tabs,
  Tab,
  Paper,
  Chip,
  Fade,
  Backdrop,
  DialogContent,
  DialogContentText,
  Grid,
} from "@mui/material";
import {
  Save as SaveIcon,
  Refresh as RefreshIcon,
  Settings as SettingsIcon,
  Warning as WarningIcon,
  Science as DiscoveryIcon,
  Language as LanguageIcon,
  Cloud as ApiIcon,
  CameraAlt as CaptureIcon,
} from "@mui/icons-material";
import { CaptureSettings } from "@organisms/settings/Capture";
import { NetworkSettings } from "@organisms/settings/Network";
import { LoggingSettings } from "@organisms/settings/Logging";
import { FeatureSettings } from "@organisms/settings/Feature";
import { CheckerSettings } from "@/components/organisms/settings/Discovery";
import { ControlSettings } from "@organisms/settings/Control";
import { GeoSettings } from "@organisms/settings/Geo";
import { ApiSettings } from "@organisms/settings/Api";

import { B4Config, B4SetConfig } from "@models/Config";
import { colors, spacing, button_primary, button_secondary } from "@design";
import { B4Dialog } from "@molecules/common/B4Dialog";

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel({
  children,
  value,
  index,
  ...other
}: Readonly<TabPanelProps>) {
  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`settings-tabpanel-${index}`}
      aria-labelledby={`settings-tab-${index}`}
      {...other}
    >
      {value === index && (
        <Fade in>{<Box sx={{ pt: 3 }}>{children}</Box>}</Fade>
      )}
    </div>
  );
}

enum TABS {
  GENERAL = 0,
  DOMAINS,
  DISCOVERY,
  API,
  CAPTURE,
}

// Settings categories with route paths
const SETTING_CATEGORIES = [
  {
    id: TABS.GENERAL,
    path: "general",
    label: "Core",
    icon: <SettingsIcon />,
    description: "Global network and queue configuration",
    requiresRestart: true,
  },
  {
    id: TABS.DOMAINS,
    path: "domains",
    label: "Geodat Settings",
    icon: <LanguageIcon />,
    description: "Global geodata configuration",
    requiresRestart: false,
  },
  {
    id: TABS.DISCOVERY,
    path: "discovery",
    label: "Discovery",
    icon: <DiscoveryIcon />,
    description: "DPI bypass domains testing",
    requiresRestart: false,
  },
  {
    id: TABS.API,
    path: "api",
    label: "API",
    icon: <ApiIcon />,
    description: "API settings for various services",
    requiresRestart: false,
  },
  {
    id: TABS.CAPTURE,
    path: "capture",
    label: "Capture",
    icon: <CaptureIcon />,
    description: "Capture real payloads from live traffic",
    requiresRestart: false,
  },
];

export default function Settings() {
  const [config, setConfig] = useState<B4Config | null>(null);
  const [originalConfig, setOriginalConfig] = useState<B4Config | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [showResetDialog, setShowResetDialog] = useState(false);

  const navigate = useNavigate();
  const location = useLocation();

  // Determine current tab based on URL
  const currentTabPath = location.pathname.split("/settings/")[1] || "general";
  const currentTab =
    SETTING_CATEGORIES.find((cat) => cat.path === currentTabPath)?.id ??
    TABS.GENERAL;

  // Handle tab change
  const handleTabChange = (_: React.SyntheticEvent, newValue: number) => {
    const category = SETTING_CATEGORIES.find(
      (cat) => cat.id === (newValue as TABS)
    );
    if (category) {
      navigate(`/settings/${category.path}`);
    }
  };

  // Navigate to default tab if no specific tab is in URL
  useEffect(() => {
    if (
      location.pathname === "/settings" ||
      location.pathname === "/settings/"
    ) {
      navigate("/settings/general", { replace: true });
    }
  }, [location.pathname, navigate]);

  const [snackbar, setSnackbar] = useState<{
    open: boolean;
    message: string;
    severity: "success" | "error" | "info";
  }>({
    open: false,
    message: "",
    severity: "info",
  });

  // Check if configuration has been modified
  const hasChanges = useMemo(() => {
    if (!config || !originalConfig) return false;
    return JSON.stringify(config) !== JSON.stringify(originalConfig);
  }, [config, originalConfig]);

  // Check which categories have changes
  const categoryHasChanges = useMemo(() => {
    if (!hasChanges || !config || !originalConfig) return {};

    return {
      // Core
      [TABS.GENERAL]:
        JSON.stringify(config.system.logging) !==
          JSON.stringify(originalConfig.system.logging) ||
        JSON.stringify(config.queue) !== JSON.stringify(originalConfig.queue) ||
        JSON.stringify(config.system.web_server) !==
          JSON.stringify(originalConfig.system.web_server) ||
        JSON.stringify(config.system.tables) !==
          JSON.stringify(originalConfig.system.tables),

      // Geosite Settings
      [TABS.DOMAINS]:
        JSON.stringify(config.system.geo) !==
        JSON.stringify(originalConfig.system.geo),

      // Discovery
      [TABS.DISCOVERY]:
        JSON.stringify(config.system.checker) !==
        JSON.stringify(originalConfig.system.checker),

      // API
      [TABS.API]:
        JSON.stringify(config.system.api) !==
        JSON.stringify(originalConfig.system.api),
    };
  }, [config, originalConfig, hasChanges]);

  useEffect(() => {
    void loadConfig();
  }, []);

  const loadConfig = async () => {
    try {
      setLoading(true);
      const response = await fetch("/api/config");
      if (!response.ok) throw new Error("Failed to load configuration");
      const data = (await response.json()) as unknown as B4Config & {};
      setConfig(data);
      setOriginalConfig(structuredClone(data)); // Deep clone
    } catch (error) {
      console.error("Error loading configuration:", error);
      setSnackbar({
        open: true,
        message: "Failed to load configuration",
        severity: "error",
      });
    } finally {
      setLoading(false);
    }
  };

  const saveConfig = async () => {
    if (!config) return;

    try {
      setSaving(true);
      const response = await fetch("/api/config", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(config),
      });

      if (!response.ok) {
        const error = await response.text();
        throw new Error(error || "Failed to save configuration");
      }

      setOriginalConfig(structuredClone(config));

      const requiresRestart = categoryHasChanges[0]; // Core settings require restart
      setSnackbar({
        open: true,
        message: requiresRestart
          ? "Configuration saved! Please restart B4 for core settings to take effect."
          : "Configuration saved successfully!",
        severity: "success",
      });
    } catch (error) {
      setSnackbar({
        open: true,
        message: error instanceof Error ? error.message : "Failed to save",
        severity: "error",
      });
    } finally {
      setSaving(false);
      await loadConfig();
    }
  };

  const resetChanges = () => {
    if (originalConfig) {
      setConfig(structuredClone(originalConfig));
      setShowResetDialog(false);
      setSnackbar({
        open: true,
        message: "Changes discarded",
        severity: "info",
      });
    }
  };

  const handleChange = (
    field: string,
    value:
      | string
      | number
      | boolean
      | string[]
      | B4SetConfig[]
      | null
      | undefined
  ) => {
    if (!config) return;

    const keys = field.split(".");

    if (keys.length === 1) {
      setConfig({ ...config, [field]: value });
    } else {
      const newConfig = { ...config };
      let current: Record<string, unknown> = newConfig;

      for (let i = 0; i < keys.length - 1; i++) {
        current[keys[i]] = { ...(current[keys[i]] as object) };
        current = current[keys[i]] as Record<string, unknown>;
      }

      current[keys[keys.length - 1]] = value;
      setConfig(newConfig);
    }
  };

  if (loading || !config) {
    return (
      <Backdrop open sx={{ zIndex: 9999 }}>
        <Stack alignItems="center" spacing={2}>
          <CircularProgress sx={{ color: colors.secondary }} />
          <Typography sx={{ color: colors.text.primary }}>
            Loading configuration...
          </Typography>
        </Stack>
      </Backdrop>
    );
  }

  const validTab = Math.max(currentTab, 0);

  return (
    <Container
      maxWidth={false}
      sx={{
        height: "100%",
        display: "flex",
        flexDirection: "column",
        overflow: "hidden",
        py: 3,
      }}
    >
      {/* Header with tabs */}
      <Paper
        elevation={0}
        sx={{
          bgcolor: colors.background.paper,
          borderRadius: 2,
          border: `1px solid ${colors.border.default}`,
        }}
      >
        <Box sx={{ p: 2, pb: 0 }}>
          {/* Action bar */}
          <Stack
            direction="row"
            justifyContent="space-between"
            alignItems="center"
            sx={{ mb: 2 }}
          >
            <Stack direction="row" spacing={2} alignItems="center">
              <Typography variant="h6" sx={{ color: colors.text.primary }}>
                Configuration
              </Typography>
              {hasChanges && (
                <Chip
                  label="Modified"
                  size="small"
                  icon={<WarningIcon />}
                  sx={{
                    bgcolor: colors.accent.secondary,
                    color: colors.secondary,
                  }}
                />
              )}
            </Stack>

            <Stack direction="row" spacing={1}>
              {categoryHasChanges[TABS.GENERAL] && (
                <Alert severity="warning" sx={{ py: 0, px: spacing.sm }}>
                  Core settings require <strong>B4</strong> restart to take
                  effect
                </Alert>
              )}
              <Button
                size="small"
                variant="text"
                onClick={() => setShowResetDialog(true)}
                disabled={!hasChanges || saving}
                sx={{
                  color: colors.text.secondary,
                  "&:hover": { bgcolor: colors.accent.primaryHover },
                }}
              >
                Discard Changes
              </Button>
              <Button
                size="small"
                variant="outlined"
                startIcon={<RefreshIcon />}
                onClick={() => {
                  void loadConfig();
                }}
                disabled={saving}
                sx={{
                  borderColor: colors.border.default,
                  color: colors.text.primary,
                  "&:hover": {
                    borderColor: colors.secondary,
                    bgcolor: colors.accent.secondaryHover,
                  },
                }}
              >
                Reload
              </Button>

              <Button
                size="small"
                variant="contained"
                startIcon={
                  saving ? <CircularProgress size={16} /> : <SaveIcon />
                }
                onClick={() => {
                  void saveConfig();
                }}
                disabled={!hasChanges || saving}
                sx={{
                  bgcolor: colors.secondary,
                  color: colors.background.default,
                  "&:hover": { bgcolor: colors.primary },
                  "&:disabled": {
                    bgcolor: colors.accent.secondary,
                    color: colors.text.secondary,
                  },
                }}
              >
                {saving ? "Saving..." : "Save Changes"}
              </Button>
            </Stack>
          </Stack>

          {/* Tabs */}
          <Tabs
            value={validTab}
            onChange={handleTabChange}
            variant="scrollable"
            scrollButtons="auto"
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
            {SETTING_CATEGORIES.sort(
              (a, b) => (a.id as number) - (b.id as number)
            ).map((category) => (
              <Tab
                key={category.id}
                label={
                  <Stack direction="row" spacing={1} alignItems="center">
                    {category.icon}
                    <span>{category.label}</span>
                    {(categoryHasChanges as Record<number, boolean>)[
                      category.id
                    ] && (
                      <Box
                        sx={{
                          width: 6,
                          height: 6,
                          borderRadius: "50%",
                          bgcolor: colors.secondary,
                        }}
                      />
                    )}
                  </Stack>
                }
              />
            ))}
          </Tabs>
        </Box>
      </Paper>

      {/* Tab Content */}
      <Box sx={{ flex: 1, overflow: "auto", pb: 2 }}>
        {/* Core Settings */}
        <TabPanel value={validTab} index={TABS.GENERAL}>
          <Grid container spacing={spacing.lg}>
            <Grid size={{ xs: 12, md: 12 }}>
              <NetworkSettings config={config} onChange={handleChange} />
            </Grid>
            <Grid size={{ xs: 12, md: 6 }}>
              <Stack spacing={spacing.lg}>
                <ControlSettings
                  loadConfig={() => {
                    void loadConfig();
                  }}
                />
                <LoggingSettings config={config} onChange={handleChange} />
              </Stack>
            </Grid>
            <Grid size={{ xs: 12, md: 6 }}>
              <FeatureSettings config={config} onChange={handleChange} />
            </Grid>
          </Grid>
        </TabPanel>

        <TabPanel value={validTab} index={TABS.DOMAINS}>
          <GeoSettings
            config={config}
            onChange={handleChange}
            loadConfig={() => {
              void loadConfig();
            }}
          />
        </TabPanel>

        <TabPanel value={validTab} index={TABS.API}>
          <ApiSettings config={config} onChange={handleChange} />
        </TabPanel>

        <TabPanel value={validTab} index={TABS.DISCOVERY}>
          <CheckerSettings config={config} onChange={handleChange} />
        </TabPanel>

        <TabPanel value={validTab} index={TABS.CAPTURE}>
          <CaptureSettings />
        </TabPanel>
      </Box>

      {/* Reset Confirmation Dialog */}
      <B4Dialog
        title="Discard changes"
        open={showResetDialog}
        onClose={() => setShowResetDialog(false)}
        actions={
          <>
            <Button
              variant="outlined"
              onClick={() => setShowResetDialog(false)}
              sx={{ ...button_secondary }}
            >
              Cancel
            </Button>
            <Box sx={{ flex: 1 }} />
            <Button
              onClick={resetChanges}
              variant="contained"
              sx={{ ...button_primary }}
            >
              Discard Changes
            </Button>
          </>
        }
      >
        <DialogContent>
          <DialogContentText>
            Are you sure you want to discard all unsaved changes? This action
            cannot be undone.
          </DialogContentText>
        </DialogContent>
      </B4Dialog>

      {/* Snackbar */}
      <Snackbar
        open={snackbar.open}
        autoHideDuration={6000}
        onClose={() => setSnackbar({ ...snackbar, open: false })}
        anchorOrigin={{ vertical: "bottom", horizontal: "right" }}
      >
        <Alert
          onClose={() => setSnackbar({ ...snackbar, open: false })}
          severity={snackbar.severity}
          sx={{ width: "100%" }}
        >
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Container>
  );
}
