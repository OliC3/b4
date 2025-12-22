import { useState, useEffect, forwardRef } from "react";
import {
  Button,
  Typography,
  Box,
  Divider,
  Stack,
  LinearProgress,
  Chip,
  FormControlLabel,
  Switch,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
} from "@mui/material";

import {
  NewReleaseIcon,
  DescriptionIcon,
  OpenInNewIcon,
  CheckCircleIcon,
  CloseIcon,
  CloudDownloadIcon,
} from "@b4.icons";
import { B4Alert } from "@b4.elements";
import ReactMarkdown from "react-markdown";
import { useSystemUpdate } from "@hooks/useSystemUpdate";
import { colors } from "@design";
import { B4Dialog } from "@common/B4Dialog";
import { GitHubRelease, compareVersions } from "@hooks/useGitHubRelease";
import React from "react";

interface UpdateModalProps {
  open: boolean;
  onClose: () => void;
  onDismiss: () => void;
  currentVersion: string;
  releases: GitHubRelease[];
  includePrerelease: boolean;
  onTogglePrerelease: (include: boolean) => void;
}

const H2Typography = forwardRef<
  HTMLHeadingElement,
  React.ComponentProps<typeof Typography>
>(function H2Typography(props, ref) {
  return (
    <Typography
      component="h2"
      variant="subtitle2"
      sx={{ fontWeight: 800, textTransform: "uppercase" }}
      ref={ref}
      {...props}
    />
  );
});

