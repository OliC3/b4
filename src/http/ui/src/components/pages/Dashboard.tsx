// src/http/ui/src/components/pages/Dashboard.tsx
import React, { useEffect, useState, useRef } from "react";
import {
  Box,
  Container,
  Typography,
  Stack,
  Chip,
  LinearProgress,
} from "@mui/material";
import {
  CheckCircle as CheckCircleIcon,
  Error as ErrorIcon,
} from "@mui/icons-material";
import { DashboardMetricsGrid } from "../organisms/metrics/DashboardMetricsGrid";
import { DashboardStatusBar } from "../organisms/metrics/DashboardStatusBar";
import { DashboardCharts } from "../organisms/metrics/DashboardCharts";
import { DashboardActivityPanels } from "../organisms/metrics/DashboardActivityPanels";
import { colors } from "../../Theme";

export interface Metrics {
  total_connections: number;
  active_flows: number;
  packets_processed: number;
  bytes_processed: number;
  tcp_connections: number;
  udp_connections: number;
  targeted_connections: number;
  connection_rate: Array<{ timestamp: number; value: number }>;
  packet_rate: Array<{ timestamp: number; value: number }>;
  top_domains: Record<string, number>;
  protocol_dist: Record<string, number>;
  geo_dist: Record<string, number>;
  start_time: string;
  uptime: string;
  cpu_usage: number;
  memory_usage: {
    allocated: number;
    total_allocated: number;
    system: number;
    num_gc: number;
    heap_alloc: number;
    heap_inuse: number;
    percent: number;
  };
  worker_status: Array<{
    id: number;
    status: string;
    processed: number;
  }>;
  nfqueue_status: string;
  iptables_status: string;
  recent_connections: Array<{
    timestamp: string;
    protocol: string;
    domain: string;
    source: string;
    destination: string;
    is_target: boolean;
  }>;
  recent_events: Array<{
    timestamp: string;
    level: string;
    message: string;
  }>;
  current_cps: number;
  current_pps: number;
}

export default function Dashboard() {
  const [metrics, setMetrics] = useState<Metrics | null>(null);
  const [connected, setConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    // Connect to WebSocket for real-time metrics
    const connectWebSocket = () => {
      const ws = new WebSocket(
        (location.protocol === "https:" ? "wss://" : "ws://") +
          location.host +
          "/api/ws/metrics"
      );

      ws.onopen = () => {
        setConnected(true);
        console.log("Metrics WebSocket connected");
      };

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          setMetrics(data);
        } catch (error) {
          console.error("Failed to parse metrics:", error);
        }
      };

      ws.onerror = (error) => {
        console.error("Metrics WebSocket error:", error);
        setConnected(false);
      };

      ws.onclose = () => {
        setConnected(false);
        console.log("Metrics WebSocket disconnected, reconnecting...");
        setTimeout(connectWebSocket, 3000);
      };

      wsRef.current = ws;
    };

    connectWebSocket();

    // Cleanup on unmount
    return () => {
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, []);

  if (!metrics) {
    return (
      <Container maxWidth={false} sx={{ py: 3 }}>
        <Box sx={{ textAlign: "center", py: 8 }}>
          <LinearProgress sx={{ mb: 2 }} />
          <Typography>Loading dashboard metrics...</Typography>
        </Box>
      </Container>
    );
  }

  return (
    <Container maxWidth={false} sx={{ py: 3, px: 3 }}>
      {/* Header */}
      <Box
        sx={{
          mb: 3,
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
        }}
      >
        <Stack direction="row" spacing={2} alignItems="center">
          <Chip
            label={connected ? "Connected" : "Disconnected"}
            color={connected ? "success" : "error"}
            size="small"
            icon={connected ? <CheckCircleIcon /> : <ErrorIcon />}
          />
          <Typography variant="caption" sx={{ color: colors.text.secondary }}>
            Uptime: {metrics.uptime}
          </Typography>
        </Stack>
      </Box>

      {/* Key Metrics Cards */}
      <Box sx={{ mb: 3 }}>
        <DashboardMetricsGrid metrics={metrics} />
      </Box>

      {/* Status Bar */}
      <DashboardStatusBar metrics={metrics} />

      {/* Charts Row */}
      <Box sx={{ mb: 3 }}>
        <DashboardCharts
          connectionRate={metrics.connection_rate}
          protocolDist={metrics.protocol_dist}
        />
      </Box>

      {/* Activity Panels */}
      <DashboardActivityPanels
        topDomains={metrics.top_domains}
        recentConnections={metrics.recent_connections}
      />
    </Container>
  );
}
