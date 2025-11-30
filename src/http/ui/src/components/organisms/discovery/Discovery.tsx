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
  Chip,
  IconButton,
  Tooltip,
  Snackbar,
  CircularProgress,
  Collapse,
} from "@mui/material";
import {
  PlayArrow as StartIcon,
  Stop as StopIcon,
  Refresh as RefreshIcon,
  Add as AddIcon,
  Speed as SpeedIcon,
  ExpandMore as ExpandIcon,
  ExpandLess as CollapseIcon,
  TrendingUp as ImprovementIcon,
  Science as ScienceIcon,
} from "@mui/icons-material";
import { colors, button_yellow_outline } from "@design";
import { B4SetConfig } from "@/models/Config";
import SettingTextField from "@atoms/common/B4TextField";
import { DiscoveryAddDialog } from "./AddDialog";
import { B4Section } from "@molecules/common/B4Section";

// Strategy family types matching backend
type StrategyFamily =
  | "none"
  | "tcp_frag"
  | "tls_record"
  | "oob"
  | "ip_frag"
  | "fake_sni"
  | "sack"
  | "syn_fake";

type DiscoveryPhase =
  | "baseline"
  | "strategy_detection"
  | "optimization"
  | "combination";

interface DomainPresetResult {
  preset_name: string;
  family?: StrategyFamily;
  phase?: DiscoveryPhase;
  status: "complete" | "failed";
  duration: number;
  speed: number;
  bytes_read: number;
  error?: string;
  status_code: number;
  set?: B4SetConfig;
}

interface DiscoveryResult {
  domain: string;
  best_preset: string;
  best_speed: number;
  best_success: boolean;
  results: Record<string, DomainPresetResult>;
  baseline_speed?: number;
  improvement?: number;
}

interface DiscoverySuite {
  id: string;
  status: "pending" | "running" | "complete" | "failed" | "canceled";
  start_time: string;
  end_time: string;
  total_checks: number;
  completed_checks: number;
  current_phase?: DiscoveryPhase;
  domain_discovery_results?: Record<string, DiscoveryResult>;
}

interface DiscoveryResponse {
  id: string;
  estimated_tests: number;
  message: string;
}

// Friendly names for strategy families
const familyNames: Record<StrategyFamily, string> = {
  none: "Baseline",
  tcp_frag: "TCP Fragmentation",
  tls_record: "TLS Record Split",
  oob: "Out-of-Band",
  ip_frag: "IP Fragmentation",
  fake_sni: "Fake SNI",
  sack: "SACK Drop",
  syn_fake: "SYN Fake",
};

// Friendly names for phases
const phaseNames: Record<DiscoveryPhase, string> = {
  baseline: "Baseline Test",
  strategy_detection: "Strategy Detection",
  optimization: "Optimization",
  combination: "Combination Test",
};

