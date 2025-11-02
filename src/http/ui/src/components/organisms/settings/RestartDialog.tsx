import React, { useState } from "react";
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Alert,
  CircularProgress,
  Stack,
  Typography,
  LinearProgress,
  Box,
  Divider,
} from "@mui/material";
import {
  RestartAlt as RestartIcon,
  CheckCircle as CheckIcon,
  Error as ErrorIcon,
  Info as InfoIcon,
} from "@mui/icons-material";
import { useSystemRestart } from "../../../hooks/useSystemRestart";
import { colors } from "../../../Theme";

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

    if (response && response.success) {
      setState("waiting");
      setMessage("Service is restarting, waiting for reconnection...");

      // Wait for service to come back online
      const reconnected = await waitForReconnection(30);

      if (reconnected) {
        setState("success");
        setMessage("Service restarted successfully!");

        // Auto-close and reload after success
        setTimeout(() => {
          window.location.reload();
        }, 2000);
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

  const getDialogContent = () => {
    switch (state) {
      case "confirm":
        return (
          <>
            <DialogContent sx={{ mt: 2 }}>
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
                <Typography
                  variant="caption"
                  sx={{ color: colors.text.secondary }}
                >
                  Expected downtime: 5-10 seconds
                </Typography>
              </Alert>
            </DialogContent>

            <Divider sx={{ borderColor: colors.border.default }} />

            <DialogActions sx={{ p: 2 }}>
              <Button
                onClick={handleClose}
                sx={{
                  color: colors.text.secondary,
                  "&:hover": {
                    bgcolor: colors.accent.primaryHover,
                  },
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
                  bgcolor: colors.primary,
                  "&:hover": {
                    bgcolor: colors.secondary,
                  },
                }}
              >
                Restart Service
              </Button>
            </DialogActions>
          </>
        );

      case "restarting":
      case "waiting":
        return (
          <>
            <DialogContent sx={{ mt: 2 }}>
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
                  <CircularProgress
                    size={48}
                    sx={{
                      color: colors.secondary,
                    }}
                  />
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
            </DialogContent>
          </>
        );

      case "success":
        return (
          <>
            <DialogContent sx={{ mt: 2 }}>
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
                  <CheckIcon
                    sx={{
                      fontSize: 64,
                      color: colors.secondary,
                    }}
                  />
                </Box>
                <Box sx={{ textAlign: "center" }}>
                  <Typography
                    variant="h6"
                    sx={{ color: colors.text.primary, mb: 1 }}
                  >
                    {message}
                  </Typography>
                  <Typography
                    variant="body2"
                    sx={{ color: colors.text.secondary }}
                  >
                    Reloading interface...
                  </Typography>
                </Box>
              </Stack>
            </DialogContent>
          </>
        );

      case "error":
        return (
          <>
            <DialogContent sx={{ mt: 2 }}>
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
                  <ErrorIcon
                    sx={{
                      fontSize: 64,
                      color: colors.quaternary,
                    }}
                  />
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
            </DialogContent>

            <Divider sx={{ borderColor: colors.border.default }} />

            <DialogActions sx={{ p: 2 }}>
              <Button
                onClick={handleClose}
                variant="contained"
                sx={{
                  bgcolor: colors.secondary,
                  color: colors.background.default,
                  "&:hover": {
                    bgcolor: colors.primary,
                  },
                }}
              >
                Close
              </Button>
            </DialogActions>
          </>
        );
    }
  };

  return (
    <Dialog
      open={open}
      onClose={handleClose}
      maxWidth="sm"
      fullWidth
      disableEscapeKeyDown={state === "restarting" || state === "waiting"}
      PaperProps={{
        sx: {
          bgcolor: colors.background.paper,
          border: `2px solid ${colors.border.default}`,
          borderRadius: 4,
        },
      }}
    >
      <DialogTitle
        sx={{
          bgcolor: colors.background.dark,
          color: colors.text.primary,
          borderBottom: `1px solid ${colors.border.default}`,
        }}
      >
        <Stack direction="row" alignItems="center" spacing={2}>
          <Box
            sx={{
              p: 1.5,
              borderRadius: 2,
              bgcolor: colors.accent.secondary,
              color: colors.secondary,
              display: "flex",
              alignItems: "center",
            }}
          >
            <RestartIcon />
          </Box>
          <Box>
            <Typography sx={{ mt: 1.5, lineHeight: 0 }}>
              Restart B4 Service
            </Typography>
            <Typography
              variant="caption"
              sx={{
                color: colors.text.secondary,
              }}
            >
              System Service Management
            </Typography>
          </Box>
        </Stack>
      </DialogTitle>
      {getDialogContent()}
    </Dialog>
  );
};
