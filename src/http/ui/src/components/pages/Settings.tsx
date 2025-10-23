import { useEffect, useState } from "react";
import {
  Container,
  Box,
  Stack,
  Button,
  Alert,
  Snackbar,
  CircularProgress,
  Typography,
  Grid,
} from "@mui/material";
import { Save as SaveIcon, Refresh as RefreshIcon } from "@mui/icons-material";

import { NetworkSettings } from "../organisms/settings/Network";
import { LoggingSettings } from "../organisms/settings/Logging";
import { FeatureSettings } from "../organisms/settings/Feature";
import { DomainSettings } from "../organisms/settings/Domain";
import { FragmentationSettings } from "../organisms/settings/Fragmentation";
import { FakingSettings } from "../organisms/settings/Faking";
import { UDPSettings } from "../organisms/settings/Udp";

import B4Config from "../../models/Config";

import { colors } from "../../Theme";

export default function Settings() {
  const [config, setConfig] = useState<B4Config | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const [snackbar, setSnackbar] = useState<{
    open: boolean;
    message: string;
    severity: "success" | "error" | "info";
  }>({
    open: false,
    message: "",
    severity: "info",
  });

  useEffect(() => {
    loadConfig();
  }, []);

  const loadConfig = async () => {
    try {
      setLoading(true);
      const response = await fetch("/api/config");
      if (!response.ok) throw new Error("Failed to load configuration");
      const data = await response.json();
      console.log("Loaded configuration:", data);
      setConfig(data);
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
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(config),
      });

      if (!response.ok) {
        const error = await response.text();
        throw new Error(error || "Failed to save configuration");
      }

      setSnackbar({
        open: true,
        message:
          "Configuration saved successfully! Restart required for some changes to take effect.",
        severity: "success",
      });
    } catch (error) {
      setSnackbar({
        open: true,
        message:
          error instanceof Error
            ? error.message
            : "Failed to save configuration",
        severity: "error",
      });
    } finally {
      setSaving(false);
    }
  };

  const handleChange = (field: string, value: any) => {
    if (!config) return;

    // Handle nested fields (e.g., "logging.level")
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
      setConfig({
        ...config,
        [field]: value,
      });
    }
  };

  if (loading || !config) {
    return (
      <Container maxWidth={false} sx={{ py: 3, textAlign: "center" }}>
        <CircularProgress sx={{ mt: 8 }} />
        <Typography sx={{ mt: 2, color: colors.text.secondary }}>
          Loading configuration...
        </Typography>
      </Container>
    );
  }

  return (
    <Container
      maxWidth={false}
      sx={{
        flex: 1,
        pb: 3,
        display: "flex",
        flexDirection: "column",
        overflow: "auto",
      }}
    >
      {/* Action Bar */}
      <Box
        sx={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          position: "sticky",
          top: 0,
          bgcolor: colors.background.default,
          zIndex: 10,
        }}
      >
        <Alert severity="info" sx={{ mb: 3 }}>
          Changes to network, queue, and thread settings require B<sup>4</sup> a
          restart to take effect.
        </Alert>
        <Stack direction="row" spacing={2}>
          <Button
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
            variant="contained"
            startIcon={saving ? <CircularProgress size={20} /> : <SaveIcon />}
            onClick={saveConfig}
            disabled={saving}
            sx={{
              bgcolor: colors.secondary,
              color: colors.background.default,
              "&:hover": {
                bgcolor: colors.primary,
              },
            }}
          >
            {saving ? "Saving..." : "Save Changes"}
          </Button>
        </Stack>
      </Box>

      {/* Settings Sections */}
      <Stack spacing={3}>
        <Grid container spacing={3}>
          <Grid size={{ xs: 12, lg: 6 }}>
            <NetworkSettings config={config} onChange={handleChange} />
          </Grid>
          <Grid size={{ xs: 12, lg: 6 }}>
            <FeatureSettings config={config} onChange={handleChange} />
          </Grid>
        </Grid>

        <DomainSettings config={config} onChange={handleChange} />
        <FragmentationSettings config={config} onChange={handleChange} />
        <FakingSettings config={config} onChange={handleChange} />
        <UDPSettings config={config} onChange={handleChange} />
        <LoggingSettings config={config} onChange={handleChange} />
      </Stack>

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
