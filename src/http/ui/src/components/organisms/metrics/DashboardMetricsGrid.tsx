import React from "react";
import { Grid } from "@mui/material";
import {
  Speed as SpeedIcon,
  Storage as StorageIcon,
  SwapHoriz as SwapHorizIcon,
  Memory as MemoryIcon,
} from "@mui/icons-material";
import { MetricCard } from "../../atoms/metrics/MetricCard";
import { formatBytes, formatNumber } from "../../../utils";
import { colors } from "../../../Theme";

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
        <MetricCard
          title="Total Connections"
          value={formatNumber(metrics.total_connections)}
          subtitle={`${metrics.targeted_connections} targeted`}
          icon={<SwapHorizIcon />}
          color={colors.primary}
        />
      </Grid>
      <Grid size={{ xs: 12, sm: 6, md: 3 }}>
        <MetricCard
          title="Active Flows"
          value={formatNumber(metrics.active_flows)}
          subtitle={`${metrics.current_cps.toFixed(1)} conn/s`}
          icon={<SpeedIcon />}
          color={colors.secondary}
        />
      </Grid>
      <Grid size={{ xs: 12, sm: 6, md: 3 }}>
        <MetricCard
          title="Packets Processed"
          value={formatNumber(metrics.packets_processed)}
          subtitle={`${metrics.current_pps.toFixed(1)} pkt/s`}
          icon={<StorageIcon />}
          color={colors.tertiary}
        />
      </Grid>
      <Grid size={{ xs: 12, sm: 6, md: 3 }}>
        <MetricCard
          title="Data Processed"
          value={formatBytes(metrics.bytes_processed)}
          subtitle={`Memory: ${metrics.memory_usage.percent.toFixed(1)}%`}
          icon={<MemoryIcon />}
          color={colors.quaternary}
        />
      </Grid>
    </Grid>
  );
};
