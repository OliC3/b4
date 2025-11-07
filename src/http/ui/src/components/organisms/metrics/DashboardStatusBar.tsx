import React from "react";
import { Paper, Stack, Typography } from "@mui/material";
import { StatusBadge } from "@atoms/metrics/StatusBadge";
import { formatNumber } from "@utils";
import { colors } from "@design";

interface DashboardStatusBarProps {
  metrics: {
    nfqueue_status: string;
    tables_status: string;
    worker_status: Array<any>;
    tcp_connections: number;
    udp_connections: number;
  };
}

export const DashboardStatusBar: React.FC<DashboardStatusBarProps> = ({
  metrics,
}) => {
  return (
    <Paper
      sx={{
        p: 2,
        mb: 3,
        bgcolor: colors.background.paper,
        borderColor: colors.border.default,
      }}
      variant="outlined"
    >
      <Stack direction="row" spacing={2} alignItems="center" flexWrap="wrap">
        <Typography variant="subtitle2" sx={{ color: colors.text.secondary }}>
          System Status:
        </Typography>
        <StatusBadge
          label={`NFQueue: ${metrics.nfqueue_status}`}
          status="active"
        />
        <StatusBadge
          label={`firewall: ${metrics.tables_status}`}
          status="active"
        />
        <StatusBadge
          label={`${metrics.worker_status.length} threads`}
          status={metrics.worker_status.length > 0 ? "active" : "error"}
        />
        <StatusBadge
          label={`TCP: ${formatNumber(metrics.tcp_connections)}`}
          status="active"
        />
        <StatusBadge
          label={`UDP: ${formatNumber(metrics.udp_connections)}`}
          status="active"
        />
      </Stack>
    </Paper>
  );
};
