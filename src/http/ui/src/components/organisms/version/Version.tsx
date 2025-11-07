import React, { useState } from "react";
import { Box, Link, Stack, Divider } from "@mui/material";
import { colors } from "@design";
import GitHubIcon from "@mui/icons-material/GitHub";
import { VersionBadge } from "@/components/molecules/version/Badge";
import { UpdateModal } from "./UpdateDialog";
import { useGitHubRelease, dismissVersion } from "@hooks/useGitHubRelease";

export default function Version() {
  const [updateModalOpen, setUpdateModalOpen] = useState(false);
  const { latestRelease, isNewVersionAvailable, isLoading, currentVersion } =
    useGitHubRelease();

  const handleVersionClick = () => {
    if (isNewVersionAvailable && latestRelease) {
      setUpdateModalOpen(true);
    }
  };

  const handleDismissUpdate = () => {
    if (latestRelease) {
      dismissVersion(latestRelease.tag_name);
      setUpdateModalOpen(false);
    }
  };

  const handleCloseModal = () => {
    setUpdateModalOpen(false);
  };

  return (
    <>
      <Box
        sx={{
          py: 2,
        }}
      >
        <Divider sx={{ mb: 2, borderColor: colors.border.default }} />
        <Stack spacing={1.5} alignItems="center">
          <Link
            href="https://github.com/daniellavrushin/b4"
            target="_blank"
            rel="noopener noreferrer"
            sx={{
              display: "flex",
              alignItems: "center",
              gap: 0.5,
              color: colors.text.secondary,
              textDecoration: "none",
              transition: "color 0.2s ease",
              "&:hover": {
                color: colors.secondary,
              },
            }}
          >
            <GitHubIcon sx={{ fontSize: "1rem" }} />
            <span style={{ fontSize: "0.75rem" }}>DanielLavrushin/b4</span>
          </Link>
          <VersionBadge
            version={currentVersion}
            hasUpdate={isNewVersionAvailable}
            isLoading={isLoading}
            onClick={handleVersionClick}
          />
        </Stack>
      </Box>

      {latestRelease && (
        <UpdateModal
          open={updateModalOpen}
          onClose={handleCloseModal}
          onDismiss={handleDismissUpdate}
          currentVersion={currentVersion}
          latestVersion={latestRelease.tag_name}
          releaseNotes={latestRelease.body}
          releaseUrl={latestRelease.html_url}
          publishedAt={latestRelease.published_at}
        />
      )}
    </>
  );
}
