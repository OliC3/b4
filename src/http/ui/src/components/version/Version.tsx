import { useState } from "react";
import { Box, Link, Stack, Divider } from "@mui/material";
import { colors } from "@design";
import { VersionBadge } from "./Badge";
import { UpdateModal } from "./UpdateDialog";
import { useGitHubRelease, dismissVersion } from "@hooks/useGitHubRelease";
import { GitHubIcon } from "@b4.icons";

export default function Version() {
  const [updateModalOpen, setUpdateModalOpen] = useState(false);
  const {
    releases,
    latestRelease,
    isNewVersionAvailable,
    isLoading,
    currentVersion,
    includePrerelease,
    setIncludePrerelease,
  } = useGitHubRelease();

  const handleVersionClick = () => {
    setUpdateModalOpen(true);
  };

  const handleDismissUpdate = () => {
    if (latestRelease) {
      dismissVersion(latestRelease.tag_name);
    }
    setUpdateModalOpen(false);
  };

  return (
    <>
      <Box sx={{ py: 2 }}>
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
              "&:hover": { color: colors.secondary },
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

      <UpdateModal
        open={updateModalOpen}
        onClose={() => setUpdateModalOpen(false)}
        onDismiss={handleDismissUpdate}
        currentVersion={currentVersion}
        releases={releases}
        includePrerelease={includePrerelease}
        onTogglePrerelease={setIncludePrerelease}
      />
    </>
  );
}
