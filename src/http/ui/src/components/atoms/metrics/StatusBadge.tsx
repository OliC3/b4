// src/http/ui/src/components/atoms/StatusBadge.tsx
import React from "react";
import { Chip } from "@mui/material";
import {
  CheckCircle as CheckCircleIcon,
  Warning as WarningIcon,
  Error as ErrorIcon,
} from "@mui/icons-material";
import { colors } from "../../../Theme";

interface StatusBadgeProps {
  label: string;
  status: "active" | "inactive" | "warning" | "error";
}

export const StatusBadge: React.FC<StatusBadgeProps> = ({ label, status }) => {
  const statusConfig = {
    active: {
      color: "#4caf50",
      icon: <CheckCircleIcon sx={{ fontSize: 16 }} />,
    },
    inactive: {
      color: colors.text.secondary,
      icon: <ErrorIcon sx={{ fontSize: 16 }} />,
    },
    warning: { color: "#ff9800", icon: <WarningIcon sx={{ fontSize: 16 }} /> },
    error: { color: "#f44336", icon: <ErrorIcon sx={{ fontSize: 16 }} /> },
  };

  const config = statusConfig[status];

  return (
    <Chip
      label={label}
      icon={config.icon}
      size="small"
      sx={{
        bgcolor: config.color + "22",
        color: config.color,
        borderColor: config.color + "44",
        fontWeight: 600,
      }}
      variant="outlined"
    />
  );
};
