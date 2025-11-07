import React from "react";
import { Grid } from "@mui/material";
import {
  Speed as SpeedIcon,
  Storage as StorageIcon,
  SwapHoriz as SwapHorizIcon,
  Memory as MemoryIcon,
} from "@mui/icons-material";
import { StatCard } from "@molecules/metrics/StatCard";
import { formatBytes, formatNumber } from "@utils";
import { colors } from "@design";

interface DashboardMetricsGridProps {
  metrics: {
    total_connections: number;
    active_flows: number;
    packets_processed: number;
    bytes_processed: number;
    targeted_connections: number;
    current_cps: number;
    current_pps: number;
    memory_usage: {
      percent: number;
    };
  };
}

export const DashboardMetricsGrid: React.FC<DashboardMetricsGridProps> = ({
  metrics,
}) => {
  return (
    <Grid container spacing={3}>
      <Grid size={{ xs: 12, sm: 6, md: 3 }}>
        <StatCard
          title="Total Connections"
          value={formatNumber(metrics.total_connections)}
          subtitle={`${metrics.targeted_connections} targeted`}
          icon={<SwapHorizIcon />}
          color={colors.primary}
          variant="outlined"
        />
      </Grid>

      <Grid size={{ xs: 12, sm: 6, md: 3 }}>
        <StatCard
          title="Active Flows"
          value={formatNumber(metrics.active_flows)}
          subtitle={`${metrics.current_cps.toFixed(1)} conn/s`}
          icon={<SpeedIcon />}
          color={colors.secondary}
          variant="outlined"
        />
      </Grid>

      <Grid size={{ xs: 12, sm: 6, md: 3 }}>
        <StatCard
          title="Packets Processed"
          value={formatNumber(metrics.packets_processed)}
          subtitle={`${metrics.current_pps.toFixed(1)} pkt/s`}
          icon={<StorageIcon />}
          color={colors.tertiary}
          variant="outlined"
        />
      </Grid>

      <Grid size={{ xs: 12, sm: 6, md: 3 }}>
        <StatCard
          title="Data Processed"
          value={formatBytes(metrics.bytes_processed)}
          subtitle={`Memory: ${metrics.memory_usage.percent.toFixed(1)}%`}
          icon={<MemoryIcon />}
          color={colors.quaternary}
          variant="outlined"
        />
      </Grid>
    </Grid>
  );
};