export const UpdateModal = ({
  open,
  onClose,
  onDismiss,
  currentVersion,
  releases,
  includePrerelease,
  onTogglePrerelease,
}: UpdateModalProps) => {
  const { performUpdate, waitForReconnection } = useSystemUpdate();
  const [updateStatus, setUpdateStatus] = useState<
    "idle" | "updating" | "reconnecting" | "success" | "error"
  >("idle");
  const [updateMessage, setUpdateMessage] = useState("");
  const [selectedVersion, setSelectedVersion] = useState<string>("");

  useEffect(() => {
    if (releases.length > 0 && !selectedVersion) {
      setSelectedVersion(releases[0].tag_name);
    }
  }, [releases, selectedVersion]);

  useEffect(() => {
    if (!open) {
      setUpdateStatus("idle");
      setUpdateMessage("");
    }
  }, [open]);

  const selectedRelease =
    releases.find((r) => r.tag_name === selectedVersion) || releases[0];

  const isDowngrade =
    selectedVersion &&
    compareVersions(`v${currentVersion}`, selectedVersion) > 0;
  const isCurrent = selectedVersion === `v${currentVersion}`;

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString("en-US", {
      year: "numeric",
      month: "long",
      day: "numeric",
    });
  };

  const handleUpdate = async () => {
    setUpdateStatus("updating");
    setUpdateMessage("Initiating update...");

    const result = await performUpdate(selectedVersion);
    if (!result || !result.success) {
      setUpdateStatus("error");
      setUpdateMessage(result?.message || "Failed to initiate update.");
      return;
    }

    setUpdateMessage("Update in progress. Waiting for service to restart...");
    setUpdateStatus("reconnecting");

    const reconnected = await waitForReconnection();

    if (reconnected) {
      setUpdateStatus("success");
      setUpdateMessage("Update completed successfully! Refreshing...");
      setTimeout(() => globalThis.window.location.reload(), 5000);
    } else {
      setUpdateStatus("error");
      setUpdateMessage(
        "Update may have completed but service did not restart. Please check manually."
      );
    }
  };

  const isUpdating =
    updateStatus === "updating" || updateStatus === "reconnecting";

  const getDialogProps = () => {
    const base = {
      title: "Version Management",
      subtitle: selectedRelease
        ? `Published on ${formatDate(selectedRelease.published_at)}`
        : "",
      icon: <NewReleaseIcon />,
    };
    switch (updateStatus) {
      case "updating":
      case "reconnecting":
        return {
          ...base,
          title: "Updating B4 Service",
          subtitle: "Please wait...",
        };
      case "success":
        return { ...base, title: "Update Successful", subtitle: "" };
      case "error":
        return { ...base, title: "Update Failed", subtitle: "" };
      default:
        return base;
    }
  };

  const getStatusContent = () => {
    switch (updateStatus) {
      case "updating":
      case "reconnecting":
        return (
          <Box sx={{ mb: 3 }}>
            <Typography sx={{ mb: 1, color: colors.text.secondary }}>
              {updateMessage}
            </Typography>
            <LinearProgress />
          </Box>
        );
      case "success":
        return (
          <B4Alert severity="success" icon={<CheckCircleIcon />} sx={{ mb: 2 }}>
            {updateMessage}
          </B4Alert>
        );
      case "error":
        return (
          <B4Alert severity="error" sx={{ mb: 2 }}>
            {updateMessage}
          </B4Alert>
        );
      default:
        return null;
    }
  };

  const dialogContent = () => (
    <>
      {getStatusContent()}

      {updateStatus === "idle" && (
        <Box sx={{ mb: 3 }}>
          <Stack
            direction="row"
            spacing={2}
            alignItems="center"
            sx={{ mb: 2, mt: 2 }}
          >
            <FormControl size="small" sx={{ minWidth: 220 }}>
              <InputLabel>Select Version</InputLabel>
              <Select
                value={selectedVersion}
                label="Select Version"
                onChange={(e) => setSelectedVersion(e.target.value)}
              >
                {releases.map((r) => (
                  <MenuItem key={r.tag_name} value={r.tag_name}>
                    {r.tag_name}
                    {r.prerelease && " (pre-release)"}
                    {r.tag_name === `v${currentVersion}` && " (current)"}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
            <FormControlLabel
              control={
                <Switch
                  checked={includePrerelease}
                  onChange={(e) => onTogglePrerelease(e.target.checked)}
                  size="small"
                />
              }
              label="Include pre-releases"
            />
          </Stack>
          <Stack direction="row" spacing={1}>
            <Chip
              label={`Current: v${currentVersion}`}
              size="small"
              sx={{
                bgcolor: colors.accent.primary,
                color: colors.text.primary,
              }}
            />
            {!isCurrent && (
              <Chip
                label={isDowngrade ? "Downgrade" : "Upgrade"}
                size="small"
                color={isDowngrade ? "warning" : "success"}
                sx={{ fontWeight: 600 }}
              />
            )}
            {selectedRelease?.prerelease && (
              <Chip label="Pre-release" size="small" color="info" />
            )}
          </Stack>
        </Box>
      )}

      {selectedRelease && (
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
            Release Notes - {selectedRelease.tag_name}
          </Typography>
          <Box
            sx={{
              color: colors.text.primary,
              "& h1, & h2, & h3": { color: colors.secondary, mt: 2, mb: 1 },
              "& p": { mb: 1, lineHeight: 1.6 },
              "& ul, & ol": { pl: 3, mb: 1 },
              "& code": {
                bgcolor: colors.background.paper,
                color: colors.secondary,
                px: 0.5,
                py: 0.25,
                borderRadius: 0.5,
                fontSize: "0.9em",
              },
              "& a": { color: colors.secondary },
            }}
          >
            <ReactMarkdown components={{ h2: H2Typography }}>
              {selectedRelease.body || "No release notes available."}
            </ReactMarkdown>
          </Box>
        </Box>
      )}

      <Divider sx={{ my: 2, borderColor: colors.border.default }} />

      <Stack direction="row" spacing={2} justifyContent="center">
        <Button
          variant="outlined"
          startIcon={<DescriptionIcon />}
          href="https://github.com/DanielLavrushin/b4/blob/main/changelog.md"
          target="_blank"
          disabled={isUpdating}
        >
          Full Changelog
        </Button>
        {selectedRelease && (
          <Button
            variant="outlined"
            startIcon={<OpenInNewIcon />}
            href={selectedRelease.html_url}
            target="_blank"
            disabled={isUpdating}
          >
            View on GitHub
          </Button>
        )}
      </Stack>
    </>
  );

  const dialogActions = () => (
    <>
      <Button
        onClick={onDismiss}
        startIcon={<CloseIcon />}
        disabled={isUpdating}
      >
        Don't Show Again
      </Button>
      <Box sx={{ flex: 1 }} />
      {updateStatus === "idle" && (
        <>
          <Button onClick={onClose} variant="outlined" disabled={isUpdating}>
            Close
          </Button>
          <Button
            onClick={() => void handleUpdate()}
            variant="contained"
            startIcon={<CloudDownloadIcon />}
            disabled={isUpdating || isCurrent}
            color={isDowngrade ? "warning" : "primary"}
          >
            {isDowngrade ? "Downgrade" : "Update"}
          </Button>
        </>
      )}
      {updateStatus === "success" && (
        <Button
          variant="contained"
          onClick={() => globalThis.window.location.reload()}
        >
          Reload Page
        </Button>
      )}
    </>
  );

  return (
    <B4Dialog
      {...getDialogProps()}
      open={open}
      onClose={isUpdating ? () => {} : onClose}
      actions={dialogActions()}
      maxWidth="lg"
    >
      {dialogContent()}
    </B4Dialog>
  );
};
