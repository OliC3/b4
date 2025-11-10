import React, { useState, useEffect } from "react";
import {
  Box,
  Button,
  Stack,
  Typography,
  LinearProgress,
  Alert,
  Paper,
  Divider,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
} from "@mui/material";
import {
  PlayArrow as StartIcon,
  Stop as StopIcon,
  Refresh as RefreshIcon,
} from "@mui/icons-material";
import { colors } from "@design";
import { useConfigLoad } from "@hooks/useConfig";

interface PresetSummary {
  average_speed: number;
  success_rate: number;
  fastest_domain: string;
  slowest_domain: string;
}

interface DiscoverySuite {
  id: string;
  status: "pending" | "running" | "complete" | "failed" | "canceled";
  start_time: string;
  end_time: string;
  total_checks: number;
  completed_checks: number;
  preset_results?: Record<string, PresetSummary>;
}

interface DomainPresetResult {
  preset_name: string;
  status: "complete" | "failed";
  duration: number;
  speed: number;
  bytes_read: number;
  error?: string;
  status_code: number;
}

interface DomainDiscoveryResult {
  domain: string;
  best_preset: string;
  best_speed: number;
  best_success: boolean;
  results: Record<string, DomainPresetResult>;
}

interface DiscoverySuite {
  id: string;
  status: "pending" | "running" | "complete" | "failed" | "canceled";
  start_time: string;
  end_time: string;
  total_checks: number;
  completed_checks: number;
  domain_discovery_results?: Record<string, DomainDiscoveryResult>;
}