export const DiscoveryRunner: React.FC = () => {
  const [running, setRunning] = useState(false);
  const [suiteId, setSuiteId] = useState<string | null>(null);
  const [suite, setSuite] = useState<DiscoverySuite | null>(null);

  const [error, setError] = useState<string | null>(null);
  const [expandedDomains, setExpandedDomains] = useState<Set<string>>(
    new Set()
  );
  const [snackbar, setSnackbar] = useState<{
    open: boolean;
    message: string;
    severity: "success" | "error";
  }>({ open: false, message: "", severity: "success" });

  const [domain, setDomain] = useState("");
  const [addingPreset, setAddingPreset] = useState(false);
  const [addDialog, setAddDialog] = useState<{
    open: boolean;
    domain: string;
    presetName: string;
    setConfig: B4SetConfig | null;
  }>({ open: false, domain: "", presetName: "", setConfig: null });

  const progress = suite
    ? (suite.completed_checks / suite.total_checks) * 100
    : 0;
  const isReconnecting = suiteId && running && !suite;

  const handleAddStrategy = (domain: string, result: DomainPresetResult) => {
    setAddDialog({
      open: true,
      domain,
      presetName: result.preset_name,
      setConfig: result.set || null,
    });
  };
  const toggleDomainExpand = (domain: string) => {
    setExpandedDomains((prev) => {
      const next = new Set(prev);
      if (next.has(domain)) {
        next.delete(domain);
      } else {
        next.add(domain);
      }
      return next;
    });
  };

  useEffect(() => {
    const savedSuiteId = localStorage.getItem("discovery_suiteId");
    if (savedSuiteId) {
      setSuiteId(savedSuiteId);
      setRunning(true); // Will trigger polling, which will update status
    }
  }, []);

  useEffect(() => {
    if (suiteId) {
      localStorage.setItem("discovery_suiteId", suiteId);
    }
  }, [suiteId]);

  // Poll for discovery status
  useEffect(() => {
    if (!suiteId || !running) return;

    const fetchStatus = async () => {
      try {
        const response = await fetch(`/api/discovery/status?id=${suiteId}`);
        if (!response.ok) throw new Error("Failed to fetch discovery status");

        const data = (await response.json()) as DiscoverySuite;
        setSuite(data);

        if (["complete", "failed", "canceled"].includes(data.status)) {
          setRunning(false);
          localStorage.removeItem("discovery_suiteId");
        }
      } catch (err) {
        console.error("Failed to fetch discovery status:", err);
        setError(err instanceof Error ? err.message : "Unknown error");
        setRunning(false);
      }
    };

    const interval = setInterval(() => {
      void fetchStatus();
    }, 1500); // Faster polling for better UX

    return () => clearInterval(interval);
  }, [suiteId, running]);

  const startDiscovery = async () => {
    if (!domain.trim()) {
      setError("Enter a domain to test");
      return;
    }

    setError(null);
    setRunning(true);
    setSuite(null);

    try {
      const response = await fetch("/api/discovery", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          domain: domain.trim(),
        }),
      });

      if (!response.ok) {
        const text = await response.text();
        throw new Error(text || "Failed to start discovery");
      }

      const data = (await response.json()) as DiscoveryResponse;
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
      await fetch(`/api/discovery/cancel?id=${suiteId}`, { method: "DELETE" });
      setRunning(false);
    } catch (err) {
      console.error("Failed to cancel discovery:", err);
    }
  };

  const resetDiscovery = () => {
    localStorage.removeItem("discovery_suiteId");
    setSuiteId(null);
    setSuite(null);
    setError(null);
    setRunning(false);
    setExpandedDomains(new Set());
  };

  const handleAddNew = async (name: string, domain: string) => {
    if (!addDialog.setConfig) return;
    setAddingPreset(true);

    try {
      const configToAdd = {
        ...addDialog.setConfig,
        name,
        targets: { ...addDialog.setConfig.targets, sni_domains: [domain] },
      };

      const response = await fetch("/api/discovery/add", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(configToAdd),
      });

      if (!response.ok) throw new Error("Failed to add configuration");

      setSnackbar({
        open: true,
        message: `Created set "${name}"`,
        severity: "success",
      });
      setAddDialog({
        open: false,
        domain: "",
        presetName: "",
        setConfig: null,
      });
    } catch {
      setSnackbar({
        open: true,
        message: "Failed to add configuration",
        severity: "error",
      });
    } finally {
      setAddingPreset(false);
    }
  };

  const handleAddToExisting = async (setId: string, domain: string) => {
    setAddingPreset(true);

    try {
      const response = await fetch(`/api/config/sets/${setId}/domains`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ domain }),
      });

      if (!response.ok) throw new Error("Failed to add domain");

      setSnackbar({
        open: true,
        message: `Added "${domain}" to existing set`,
        severity: "success",
      });
      setAddDialog({
        open: false,
        domain: "",
        presetName: "",
        setConfig: null,
      });
    } catch {
      setSnackbar({
        open: true,
        message: "Failed to add domain",
        severity: "error",
      });
    } finally {
      setAddingPreset(false);
    }
  };

  // Group results by phase for display
  const groupResultsByPhase = (results: Record<string, DomainPresetResult>) => {
    const grouped: Record<DiscoveryPhase, DomainPresetResult[]> = {
      baseline: [],
      strategy_detection: [],
      optimization: [],
      combination: [],
    };

    Object.values(results).forEach((result) => {
      const phase = result.phase || "strategy_detection";
      grouped[phase].push(result);
    });

    return grouped;
  };

  return (
    <Stack spacing={3}>
      {/* Control Panel */}
      <B4Section
        title="Configuration Discovery"
        description="Hierarchical testing: Strategy Detection → Optimization → Combination"
        icon={<ScienceIcon />}
      >
        {/* Header with actions */}
        <Box sx={{ display: "flex", gap: 2, alignItems: "flex-start" }}>
          <SettingTextField
            label="Domain to test"
            value={domain}
            onChange={(e) => setDomain(e.target.value)}
            placeholder="youtube.com"
            disabled={running || !!isReconnecting}
            helperText="Enter a domain to discover optimal bypass configuration"
          />
          {!running && !suite && (
            <Button
              startIcon={<StartIcon />}
              onClick={() => {
                void startDiscovery();
              }}
              disabled={!domain.trim()}
              sx={{
                px: 3,
                whiteSpace: "nowrap",
                ...button_yellow_outline,
              }}
            >
              Start Discovery
            </Button>
          )}
          {(running || isReconnecting) && (
            <Button
              variant="outlined"
              startIcon={<StopIcon />}
              onClick={() => {
                void cancelDiscovery();
              }}
              sx={{
                minWidth: 120,
                whiteSpace: "nowrap",
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
                minWidth: 150,
                whiteSpace: "nowrap",
                borderColor: colors.secondary,
                color: colors.secondary,
              }}
            >
              New Discovery
            </Button>
          )}
        </Box>
        {error && <Alert severity="error">{error}</Alert>}

        {isReconnecting && (
          <Box sx={{ display: "flex", alignItems: "center", gap: 2 }}>
            <CircularProgress size={20} sx={{ color: colors.secondary }} />
            <Typography variant="body2" sx={{ color: colors.text.secondary }}>
              Reconnecting to running discovery...
            </Typography>
          </Box>
        )}
        {/* Progress indicator */}
        {running && suite && (
          <Box>
            <Box
              sx={{
                display: "flex",
                justifyContent: "space-between",
                mb: 1,
              }}
            >
              <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                <Typography variant="body2" color="text.secondary">
                  {suite.current_phase && (
                    <Chip
                      label={phaseNames[suite.current_phase]}
                      size="small"
                      sx={{
                        mr: 1,
                        bgcolor: colors.accent.secondary,
                        color: colors.secondary,
                        fontWeight: 600,
                      }}
                    />
                  )}
                  {suite.completed_checks} of {suite.total_checks} checks
                </Typography>
              </Box>
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
      </B4Section>

      {/* Results */}
      {suite?.domain_discovery_results &&
        Object.keys(suite.domain_discovery_results).length > 0 && (
          <Stack spacing={2}>
            {Object.values(suite.domain_discovery_results)
              .sort((a, b) => b.best_speed - a.best_speed)
              .map((domainResult) => {
                const isExpanded = expandedDomains.has(domainResult.domain);
                const groupedResults = groupResultsByPhase(
                  domainResult.results
                );
                const successCount = Object.values(domainResult.results).filter(
                  (r) => r.status === "complete"
                ).length;
                const totalCount = Object.keys(domainResult.results).length;

                return (
                  <Paper
                    key={domainResult.domain}
                    elevation={0}
                    sx={{
                      bgcolor: colors.background.paper,
                      border: `1px solid ${colors.border.default}`,
                      borderRadius: 2,
                      overflow: "hidden",
                    }}
                  >
                    {/* Domain Header */}
                    <Box
                      sx={{
                        p: 2,
                        bgcolor: colors.accent.primary,
                        display: "flex",
                        alignItems: "center",
                        justifyContent: "space-between",
                        cursor: "pointer",
                      }}
                      onClick={() => toggleDomainExpand(domainResult.domain)}
                    >
                      <Box
                        sx={{ display: "flex", alignItems: "center", gap: 2 }}
                      >
                        <IconButton size="small">
                          {isExpanded ? <CollapseIcon /> : <ExpandIcon />}
                        </IconButton>
                        <Typography
                          variant="h6"
                          sx={{ color: colors.text.primary }}
                        >
                          {domainResult.domain}
                        </Typography>
                        {domainResult.best_success ? (
                          <Chip
                            label="Success"
                            size="small"
                            sx={{
                              bgcolor: colors.secondary,
                              color: colors.background.default,
                            }}
                          />
                        ) : running ? (
                          <Chip
                            label="Testing..."
                            size="small"
                            sx={{
                              bgcolor: colors.accent.primary,
                              color: colors.text.secondary,
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
                        <Chip
                          label={`${successCount}/${totalCount} configs`}
                          size="small"
                          variant="outlined"
                          sx={{ borderColor: colors.border.light }}
                        />
                        {domainResult.improvement &&
                          domainResult.improvement > 0 && (
                            <Chip
                              icon={<ImprovementIcon />}
                              label={`+${domainResult.improvement.toFixed(0)}%`}
                              size="small"
                              sx={{
                                bgcolor: colors.accent.secondary,
                                color: colors.secondary,
                                "& .MuiChip-icon": { color: colors.secondary },
                              }}
                            />
                          )}
                      </Box>
                      <Typography
                        variant="h6"
                        sx={{
                          color: domainResult.best_success
                            ? colors.secondary
                            : colors.text.secondary,
                          fontWeight: 600,
                        }}
                      >
                        {domainResult.best_success
                          ? `${(domainResult.best_speed / 1024 / 1024).toFixed(
                              2
                            )} MB/s`
                          : running
                          ? `${totalCount} tested...`
                          : "No working config"}
                      </Typography>
                    </Box>

                    {/* Best Configuration Quick View (always visible) */}
                    {(domainResult.best_success ||
                      (running &&
                        Object.values(domainResult.results).some(
                          (r) => r.status === "complete"
                        ))) && (
                      <Box>
                        <Box
                          sx={{
                            p: 2,
                            bgcolor: colors.background.default,
                            display: "flex",
                            alignItems: "center",
                            justifyContent: "space-between",
                            borderBottom: running
                              ? "none"
                              : `1px solid ${colors.border.default}`,
                          }}
                        >
                          <Box
                            sx={{
                              display: "flex",
                              alignItems: "center",
                              gap: 2,
                            }}
                          >
                            <SpeedIcon sx={{ color: colors.secondary }} />
                            <Box>
                              <Typography
                                variant="caption"
                                sx={{ color: colors.text.secondary }}
                              >
                                {running
                                  ? "Current Best"
                                  : "Best Configuration"}
                              </Typography>
                              <Typography
                                variant="body1"
                                sx={{
                                  color: colors.text.primary,
                                  fontWeight: 600,
                                }}
                              >
                                {domainResult.best_preset}
                                {domainResult.results[domainResult.best_preset]
                                  ?.family && (
                                  <Chip
                                    label={
                                      familyNames[
                                        domainResult.results[
                                          domainResult.best_preset
                                        ].family!
                                      ]
                                    }
                                    size="small"
                                    sx={{
                                      ml: 1,
                                      bgcolor: colors.accent.primary,
                                    }}
                                  />
                                )}
                              </Typography>
                            </Box>
                          </Box>
                          <Button
                            variant="contained"
                            startIcon={
                              addingPreset ? (
                                <CircularProgress size={18} color="inherit" />
                              ) : (
                                <AddIcon />
                              )
                            }
                            onClick={(e) => {
                              e.stopPropagation();
                              const bestResult =
                                domainResult.results[domainResult.best_preset];
                              void handleAddStrategy(
                                domainResult.domain,
                                bestResult
                              );
                            }}
                            disabled={addingPreset}
                            sx={{
                              bgcolor: colors.secondary,
                              color: colors.background.default,
                              "&:hover": { bgcolor: colors.primary },
                            }}
                          >
                            {running ? "Use Current Best" : "Use This Strategy"}
                          </Button>
                        </Box>
                        {/* Info message while still running */}
                        {running && domainResult.best_success && (
                          <Alert
                            severity="info"
                            sx={{
                              borderRadius: 0,
                              bgcolor: colors.accent.secondary,
                              "& .MuiAlert-icon": { color: colors.secondary },
                              borderBottom: `1px solid ${colors.border.default}`,
                            }}
                          >
                            Found a working configuration! Still testing{" "}
                            {suite ? suite.total_checks - totalCount : "..."}{" "}
                            more configs — a faster option may be found.
                          </Alert>
                        )}
                      </Box>
                    )}

                    {/* Expanded Details */}
                    <Collapse in={isExpanded}>
                      <Box sx={{ p: 3 }}>
                        <Divider
                          sx={{ my: 2, borderColor: colors.border.default }}
                        />

                        {/* Results by Phase */}
                        {(
                          [
                            "baseline",
                            "strategy_detection",
                            "optimization",
                            "combination",
                          ] as DiscoveryPhase[]
                        )
                          .filter((phase) => groupedResults[phase].length > 0)
                          .map((phase) => (
                            <Box key={phase} sx={{ mb: 3 }}>
                              <Typography
                                variant="subtitle2"
                                sx={{
                                  color: colors.text.secondary,
                                  mb: 1.5,
                                  textTransform: "uppercase",
                                  fontSize: "0.7rem",
                                  display: "flex",
                                  alignItems: "center",
                                  gap: 1,
                                }}
                              >
                                {phaseNames[phase]}
                                <Chip
                                  label={groupedResults[phase].length}
                                  size="small"
                                  sx={{ height: 18, fontSize: "0.65rem" }}
                                />
                              </Typography>
                              <Stack
                                direction="row"
                                spacing={1}
                                flexWrap="wrap"
                                gap={1}
                              >
                                {groupedResults[phase]
                                  .sort((a, b) => b.speed - a.speed)
                                  .map((result) => (
                                    <Box
                                      key={result.preset_name}
                                      sx={{
                                        display: "flex",
                                        alignItems: "center",
                                        gap: 0.5,
                                      }}
                                    >
                                      <Tooltip
                                        title={
                                          result.status === "complete"
                                            ? `${result.preset_name}: ${(
                                                result.speed /
                                                1024 /
                                                1024
                                              ).toFixed(2)} MB/s`
                                            : `${result.preset_name}: ${
                                                result.error || "Failed"
                                              }`
                                        }
                                      >
                                        <Chip
                                          label={`${result.preset_name}: ${
                                            result.status === "complete"
                                              ? `${(
                                                  result.speed /
                                                  1024 /
                                                  1024
                                                ).toFixed(2)} MB/s`
                                              : "Failed"
                                          }`}
                                          size="small"
                                          sx={{
                                            bgcolor:
                                              result.preset_name ===
                                              domainResult.best_preset
                                                ? colors.accent.secondary
                                                : colors.background.dark,
                                            color:
                                              result.status === "complete"
                                                ? colors.text.primary
                                                : colors.quaternary,
                                            border:
                                              result.preset_name ===
                                              domainResult.best_preset
                                                ? `2px solid ${colors.secondary}`
                                                : `1px solid ${colors.border.light}`,
                                          }}
                                        />
                                      </Tooltip>
                                      {result.status === "complete" &&
                                        result.preset_name !==
                                          domainResult.best_preset && (
                                          <Tooltip title="Use this configuration">
                                            <IconButton
                                              size="small"
                                              onClick={() => {
                                                void handleAddStrategy(
                                                  domainResult.domain,
                                                  result
                                                );
                                              }}
                                              disabled={addingPreset}
                                              sx={{
                                                p: 0.5,
                                                bgcolor: colors.background.dark,
                                                border: `1px solid ${colors.border.light}`,
                                                "&:hover": {
                                                  bgcolor:
                                                    colors.accent.secondary,
                                                  borderColor: colors.secondary,
                                                },
                                              }}
                                            >
                                              <AddIcon fontSize="small" />
                                            </IconButton>
                                          </Tooltip>
                                        )}
                                    </Box>
                                  ))}
                              </Stack>
                            </Box>
                          ))}
                      </Box>
                    </Collapse>

                    {/* Failed state */}
                    {!domainResult.best_success && !running && (
                      <Box sx={{ p: 3 }}>
                        <Alert severity="error">
                          All {Object.keys(domainResult.results).length} tested
                          configurations failed for this domain. Check your
                          network connection and domain accessibility.
                        </Alert>
                      </Box>
                    )}
                    {!domainResult.best_success && running && (
                      <Box sx={{ p: 2, bgcolor: colors.background.default }}>
                        <Typography
                          variant="body2"
                          sx={{
                            color: colors.text.secondary,
                            display: "flex",
                            alignItems: "center",
                            gap: 1,
                          }}
                        >
                          <CircularProgress
                            size={14}
                            sx={{ color: colors.text.secondary }}
                          />
                          {suite && suite.total_checks > totalCount
                            ? `${
                                suite.total_checks - totalCount
                              } more configurations to test...`
                            : "Testing configurations..."}
                        </Typography>
                      </Box>
                    )}
                  </Paper>
                );
              })}
          </Stack>
        )}

      <DiscoveryAddDialog
        open={addDialog.open}
        domain={addDialog.domain}
        presetName={addDialog.presetName}
        setConfig={addDialog.setConfig}
        onClose={() =>
          setAddDialog({
            open: false,
            domain: "",
            presetName: "",
            setConfig: null,
          })
        }
        onAddNew={(name: string, domain: string) => {
          void handleAddNew(name, domain);
        }}
        onAddToExisting={(setId: string, domain: string) => {
          void handleAddToExisting(setId, domain);
        }}
        loading={addingPreset}
      />

      <Snackbar
        open={snackbar.open}
        autoHideDuration={6000}
        onClose={() => setSnackbar({ ...snackbar, open: false })}
        anchorOrigin={{ vertical: "bottom", horizontal: "right" }}
      >
        <Alert
          onClose={() => setSnackbar({ ...snackbar, open: false })}
          severity={snackbar.severity}
          sx={{ width: "100%" }}
        >
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Stack>
  );
};
