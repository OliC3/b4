import { useEffect, useState, useMemo } from "react";
import { useNavigate, useLocation, useParams } from "react-router-dom";
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
  Dialog,
  DialogTitle,
  DialogContent,
  DialogContentText,
  DialogActions,
} from "@mui/material";
import {
  Save as SaveIcon,
  Refresh as RefreshIcon,
  Settings as SettingsIcon,
  Language as NetworkIcon,
  Security as SecurityIcon,
  Storage as StorageIcon,
  Description as LogIcon,
  Warning as WarningIcon,
} from "@mui/icons-material";

import { NetworkSettings } from "../organisms/settings/Network";
import { LoggingSettings } from "../organisms/settings/Logging";
import { FeatureSettings } from "../organisms/settings/Feature";
import { DomainSettings } from "../organisms/settings/Domain";
import { FragmentationSettings } from "../organisms/settings/Fragmentation";
import { FakingSettings } from "../organisms/settings/Faking";
import { UDPSettings } from "../organisms/settings/Udp";

import B4Config from "../../models/Config";
import { colors } from "../../Theme";

// Tab panel component
interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel({ children, value, index, ...other }: TabPanelProps) {
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

// Settings categories with route paths
const SETTING_CATEGORIES = [
  {
    id: 0,
    path: "core",
    label: "Core",
    icon: <SettingsIcon />,
    description: "Essential network and system configuration",
    requiresRestart: true,
  },
  {
    id: 1,
    path: "domains",
    label: "Domains",
    icon: <NetworkIcon />,
    description: "Domain filtering and geodata configuration",
    requiresRestart: false,
  },
  {
    id: 2,
    path: "dpi",
    label: "DPI Bypass",
    icon: <SecurityIcon />,
    description: "Fragmentation and faking strategies",
    requiresRestart: false,
  },
  {
    id: 3,
    path: "proto",
    label: "Protocols",
    icon: <StorageIcon />,
    description: "UDP and protocol-specific settings",
    requiresRestart: false,
  },
  {
    id: 4,
    path: "logging",
    label: "Logging",
    icon: <LogIcon />,
    description: "Logging and debugging configuration",
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
  const currentTabPath = location.pathname.split("/settings/")[1] || "core";
  const currentTab = SETTING_CATEGORIES.findIndex(
    (cat) => cat.path === currentTabPath
  );

  // Handle tab change
  const handleTabChange = (_: React.SyntheticEvent, newValue: number) => {
    const category = SETTING_CATEGORIES[newValue];
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
      navigate("/settings/core", { replace: true });
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
      0:
        config.queue_start_num !== originalConfig.queue_start_num ||
        config.threads !== originalConfig.threads ||
        config.mark !== originalConfig.mark ||
        config.conn_bytes_limit !== originalConfig.conn_bytes_limit ||
        config.seg2delay !== originalConfig.seg2delay ||
        config.use_gso !== originalConfig.use_gso ||
        config.use_conntrack !== originalConfig.use_conntrack ||
        config.skip_iptables !== originalConfig.skip_iptables ||
        config.ipv4 !== originalConfig.ipv4 ||
        config.ipv6 !== originalConfig.ipv6,
      // Domains
      1:
        JSON.stringify(config.domains) !==
        JSON.stringify(originalConfig.domains),
      // DPI Bypass
      2:
        JSON.stringify(config.fragmentation) !==
          JSON.stringify(originalConfig.fragmentation) ||
        JSON.stringify(config.faking) !== JSON.stringify(originalConfig.faking),
      // Protocols
      3: JSON.stringify(config.udp) !== JSON.stringify(originalConfig.udp),
      // Logging
      4:
        JSON.stringify(config.logging) !==
        JSON.stringify(originalConfig.logging),
    };
  }, [config, originalConfig, hasChanges]);

  useEffect(() => {
    loadConfig();
  }, []);

  const loadConfig = async () => {
    try {
      setLoading(true);
      const response = await fetch("/api/config");
      if (!response.ok) throw new Error("Failed to load configuration");
      const data = await response.json();
      setConfig(data);
      setOriginalConfig(JSON.parse(JSON.stringify(data))); // Deep clone
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

      setOriginalConfig(JSON.parse(JSON.stringify(config)));

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
      setConfig(JSON.parse(JSON.stringify(originalConfig)));
      setShowResetDialog(false);
      setSnackbar({
        open: true,
        message: "Changes discarded",
        severity: "info",
      });
    }
  };

  const handleChange = (field: string, value: any) => {
    if (!config) return;

    if (field.includes(".")) {
      const [parent, child] = field.split(".");
      setConfig({
        ...config,
        [parent]: {
          ...(config[parent as keyof B4Config] as any),
          [child]: value,
        },
      });
    } else {
      setConfig({ ...config, [field]: value });
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

  const validTab = currentTab >= 0 ? currentTab : 0;

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
          mb: 3,
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
                onClick={loadConfig}
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
                onClick={saveConfig}
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
            {SETTING_CATEGORIES.map((category) => (
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
        <TabPanel value={validTab} index={0}>
          <Stack spacing={3}>
            {categoryHasChanges[0] && (
              <Alert severity="warning" icon={<WarningIcon />}>
                Core settings require B4 restart to take effect
              </Alert>
            )}
            <NetworkSettings config={config} onChange={handleChange} />
            <FeatureSettings config={config} onChange={handleChange} />
          </Stack>
        </TabPanel>

        {/* Domain Settings */}
        <TabPanel value={validTab} index={1}>
          <DomainSettings config={config} onChange={handleChange} />
        </TabPanel>

        {/* DPI Bypass Settings */}
        <TabPanel value={validTab} index={2}>
          <Stack spacing={3}>
            <FragmentationSettings config={config} onChange={handleChange} />
            <FakingSettings config={config} onChange={handleChange} />
          </Stack>
        </TabPanel>

        {/* Protocol Settings */}
        <TabPanel value={validTab} index={3}>
          <UDPSettings config={config} onChange={handleChange} />
        </TabPanel>

        {/* Logging Settings */}
        <TabPanel value={validTab} index={4}>
          <LoggingSettings config={config} onChange={handleChange} />
        </TabPanel>
      </Box>

      {/* Reset Confirmation Dialog */}
      <Dialog open={showResetDialog} onClose={() => setShowResetDialog(false)}>
        <DialogTitle>Discard Changes?</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Are you sure you want to discard all unsaved changes? This action
            cannot be undone.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowResetDialog(false)}>Cancel</Button>
          <Button onClick={resetChanges} color="warning" variant="contained">
            Discard Changes
          </Button>
        </DialogActions>
      </Dialog>

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
