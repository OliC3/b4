import React, { useState } from "react";
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Typography,
  Box,
  Link,
  Divider,
  Chip,
  Stack,
  LinearProgress,
  Alert,
} from "@mui/material";
import {
  NewReleases as NewReleasesIcon,
  Close as CloseIcon,
  OpenInNew as OpenInNewIcon,
  Description as DescriptionIcon,
  CloudDownload as CloudDownloadIcon,
  CheckCircle as CheckCircleIcon,
} from "@mui/icons-material";
import { colors } from "../../../Theme";
import ReactMarkdown from "react-markdown";
import { useSystemUpdate } from "../../../hooks/useSystemUpdate";

interface UpdateModalProps {
  open: boolean;
  onClose: () => void;
  onDismiss: () => void;
  currentVersion: string;
  latestVersion: string;
  releaseNotes: string;
  releaseUrl: string;
  publishedAt: string;
}

export const UpdateModal: React.FC<UpdateModalProps> = ({
  open,
  onClose,
  onDismiss,
  currentVersion,
  latestVersion,
  releaseNotes,
  releaseUrl,
  publishedAt,
}) => {
  const { performUpdate, waitForReconnection, loading, error } =
    useSystemUpdate();
  const [updateStatus, setUpdateStatus] = useState<
    "idle" | "updating" | "reconnecting" | "success" | "error"
  >("idle");
  const [updateMessage, setUpdateMessage] = useState("");

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleDateString("en-US", {
      year: "numeric",
      month: "long",
      day: "numeric",
    });
  };

  const handleUpdate = async () => {
    setUpdateStatus("updating");
    setUpdateMessage("Initiating update...");

    const result = await performUpdate(latestVersion);
    if (!result || !result.success) {
      setUpdateStatus("error");
      setUpdateMessage(
        result?.message || error || "Failed to initiate update."
      );
      return;
    }

    setUpdateMessage("Update in progress. Waiting for service to restart...");
    setUpdateStatus("reconnecting");

    // Wait for service to come back online
    const reconnected = await waitForReconnection();

    if (reconnected) {
      setUpdateStatus("success");
      setUpdateMessage("Update completed successfully! Refreshing...");

      // Reload the page after successful update to get new version
      setTimeout(() => {
        window.location.reload();
      }, 5000);
    } else {
      setUpdateStatus("error");
      setUpdateMessage(
        "Update may have completed but service did not restart. Please check manually."
      );
    }
  };

  const isUpdating =
    updateStatus === "updating" || updateStatus === "reconnecting";
  const showUpdateProgress = updateStatus !== "idle";

  return (
    <Dialog
      open={open}
      onClose={isUpdating ? undefined : onClose}
      maxWidth="md"
      fullWidth
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
              p: 1,
              borderRadius: 2,
              bgcolor: colors.accent.secondary,
              color: colors.secondary,
              display: "flex",
              alignItems: "center",
            }}
          >
            <NewReleasesIcon />
          </Box>
          <Box sx={{ flex: 1 }}>
            <Typography sx={{ mt: 1.5, lineHeight: 0 }}>
              New Version Available
            </Typography>
            <Typography variant="caption" sx={{ color: colors.text.secondary }}>
              Published on {formatDate(publishedAt)}
            </Typography>
          </Box>
          <Stack direction="row" spacing={1}>
            <Chip
              label={`Current: ${currentVersion}`}
              size="small"
              sx={{
                bgcolor: colors.accent.primary,
                color: colors.text.primary,
              }}
            />
            <Chip
              label={`Latest: ${latestVersion}`}
              size="small"
              sx={{
                bgcolor: colors.accent.secondary,
                color: colors.secondary,
                fontWeight: 600,
              }}
            />
          </Stack>
        </Stack>
      </DialogTitle>

      <DialogContent sx={{ mt: 2 }}>
        {showUpdateProgress && (
          <Box sx={{ mb: 3 }}>
            <Alert
              severity={
                updateStatus === "success"
                  ? "success"
                  : updateStatus === "error"
                  ? "error"
                  : "info"
              }
              icon={
                updateStatus === "success" ? (
                  <CheckCircleIcon />
                ) : updateStatus === "error" ? (
                  <CloseIcon />
                ) : (
                  <CloudDownloadIcon />
                )
              }
              sx={{
                bgcolor:
                  updateStatus === "success"
                    ? colors.accent.secondary
                    : updateStatus === "error"
                    ? colors.accent.primary
                    : colors.accent.tertiary,
                color: colors.text.primary,
              }}
            >
              {updateMessage}
            </Alert>
            {isUpdating && <LinearProgress sx={{ mt: 2 }} />}
          </Box>
        )}

        <Box
          sx={{
            maxHeight: 400,
            overflow: "auto",
            p: 2,
            bgcolor: colors.background.default,
            borderRadius: 1,
            border: `1px solid ${colors.border.default}`,
          }}
        >
          <Typography
            variant="subtitle1"
            sx={{
              color: colors.secondary,
              mb: 2,
              fontWeight: 600,
              textTransform: "uppercase",
            }}
          >
            Release Notes
          </Typography>
          <Box
            sx={{
              color: colors.text.primary,
              "& h1, & h2, & h3": {
                color: colors.secondary,
                mt: 2,
                mb: 1,
              },
              "& h1": { fontSize: "1.5rem" },
              "& h2": { fontSize: "1.25rem" },
              "& h3": { fontSize: "1.1rem" },
              "& p": {
                mb: 1,
                lineHeight: 1.6,
              },
              "& ul, & ol": {
                pl: 3,
                mb: 1,
              },
              "& li": {
                mb: 0.5,
              },
              "& code": {
                bgcolor: colors.background.paper,
                color: colors.secondary,
                px: 0.5,
                py: 0.25,
                borderRadius: 0.5,
                fontSize: "0.9em",
                fontFamily: "monospace",
              },
              "& pre": {
                bgcolor: colors.background.paper,
                p: 1.5,
                borderRadius: 1,
                overflow: "auto",
                border: `1px solid ${colors.border.default}`,
              },
              "& a": {
                color: colors.secondary,
                textDecoration: "none",
                "&:hover": {
                  textDecoration: "underline",
                },
              },
              "& blockquote": {
                borderLeft: `4px solid ${colors.secondary}`,
                pl: 2,
                ml: 0,
                fontStyle: "italic",
                color: colors.text.secondary,
              },
            }}
          >
            <ReactMarkdown
              components={{
                h2(props) {
                  const { node, ref, ...rest } = props;
                  return (
                    <Typography
                      variant="subtitle2"
                      sx={{
                        fontWeight: 800,
                        textTransform: "uppercase",
                      }}
                      {...rest}
                    />
                  );
                },
              }}
            >
              {releaseNotes}
            </ReactMarkdown>
          </Box>
        </Box>

        <Divider sx={{ my: 2, borderColor: colors.border.default }} />

        <Stack direction="row" spacing={2} justifyContent="center">
          <Button
            variant="outlined"
            startIcon={<DescriptionIcon />}
            href="https://github.com/DanielLavrushin/b4/blob/main/changelog.md"
            target="_blank"
            rel="noopener noreferrer"
            disabled={isUpdating}
            sx={{
              borderColor: colors.border.default,
              color: colors.text.primary,
              "&:hover": {
                borderColor: colors.secondary,
                bgcolor: colors.accent.secondaryHover,
              },
            }}
          >
            Read Full Changelog
          </Button>
          <Button
            variant="outlined"
            startIcon={<OpenInNewIcon />}
            href={releaseUrl}
            target="_blank"
            rel="noopener noreferrer"
            disabled={isUpdating}
            sx={{
              borderColor: colors.border.default,
              color: colors.text.primary,
              "&:hover": {
                borderColor: colors.secondary,
                bgcolor: colors.accent.secondaryHover,
              },
            }}
          >
            View on GitHub
          </Button>
        </Stack>
      </DialogContent>

      <DialogActions
        sx={{
          borderTop: `1px solid ${colors.border.default}`,
          p: 2,
        }}
      >
        <Button
          onClick={onDismiss}
          startIcon={<CloseIcon />}
          disabled={isUpdating}
          sx={{
            color: colors.text.secondary,
            "&:hover": {
              bgcolor: colors.accent.primaryHover,
            },
          }}
        >
          Don't Show Again for This Version
        </Button>
        <Box sx={{ flex: 1 }} />

        {updateStatus === "idle" && (
          <>
            <Button
              onClick={onClose}
              variant="outlined"
              disabled={isUpdating}
              sx={{
                borderColor: colors.border.default,
                color: colors.text.primary,
                "&:hover": {
                  borderColor: colors.secondary,
                  bgcolor: colors.accent.secondaryHover,
                },
              }}
            >
              Remind Me Later
            </Button>
            <Button
              onClick={handleUpdate}
              variant="contained"
              startIcon={<CloudDownloadIcon />}
              disabled={isUpdating}
              sx={{
                bgcolor: colors.primary,
                "&:hover": {
                  bgcolor: colors.secondary,
                },
              }}
            >
              Update Now
            </Button>
          </>
        )}

        {updateStatus === "success" && (
          <Button
            variant="contained"
            onClick={() => window.location.reload()}
            sx={{
              bgcolor: colors.secondary,
              "&:hover": {
                bgcolor: colors.primary,
              },
            }}
          >
            Reload Page
          </Button>
        )}
      </DialogActions>
    </Dialog>
  );
};
