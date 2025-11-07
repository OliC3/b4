import { Box, Chip, CircularProgress, Tooltip } from "@mui/material";
import { NewReleases as NewReleasesIcon } from "@mui/icons-material";
import { colors } from "@design";

interface VersionBadgeProps {
  version: string;
  hasUpdate?: boolean;
  isLoading?: boolean;
  onClick?: () => void;
}

export const VersionBadge: React.FC<VersionBadgeProps> = ({
  version,
  hasUpdate = false,
  isLoading = false,
  onClick,
}) => {
  if (isLoading) {
    return (
      <Box sx={{ display: "flex", alignItems: "center", gap: 1, px: 2 }}>
        <CircularProgress size={12} sx={{ color: colors.secondary }} />
        <span style={{ color: colors.text.secondary, fontSize: "0.75rem" }}>
          Checking for updates...
        </span>
      </Box>
    );
  }

  return (
    <Box
      sx={{
        display: "flex",
        alignItems: "center",
        gap: 1,
        px: 2,
        cursor: hasUpdate ? "pointer" : "default",
      }}
      onClick={hasUpdate ? onClick : undefined}
    >
      {hasUpdate ? (
        <Tooltip title="New version available! Click to view details">
          <Chip
            label={`v${version}`}
            size="small"
            icon={<NewReleasesIcon />}
            sx={{
              bgcolor: colors.accent.secondary,
              color: colors.secondary,
              fontWeight: 600,
              animation: "pulse 2s ease-in-out infinite",
              "@keyframes pulse": {
                "0%, 100%": {
                  opacity: 1,
                },
                "50%": {
                  opacity: 0.7,
                },
              },
              "& .MuiChip-icon": {
                color: colors.secondary,
              },
              "&:hover": {
                bgcolor: colors.accent.secondaryHover,
                transform: "scale(1.05)",
              },
              transition: "all 0.2s ease",
            }}
          />
        </Tooltip>
      ) : (
        <span style={{ color: colors.secondary, fontSize: "0.75rem" }}>
          v{version}
        </span>
      )}
    </Box>
  );
};
