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
  Grid,
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
} from "@mui/icons-material";
import { button_secondary, colors } from "@design";
import { useTestDomains } from "@hooks/useTestDomains";
import { B4SetConfig } from "@/models/Config";
import SettingTextField from "@atoms/common/B4TextField";
import { AddSniModal } from "@organisms/domains/AddSniModal";
import { generateDomainVariants } from "@utils";

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

interface DomainDiscoveryResult {
  domain: string;
  best_preset: string;
  best_speed: number;
  best_success: boolean;
  results: Record<string, DomainPresetResult>;
  working_families?: StrategyFamily[];
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
  working_families?: string[];
  domain_discovery_results?: Record<string, DomainDiscoveryResult>;
}

interface DiscoveryStartResponse {
  id: string;
  total_domains: number;
  total_clusters: number;
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
  const [addingPreset, setAddingPreset] = useState<string | null>(null);
  const [variants, setVariants] = useState<string[]>([]);
  const [selectedVariant, setSelectedVariant] = useState<string | null>(null);
  const [expandedDomains, setExpandedDomains] = useState<Set<string>>(
    new Set()
  );
  const [snackbar, setSnackbar] = useState<{
    open: boolean;
    message: string;
    severity: "success" | "error";
  }>({ open: false, message: "", severity: "success" });

  const { domains, addDomain, removeDomain, clearDomains, resetToDefaults } =
    useTestDomains();
  const [newDomain, setNewDomain] = useState("");

  const [variantModal, setVariantModal] = useState<{
    open: boolean;
    domain: string;
    result: DomainPresetResult | null;
  }>({ open: false, domain: "", result: null });

