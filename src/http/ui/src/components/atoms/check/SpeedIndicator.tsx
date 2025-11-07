import React from "react";
import { Box, Typography } from "@mui/material";
import {
  Speed as SpeedIcon,
  // TrendingUp as UpIcon,
  // TrendingDown as DownIcon,
} from "@mui/icons-material";
import { colors } from "@design";

interface SpeedIndicatorProps {
  speed: number; // bytes per second
  improvement?: number; // percentage
  compact?: boolean;
}

const formatSpeed = (bytesPerSecond: number): string => {
  if (bytesPerSecond < 1024) return `${bytesPerSecond.toFixed(0)} B/s`;
  if (bytesPerSecond < 1024 * 1024)
    return `${(bytesPerSecond / 1024).toFixed(1)} KB/s`;
  return `${(bytesPerSecond / (1024 * 1024)).toFixed(2)} MB/s`;
};

export const SpeedIndicator: React.FC<SpeedIndicatorProps> = ({
  speed,
  improvement,
  compact = false,
}) => {
  const speedText = formatSpeed(speed);

  if (compact) {
    return (
      <Typography
        variant="body2"
        sx={{
          display: "flex",
          alignItems: "center",
          gap: 0.5,
          color: colors.text.primary,
        }}
      >
        <SpeedIcon fontSize="small" />
        {speedText}
        {improvement !== undefined && improvement > 0 && (
          <Typography
            component="span"
            variant="caption"
            sx={{ color: colors.secondary, ml: 0.5 }}
          >
            (+{improvement.toFixed(1)}%)
          </Typography>
        )}
      </Typography>
    );
  }

  return (
    <Box
      sx={{
        display: "flex",
        alignItems: "center",
        gap: 2,
        p: 1.5,
        borderRadius: 2,
        bgcolor: colors.accent.primary,
      }}
    >
      <SpeedIcon sx={{ color: colors.secondary }} />
      <Box>
        <Typography variant="h6" sx={{ color: colors.text.primary }}>
          {speedText}
        </Typography>
        {improvement !== undefined && (
          <Typography
            variant="caption"
            sx={{
              display: "flex",
              alignItems: "center",
              gap: 0.5,
              color: improvement > 0 ? colors.secondary : colors.text.secondary,
            }}
          >
            {/* {improvement > 0 ? (
              <UpIcon fontSize="small" />
            ) : (
              <DownIcon fontSize="small" />
            )}
            {improvement > 0 ? "+" : ""}
            {improvement.toFixed(1)}% vs baseline */}
          </Typography>
        )}
      </Box>
    </Box>
  );
};
