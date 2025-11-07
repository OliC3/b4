import React from "react";
import { Chip } from "@mui/material";
import TcpIcon from "@mui/icons-material/SyncAlt";
import UdpIcon from "@mui/icons-material/TrendingFlat";
import { colors } from "@design";

interface ProtocolChipProps {
  protocol: "TCP" | "UDP";
}

export const ProtocolChip: React.FC<ProtocolChipProps> = ({ protocol }) => {
  return (
    <Chip
      label={protocol}
      size="small"
      icon={
        protocol === "TCP" ? (
          <TcpIcon color="primary" />
        ) : (
          <UdpIcon color="secondary" />
        )
      }
      sx={{
        bgcolor: colors.accent.primary,
        color: protocol === "TCP" ? colors.primary : colors.secondary,
        fontWeight: 600,
      }}
    />
  );
};
