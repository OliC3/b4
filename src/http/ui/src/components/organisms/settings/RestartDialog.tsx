import React, { useState } from "react";
import {
  Button,
  Alert,
  CircularProgress,
  Stack,
  Typography,
  LinearProgress,
  Box,
} from "@mui/material";
import {
  RestartAlt as RestartIcon,
  CheckCircle as CheckIcon,
  Error as ErrorIcon,
  Info as InfoIcon,
} from "@mui/icons-material";
import { useSystemRestart } from "@hooks/useSystemRestart";
import { colors, button_primary, button_secondary } from "@design";
import { B4Dialog } from "@molecules/common/B4Dialog";

interface RestartDialogProps {
  open: boolean;
  onClose: () => void;
}

type RestartState = "confirm" | "restarting" | "waiting" | "success" | "error";

export const RestartDialog: React.FC<RestartDialogProps> = ({
  open,
  onClose,
}) => {
  const [state, setState] = useState<RestartState>("confirm");
  const [message, setMessage] = useState("");
  const { restart, waitForReconnection, error } = useSystemRestart();

  const handleRestart = async () => {
    setState("restarting");
    setMessage("Initiating restart...");

    const response = await restart();

    if (response?.success) {
      setState("waiting");
      setMessage("Service is restarting, waiting for reconnection...");

      const reconnected = await waitForReconnection(30);

      if (reconnected) {
        setState("success");
        setMessage("Service restarted successfully!");
        setTimeout(() => globalThis.window.location.reload(), 5000);
      } else {
        setState("error");
        setMessage("Service restart timed out. Please check manually.");
      }
    } else {
      setState("error");
      setMessage(error || "Failed to restart service");
    }
  };

  const handleClose = () => {
    if (state !== "restarting" && state !== "waiting") {
      setState("confirm");
      setMessage("");
      onClose();
    }
  };

  // Dynamic dialog props based on state
  const defaultDeailgoProps = {
    title: "Restart B4 Service",
    subtitle: "System Service Management",
    icon: <RestartIcon />,
  };

  const getDialogProps = () => {
    switch (state) {
      case "confirm":
        return {
          ...defaultDeailgoProps,
          title: "Restart B4 Service",
          subtitle: "System Service Management",
        };
      case "restarting":
      case "waiting":
        return {
          ...defaultDeailgoProps,
          title: "Restarting Service",
          subtitle: "Please wait...",
        };
      case "success":
        return {
          ...defaultDeailgoProps,
          title: "Restart Successful",
          subtitle: "Service is back online",
        };
      case "error":
        return {
          ...defaultDeailgoProps,
          title: "Restart Failed",
          subtitle: "An error occurred",
        };
      default:
        return {
          ...defaultDeailgoProps,
        };
    }
  };

  // Content for each state
  const renderContent = () => {
    switch (state) {
      case "confirm":
        return (
          <Alert
            severity="info"
            icon={<InfoIcon />}
            sx={{
              bgcolor: colors.background.default,
              border: `1px solid ${colors.border.default}`,
              "& .MuiAlert-icon": {
                color: colors.secondary,
              },
            }}
          >
            <Typography variant="body2" sx={{ mb: 1 }}>
              This will restart the B4 service. The web interface will be
              temporarily unavailable during the restart.
            </Typography>
            <Typography variant="caption" sx={{ color: colors.text.secondary }}>
              Expected downtime: 5-10 seconds
            </Typography>
          </Alert>
        );

      case "restarting":
      case "waiting":
        return (
          <Stack spacing={3} alignItems="center" sx={{ py: 4 }}>
            <Box
              sx={{
                p: 2,
                borderRadius: 3,
                bgcolor: colors.accent.secondary,
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
              }}
            >
              <CircularProgress size={48} sx={{ color: colors.secondary }} />
            </Box>
            <Box sx={{ textAlign: "center" }}>
              <Typography
                variant="h6"
                sx={{ color: colors.text.primary, mb: 1 }}
              >
                {message}
              </Typography>
              <Typography
                variant="caption"
                sx={{ color: colors.text.secondary }}
              >
                Please wait, do not close this window...
              </Typography>
            </Box>
            <Box sx={{ width: "100%", px: 2 }}>
              <LinearProgress
                sx={{
                  height: 6,
                  borderRadius: 3,
                  bgcolor: colors.background.dark,
                  "& .MuiLinearProgress-bar": {
                    bgcolor: colors.secondary,
                    borderRadius: 3,
                  },
                }}
              />
            </Box>
          </Stack>
        );

      case "success":
        return (
          <Stack spacing={3} alignItems="center" sx={{ py: 4 }}>
            <Box
              sx={{
                p: 2,
                borderRadius: 3,
                bgcolor: colors.accent.secondary,
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
              }}
            >
              <CheckIcon sx={{ fontSize: 64, color: colors.secondary }} />
            </Box>
            <Box sx={{ textAlign: "center" }}>
              <Typography
                variant="h6"
                sx={{ color: colors.text.primary, mb: 1 }}
              >
                {message}
              </Typography>
              <Typography variant="body2" sx={{ color: colors.text.secondary }}>
                Reloading interface...
              </Typography>
            </Box>
          </Stack>
        );

      case "error":
        return (
          <Stack spacing={3} alignItems="center" sx={{ py: 4 }}>
            <Box
              sx={{
                p: 2,
                borderRadius: 3,
                bgcolor: `${colors.quaternary}22`,
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
              }}
            >
              <ErrorIcon sx={{ fontSize: 64, color: colors.quaternary }} />
            </Box>
            <Box sx={{ textAlign: "center", width: "100%" }}>
              <Typography
                variant="h6"
                sx={{ color: colors.text.primary, mb: 2 }}
              >
                Restart Failed
              </Typography>
              <Alert
                severity="error"
                sx={{
                  bgcolor: colors.background.default,
                  border: `1px solid ${colors.quaternary}44`,
                }}
              >
                {message}
              </Alert>
            </Box>
          </Stack>
        );
    }
  };

  // Actions for each state
  const renderActions = () => {
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
              onClick={handleRestart}
              variant="contained"
              startIcon={<RestartIcon />}
              sx={{
                ...button_primary,
              }}
            >
              Restart Service
            </Button>
          </>
        );

      case "error":
        return (
          <Button
            onClick={handleClose}
            variant="contained"
            sx={{
              bgcolor: colors.secondary,
              color: colors.background.default,
              "&:hover": { bgcolor: colors.primary },
            }}
          >
            Close
          </Button>
        );

      default:
        return null;
    }
  };

  const dialogProps = getDialogProps();

  return (
    <B4Dialog
      {...getDialogProps()}
      open={open}
      onClose={handleClose}
      maxWidth="sm"
      fullWidth
      actions={renderActions()}
    >
      {renderContent()}
    </B4Dialog>
  );
};
