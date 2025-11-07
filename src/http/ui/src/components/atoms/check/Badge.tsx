import React from "react";
import { Chip } from "@mui/material";
import {
  HourglassEmpty as PendingIcon,
  PlayArrow as RunningIcon,
  CheckCircle as CompleteIcon,
  Error as ErrorIcon,
  Cancel as CanceledIcon,
} from "@mui/icons-material";
import { colors } from "@design";

export type TestStatus =
  | "pending"
  | "running"
  | "complete"
  | "failed"
  | "canceled";

interface TestStatusBadgeProps {
  status: TestStatus;
  size?: "small" | "medium";
}

const statusConfig: Record<
  TestStatus,
  { label: string; color: string; icon: React.ReactNode }
> = {
  pending: {
    label: "Pending",
    color: colors.text.secondary,
    icon: <PendingIcon fontSize="small" />,
  },
  running: {
    label: "Running",
    color: colors.secondary,
    icon: <RunningIcon fontSize="small" />,
  },
  complete: {
    label: "Complete",
    color: colors.secondary,
    icon: <CompleteIcon fontSize="small" />,
  },
  failed: {
    label: "Failed",
    color: colors.quaternary,
    icon: <ErrorIcon fontSize="small" />,
  },
  canceled: {
    label: "Canceled",
    color: colors.text.secondary,
    icon: <CanceledIcon fontSize="small" />,
  },
};

export const TestStatusBadge: React.FC<TestStatusBadgeProps> = ({
  status,
  size = "small",
}) => {
  const config = statusConfig[status];

  return (
    <Chip
      label={config.label}
      size={size}
      sx={{
        bgcolor: `${config.color}22`,
        color: config.color,
        borderColor: config.color,
        border: 1,
      }}
    />
  );
};
