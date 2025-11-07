import React, { useState } from "react";
import {
  Button,
  Alert,
  Stack,
  Typography,
  Box,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  CircularProgress,
} from "@mui/material";
import {
  RestartAlt as ResetIcon,
  CheckCircle as CheckIcon,
  Error as ErrorIcon,
  Warning as WarningIcon,
  Shield as ShieldIcon,
} from "@mui/icons-material";
import { useConfigReset } from "@hooks/useConfig";
import { colors, button_primary, button_secondary } from "@design";
import { B4Dialog } from "@molecules/common/B4Dialog";

interface ResetDialogProps {
  open: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

type ResetState = "confirm" | "resetting" | "success" | "error";

export const ResetDialog: React.FC<ResetDialogProps> = ({
  open,
  onClose,
  onSuccess,
}) => {
  const [state, setState] = useState<ResetState>("confirm");
  const [message, setMessage] = useState("");
  const { resetConfig, loading } = useConfigReset();

  const handleReset = async () => {
    setState("resetting");
    setMessage("Resetting configuration...");

    const response = await resetConfig();

    if (response && response.success) {
      setState("success");
      setMessage("Configuration reset successfully!");
      setTimeout(() => {
        handleClose();
        onSuccess();
      }, 2000);
    } else {
      setState("error");
      setMessage("Failed to reset configuration");
    }
  };

  const handleClose = () => {
    if (state !== "resetting") {
      setState("confirm");
      setMessage("");
      onClose();
    }
  };

  const defaultProps = {
    title: "Reset Configuration",
    subtitle: "Restore default settings",
    icon: <ResetIcon />,
  };

  // Dynamic dialog props based on state
  const getDialogProps = () => {
    switch (state) {
      case "confirm":
        return {
          ...defaultProps,
          title: "Restart B4 Service",
          subtitle: "System Service Management",
        };
      case "resetting":
        return {
          ...defaultProps,
          title: "Resetting Configuration",
          subtitle: "Please wait...",
          icon: <CircularProgress size={24} />,
        };
      case "success":
        return {
          ...defaultProps,
          title: "Restart Successful",
          subtitle: "Service is back online",
        };
      case "error":
        return {
          ...defaultProps,
          title: "Restart Failed",
          subtitle: "An error occurred",
          icon: <ErrorIcon />,
        };
      default:
        return {
          ...defaultProps,
        };
    }
  };

  const getDialogActions = () => {
    switch (state) {
      case "confirm":
        return (
          <>
            <Button
              onClick={handleClose}
              sx={{
                ...button_secondary,
              }}
            >
              Cancel
            </Button>
            <Box sx={{ flex: 1 }} />
            <Button
              onClick={handleReset}
              variant="contained"
              startIcon={<ResetIcon />}
              sx={{
                ...button_primary,
              }}
            >
              Reset to Defaults
            </Button>
          </>
        );
      case "error":
        return (
          <Button onClick={handleClose} variant="contained">
            Close
          </Button>
        );

      case "success":
      default:
        return null;
    }
  };

  const getDialogContent = () => {
    switch (state) {
      case "confirm":
        return (
          <>
            <Alert
              severity="warning"
              icon={<WarningIcon />}
              sx={{
                bgcolor: colors.background.default,
                border: `1px solid ${colors.quaternary}44`,
                mb: 3,
              }}
            >
              <Typography variant="body2" sx={{ mb: 1 }}>
                This will reset all configuration to default values except:
              </Typography>
            </Alert>

            <List dense>
              <ListItem>
                <ListItemIcon>
                  <ShieldIcon sx={{ color: colors.secondary }} />
                </ListItemIcon>
                <ListItemText
                  primary="Domain Configuration"
                  secondary="All domain filters and geodata settings will be preserved"
                />
              </ListItem>
              <ListItem>
                <ListItemIcon>
                  <ShieldIcon sx={{ color: colors.secondary }} />
                </ListItemIcon>
                <ListItemText
                  primary="Testing Configuration"
                  secondary="Checker settings and test domains will be preserved"
                />
              </ListItem>
            </List>

            <Alert
              severity="info"
              sx={{
                mt: 2,
                bgcolor: colors.background.default,
                border: `1px solid ${colors.border.default}`,
              }}
            >
              <Typography variant="caption">
                Network, DPI bypass, protocol, and logging settings will be
                reset to defaults. You may need to restart B4 for some changes
                to take effect.
              </Typography>
            </Alert>
          </>
        );

      case "resetting":
        return (
          <Stack spacing={3} alignItems="center" sx={{ py: 4 }}>
            <CircularProgress size={48} sx={{ color: colors.secondary }} />
            <Typography variant="h6" sx={{ color: colors.text.primary }}>
              {message}
            </Typography>
          </Stack>
        );

      case "success":
        return (
          <Stack spacing={3} alignItems="center" sx={{ py: 4 }}>
            <CheckIcon
              sx={{
                fontSize: 64,
                color: colors.secondary,
              }}
            />
            <Typography variant="h6" sx={{ color: colors.text.primary }}>
              {message}
            </Typography>
          </Stack>
        );

      case "error":
        return (
          <Stack spacing={3} alignItems="center" sx={{ py: 4 }}>
            <ErrorIcon sx={{ fontSize: 64, color: colors.quaternary }} />
            <Alert severity="error" sx={{ width: "100%" }}>
              {message}
            </Alert>
          </Stack>
        );
    }
  };

  return (
    <B4Dialog
      {...getDialogProps()}
      open={open}
      onClose={handleClose}
      actions={getDialogActions()}
    >
      {getDialogContent()}
    </B4Dialog>
  );
};