  const handleAddStrategy = (domain: string, result: DomainPresetResult) => {
    const domainVariants = generateDomainVariants(domain);
    setVariants(domainVariants);
    setSelectedVariant(domainVariants[0]);
    setVariantModal({ open: true, domain, result });
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
    if (domains.length === 0) {
      setError("Add at least one domain to test");
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
          domains: domains,
        }),
      });

      if (!response.ok) {
        const text = await response.text();
        throw new Error(text || "Failed to start discovery");
      }

      const data = (await response.json()) as DiscoveryStartResponse;
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
    setSuiteId(null);
    setSuite(null);
    setError(null);
    setRunning(false);
    setExpandedDomains(new Set());
  };

  const confirmAddStrategy = async () => {
    if (!variantModal.result?.set) return;

    try {
      const configToAdd = {
        ...variantModal.result.set,
        targets: {
          ...variantModal.result.set.targets,
          sni_domains: [selectedVariant],
        },
      };
      setAddingPreset(
        `${variantModal.domain}-${variantModal.result.preset_name}`
      );

      const response = await fetch("/api/discovery/add", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(configToAdd),
      });

      if (!response.ok) {
        throw new Error("Failed to add configuration");
      }

      const data = (await response.json()) as { message: string };

      setSnackbar({
        open: true,
        message: `${data.message}`,
        severity: "success",
      });
      setVariantModal({ open: false, domain: "", result: null });
    } catch (err) {
      console.error("Failed to add strategy:", err);
      setSnackbar({
        open: true,
        message: "Failed to add configuration",
        severity: "error",
      });
    } finally {
      setAddingPreset(null);
    }
  };

  const progress = suite
    ? (suite.completed_checks / suite.total_checks) * 100
    : 0;

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
          {/* Header with actions */}
          <Box
            sx={{
              display: "flex",
              alignItems: "center",
              justifyContent: "space-between",
            }}
          >
            <Box>
              <Typography variant="h6" sx={{ color: colors.text.primary }}>
                Configuration Discovery
              </Typography>
              <Typography
                variant="caption"
                sx={{ color: colors.text.secondary }}
              >
                Hierarchical testing: Strategy Detection → Optimization →
                Combination
              </Typography>
            </Box>
            <Stack direction="row" spacing={1}>
              {!running && !suite && (
                <Button
                  variant="contained"
                  startIcon={<StartIcon />}
                  onClick={() => {
                    void startDiscovery();
                  }}
                  disabled={domains.length === 0}
                  sx={{
                    bgcolor: colors.secondary,
                    "&:hover": { bgcolor: colors.primary },
                    "&:disabled": {
                      bgcolor: colors.accent.secondary,
                      color: colors.text.secondary,
                    },
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

          {error && <Alert severity="error">{error}</Alert>}

          {/* Domain Management Section */}
          <Box>
            <Stack
              direction="row"
              alignItems="center"
              justifyContent="space-between"
              sx={{ mb: 1 }}
            >
              <Typography
                variant="subtitle2"
                sx={{ color: colors.text.primary }}
              >
                Domains to Discover
              </Typography>
              <Stack direction="row" spacing={1}>
                <Button
                  size="small"
                  onClick={resetToDefaults}
                  disabled={running}
                  sx={{ ...button_secondary, textTransform: "none" }}
                >
                  Reset to Defaults
                </Button>
                <Button
                  size="small"
                  onClick={clearDomains}
                  disabled={running || domains.length === 0}
                  sx={{ ...button_secondary, textTransform: "none" }}
                >
                  Clear All
                </Button>
              </Stack>
            </Stack>

            <Grid container spacing={2}>
              <Grid size={{ sm: 12, md: 6 }}>
                <Box
                  sx={{
                    display: "flex",
                    gap: 1,
                    pb: 2,
                    width: "100%",
                    alignItems: "flex-start",
                  }}
                >
                  <SettingTextField
                    fullWidth
                    label="Add domain"
                    value={newDomain}
                    onChange={(e) => setNewDomain(e.target.value)}
                    onKeyDown={(e) => {
                      if (
                        e.key === "Enter" ||
                        e.key === "," ||
                        e.key === "Tab"
                      ) {
                        e.preventDefault();
                        addDomain(newDomain);
                        setNewDomain("");
                      }
                    }}
                    placeholder="youtube.com"
                    disabled={running}
                    helperText="Press Enter or comma to add"
                  />
                  <IconButton
                    onClick={() => {
                      addDomain(newDomain);
                      setNewDomain("");
                    }}
                    disabled={running || !newDomain.trim()}
                    sx={{
                      bgcolor: colors.accent.secondary,
                      color: colors.secondary,
                      "&:hover": {
                        bgcolor: colors.accent.secondaryHover,
                      },
                    }}
                  >
                    <AddIcon />
                  </IconButton>
                </Box>
              </Grid>
              <Grid size={{ sm: 12, md: 6 }}>
                <Box
                  sx={{
                    display: "flex",
                    flexWrap: "wrap",
                    gap: 1,
                    p: 2,
                    width: "100%",
                    border: `1px solid ${colors.border.default}`,
                    borderRadius: 1,
                    bgcolor: colors.background.dark,
                    maxHeight: 120,
                    overflowY: "auto",
                  }}
                >
                  {domains.length === 0 ? (
                    <Typography
                      variant="body2"
                      sx={{
                        color: colors.text.secondary,
                        width: "100%",
                        textAlign: "center",
                      }}
                    >
                      No domains added. Add domains above or click "Reset to
                      Defaults"
                    </Typography>
                  ) : (
                    domains.map((domain) => (
                      <Chip
                        size="small"
                        key={domain}
                        label={domain}
                        onDelete={() => removeDomain(domain)}
                        disabled={running}
                        sx={{
                          bgcolor: colors.accent.primary,
                          color: colors.secondary,
                          "& .MuiChip-deleteIcon": {
                            color: colors.secondary,
                          },
                        }}
                      />
                    ))
                  )}
                </Box>
              </Grid>
            </Grid>
          </Box>

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
        </Stack>
      </Paper>

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
                        sx={{ color: colors.secondary, fontWeight: 600 }}
                      >
                        {domainResult.best_success
                          ? `${(domainResult.best_speed / 1024 / 1024).toFixed(
                              2
                            )} MB/s`
                          : "No working config"}
                      </Typography>
                    </Box>

                    {/* Best Configuration Quick View (always visible) */}
                    {domainResult.best_success && (
                      <Box
                        sx={{
                          p: 2,
                          bgcolor: colors.background.default,
                          display: "flex",
                          alignItems: "center",
                          justifyContent: "space-between",
                          borderBottom: `1px solid ${colors.border.default}`,
                        }}
                      >
                        <Box
                          sx={{ display: "flex", alignItems: "center", gap: 2 }}
                        >
                          <SpeedIcon sx={{ color: colors.secondary }} />
                          <Box>
                            <Typography
                              variant="caption"
                              sx={{ color: colors.text.secondary }}
                            >
                              Best Configuration
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
                                  sx={{ ml: 1, bgcolor: colors.accent.primary }}
                                />
                              )}
                            </Typography>
                          </Box>
                        </Box>
                        <Button
                          variant="contained"
                          startIcon={
                            addingPreset ===
                            `${domainResult.domain}-${domainResult.best_preset}` ? (
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
                          disabled={
                            addingPreset ===
                            `${domainResult.domain}-${domainResult.best_preset}`
                          }
                          sx={{
                            bgcolor: colors.secondary,
                            color: colors.background.default,
                            "&:hover": { bgcolor: colors.primary },
                          }}
                        >
                          Use This Strategy
                        </Button>
                      </Box>
                    )}

                    {/* Expanded Details */}
                    <Collapse in={isExpanded}>
                      <Box sx={{ p: 3 }}>
                        {/* Working Families */}
                        {domainResult.working_families &&
                          domainResult.working_families.length > 0 && (
                            <Box sx={{ mb: 3 }}>
                              <Typography
                                variant="subtitle2"
                                sx={{
                                  color: colors.text.secondary,
                                  mb: 1,
                                  textTransform: "uppercase",
                                  fontSize: "0.7rem",
                                }}
                              >
                                Working Strategy Families
                              </Typography>
                              <Stack
                                direction="row"
                                spacing={1}
                                flexWrap="wrap"
                                gap={1}
                              >
                                {domainResult.working_families.map((family) => (
                                  <Chip
                                    key={family}
                                    label={familyNames[family]}
                                    sx={{
                                      bgcolor: colors.accent.secondary,
                                      color: colors.secondary,
                                    }}
                                  />
                                ))}
                              </Stack>
                            </Box>
                          )}

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
                                              disabled={
                                                addingPreset ===
                                                `${domainResult.domain}-${result.preset_name}`
                                              }
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
                    {!domainResult.best_success && (
                      <Box sx={{ p: 3 }}>
                        <Alert severity="error">
                          All {Object.keys(domainResult.results).length} tested
                          configurations failed for this domain. Check your
                          network connection and domain accessibility.
                        </Alert>
                      </Box>
                    )}
                  </Paper>
                );
              })}
          </Stack>
        )}

      <AddSniModal
        open={variantModal.open}
        domain={variantModal.domain}
        variants={variants}
        selected={selectedVariant || variants[0]}
        sets={[]}
        createNewSet={true}
        onClose={() => {
          setVariantModal({ open: false, domain: "", result: null });
          setSelectedVariant(null);
        }}
        onSelectVariant={(variant) => {
          setSelectedVariant(variant);
        }}
        onAdd={() => {
          void confirmAddStrategy();
        }}
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