export const DiscoveryRunner: React.FC = () => {
  const [running, setRunning] = useState(false);
  const [suiteId, setSuiteId] = useState<string | null>(null);
  const [suite, setSuite] = useState<DiscoverySuite | null>(null);
  const [error, setError] = useState<string | null>(null);
  const { config } = useConfigLoad();

  // Poll for discovery status
  useEffect(() => {
    if (!suiteId || !running) return;

    const fetchStatus = async () => {
      try {
        const response = await fetch(`/api/check/status?id=${suiteId}`);
        if (!response.ok) throw new Error("Failed to fetch discovery status");

        const data = (await response.json()) as DiscoverySuite;
        setSuite(data);

        if (["complete", "failed", "canceled"].includes(data.status)) {
          setRunning(false);
        }
      } catch (err) {
        console.error("Failed to fetch discovery status:", err);
        setError(err instanceof Error ? err.message : "Unknown error");
        setRunning(false);
      }
    };

    const interval = setInterval(() => {
      void fetchStatus();
    }, 2000); // Poll every 2 seconds for discovery (slower process)

    return () => clearInterval(interval);
  }, [suiteId, running]);

  const startDiscovery = async () => {
    setError(null);
    setRunning(true);
    setSuite(null);

    try {
      const timeout = (config?.system.checker.timeout || 15) * 1e9;
      const maxConcurrent = config?.system.checker.max_concurrent || 3;

      const response = await fetch("/api/check/discovery", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          timeout: timeout,
          max_concurrent: maxConcurrent,
        }),
      });

      if (!response.ok) {
        const text = await response.text();
        throw new Error(text || "Failed to start discovery");
      }

      const data = (await response.json()) as { id: string; message: string };
      setSuiteId(data.id);
    } catch (err) {
      console.error("Failed to start discovery:", err);
      setError(
        err instanceof Error ? err.message : "Failed to start discovery"
      );
      setRunning(false);
    }
  };

  const cancelDiscovery = async () => {
    if (!suiteId) return;

    try {
      await fetch(`/api/check/cancel?id=${suiteId}`, { method: "DELETE" });
      setRunning(false);
    } catch (err) {
      console.error("Failed to cancel discovery:", err);
    }
  };

  const resetDiscovery = () => {
    setSuiteId(null);
    setSuite(null);
    setError(null);
    setRunning(false);
  };

  const progress = suite
    ? (suite.completed_checks / suite.total_checks) * 100
    : 0;

  return (
    <Stack spacing={3}>
      {/* Control Panel */}
      <Paper
        elevation={0}
        sx={{
          p: 3,
          bgcolor: colors.background.paper,
          border: `1px solid ${colors.border.default}`,
          borderRadius: 2,
        }}
      >
        <Stack spacing={2}>
          <Box
            sx={{
              display: "flex",
              alignItems: "center",
              justifyContent: "space-between",
            }}
          >
            <Typography variant="h6" sx={{ color: colors.text.primary }}>
              Configuration Discovery
            </Typography>
            <Stack direction="row" spacing={1}>
              {!running && !suite && (
                <Button
                  variant="contained"
                  startIcon={<StartIcon />}
                  onClick={() => {
                    void startDiscovery();
                  }}
                  sx={{
                    bgcolor: colors.secondary,
                    "&:hover": { bgcolor: colors.primary },
                  }}
                >
                  Start Discovery
                </Button>
              )}
              {running && (
                <Button
                  variant="outlined"
                  startIcon={<StopIcon />}
                  onClick={() => {
                    void cancelDiscovery();
                  }}
                  sx={{
                    borderColor: colors.quaternary,
                    color: colors.quaternary,
                  }}
                >
                  Cancel
                </Button>
              )}
              {suite && !running && (
                <Button
                  variant="outlined"
                  startIcon={<RefreshIcon />}
                  onClick={resetDiscovery}
                  sx={{
                    borderColor: colors.secondary,
                    color: colors.secondary,
                  }}
                >
                  New Discovery
                </Button>
              )}
            </Stack>
          </Box>

          <Alert severity="warning" sx={{ bgcolor: colors.accent.tertiary }}>
            <strong>Warning:</strong> Discovery mode will temporarily apply
            different configurations to test effectiveness. This may briefly
            affect your service traffic during testing.
          </Alert>

          {error && <Alert severity="error">{error}</Alert>}

          {running && suite && (
            <Box>
              <Box
                sx={{
                  display: "flex",
                  justifyContent: "space-between",
                  mb: 1,
                }}
              >
                <Typography variant="body2" color="text.secondary">
                  Testing configurations: {suite.completed_checks} of{" "}
                  {suite.total_checks} checks completed
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  {progress.toFixed(0)}%
                </Typography>
              </Box>
              <LinearProgress
                variant="determinate"
                value={progress}
                sx={{
                  height: 8,
                  borderRadius: 4,
                  bgcolor: colors.background.dark,
                  "& .MuiLinearProgress-bar": {
                    bgcolor: colors.secondary,
                    borderRadius: 4,
                  },
                }}
              />
            </Box>
          )}
        </Stack>
      </Paper>

      {/* Results Table */}
      {suite?.domain_discovery_results &&
        Object.keys(suite.domain_discovery_results).length > 0 && (
          <Paper
            elevation={0}
            sx={{
              bgcolor: colors.background.paper,
              border: `1px solid ${colors.border.default}`,
              borderRadius: 2,
              overflow: "hidden",
            }}
          >
            <Box sx={{ p: 3 }}>
              <Typography
                variant="h6"
                sx={{ mb: 2, color: colors.text.primary }}
              >
                Best Configuration Per Domain
              </Typography>
              <Divider sx={{ mb: 2, borderColor: colors.border.default }} />
            </Box>

            <TableContainer>
              <Table>
                <TableHead>
                  <TableRow sx={{ bgcolor: colors.background.dark }}>
                    <TableCell
                      sx={{ color: colors.text.primary, fontWeight: 600 }}
                    >
                      Domain
                    </TableCell>
                    <TableCell
                      sx={{ color: colors.text.primary, fontWeight: 600 }}
                    >
                      Best Configuration
                    </TableCell>
                    <TableCell
                      align="right"
                      sx={{ color: colors.text.primary, fontWeight: 600 }}
                    >
                      Speed
                    </TableCell>
                    <TableCell
                      sx={{ color: colors.text.primary, fontWeight: 600 }}
                    >
                      Status
                    </TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {Object.values(suite.domain_discovery_results)
                    .sort((a, b) => b.best_speed - a.best_speed)
                    .map((domainResult) => (
                      <TableRow
                        key={domainResult.domain}
                        sx={{
                          "&:hover": { bgcolor: colors.accent.primaryStrong },
                        }}
                      >
                        <TableCell sx={{ color: colors.text.primary }}>
                          {domainResult.domain}
                        </TableCell>
                        <TableCell sx={{ color: colors.text.secondary }}>
                          {domainResult.best_success ? (
                            <Chip
                              label={domainResult.best_preset}
                              size="small"
                              sx={{
                                bgcolor: colors.secondary,
                                color: colors.text.primary,
                              }}
                            />
                          ) : (
                            <Typography color="error" variant="body2">
                              No successful config
                            </Typography>
                          )}
                        </TableCell>
                        <TableCell
                          align="right"
                          sx={{ color: colors.text.secondary }}
                        >
                          {domainResult.best_success
                            ? `${(
                                domainResult.best_speed /
                                1024 /
                                1024
                              ).toFixed(2)} MB/s`
                            : "-"}
                        </TableCell>
                        <TableCell>
                          {domainResult.best_success ? (
                            <Chip
                              label="Success"
                              size="small"
                              sx={{
                                bgcolor: colors.secondary,
                                color: colors.text.primary,
                              }}
                            />
                          ) : (
                            <Chip
                              label="Failed"
                              size="small"
                              sx={{
                                bgcolor: colors.quaternary,
                                color: colors.text.primary,
                              }}
                            />
                          )}
                        </TableCell>
                      </TableRow>
                    ))}
                </TableBody>
              </Table>
            </TableContainer>
          </Paper>
        )}
    </Stack>
  );
};
