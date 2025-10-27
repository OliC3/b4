import React from "react";
import {
  Box,
  Container,
  IconButton,
  Paper,
  Stack,
  Typography,
  Switch,
  FormControlLabel,
  TextField,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  List,
  ListItem,
  ListItemButton,
  ListItemText,
  ListItemIcon,
  Radio,
  Alert,
  Snackbar,
} from "@mui/material";
import RefreshIcon from "@mui/icons-material/Refresh";
import CheckCircleIcon from "@mui/icons-material/CheckCircle";
import TcpIcon from "@mui/icons-material/SyncAlt";
import UdpIcon from "@mui/icons-material/TrendingFlat";
import AddIcon from "@mui/icons-material/Add";
import DomainIcon from "@mui/icons-material/Language";

import { colors } from "../../Theme";

interface ParsedLog {
  timestamp: string;
  protocol: "TCP" | "UDP";
  isTarget: boolean;
  domain: string;
  source: string;
  destination: string;
  raw: string;
}

interface DomainModalState {
  open: boolean;
  domain: string;
  variants: string[];
  selected: string;
}

function parseLogLine(line: string): ParsedLog | null {
  // Example: 2025/10/13 22:41:12.466126 [INFO] SNI TCP: assets.alicdn.com 192.168.1.100:38894 -> 92.123.206.67:443
  const regex =
    /^(\d{4}\/\d{2}\/\d{2} \d{2}:\d{2}:\d{2}\.\d+)\s+\[INFO\]\s+SNI\s+(TCP|UDP)(?:\s+TARGET)?:\s+(\S+)\s+(\S+)\s+->\s+(\S+)$/;
  const match = line.match(regex);

  if (!match) return null;

  const [, timestamp, protocol, domain, source, destination] = match;
  const isTarget = line.includes("TARGET");

  return {
    timestamp,
    protocol: protocol as "TCP" | "UDP",
    isTarget,
    domain,
    source,
    destination,
    raw: line,
  };
}

// Generate domain variants from most specific to least specific
function generateDomainVariants(domain: string): string[] {
  const parts = domain.split(".");
  const variants: string[] = [];

  // Generate from full domain to TLD+1 (e.g., example.com)
  for (let i = 0; i < parts.length - 1; i++) {
    variants.push(parts.slice(i).join("."));
  }

  return variants;
}

const STORAGE_KEY = "b4_domains_lines";
const MAX_STORED_LINES = 1000;

export default function Domains() {
  // Load persisted lines from localStorage on mount
  const [lines, setLines] = React.useState<string[]>(() => {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (stored) {
        const parsed = JSON.parse(stored);
        return Array.isArray(parsed) ? parsed : [];
      }
    } catch (e) {
      console.error("Failed to load persisted domains:", e);
    }
    return [];
  });

  const [paused, setPaused] = React.useState(false);
  const [filter, setFilter] = React.useState("");
  const [autoScroll, setAutoScroll] = React.useState(true);
  const tableRef = React.useRef<HTMLDivElement | null>(null);

  // Modal state
  const [modalState, setModalState] = React.useState<DomainModalState>({
    open: false,
    domain: "",
    variants: [],
    selected: "",
  });

  // Snackbar state for notifications
  const [snackbar, setSnackbar] = React.useState({
    open: false,
    message: "",
    severity: "success" as "success" | "error",
  });

  // Persist lines to localStorage whenever they change
  React.useEffect(() => {
    try {
      localStorage.setItem(
        STORAGE_KEY,
        JSON.stringify(lines.slice(-MAX_STORED_LINES))
      );
    } catch (e) {
      console.error("Failed to persist domains:", e);
    }
  }, [lines]);

  React.useEffect(() => {
    const ws = new WebSocket(
      (location.protocol === "https:" ? "wss://" : "ws://") +
        location.host +
        "/api/ws/logs"
    );
    ws.onmessage = (ev) => {
      if (!paused) setLines((prev) => [...prev.slice(-999), String(ev.data)]);
    };
    ws.onerror = () => setLines((prev) => [...prev, "[WS ERROR]"]);
    return () => ws.close();
  }, [paused]);

  React.useEffect(() => {
    const el = tableRef.current;
    if (el && autoScroll) {
      el.scrollTop = el.scrollHeight;
    }
  }, [lines, autoScroll]);

  const parsedLogs = React.useMemo(() => {
    return lines
      .map(parseLogLine)
      .filter((log): log is ParsedLog => log !== null);
  }, [lines]);

  const filtered = React.useMemo(() => {
    const f = filter.trim().toLowerCase();
    const filters = f
      .split("+")
      .map((s) => s.trim())
      .filter((s) => s.length > 0);

    if (filters.length === 0) {
      return parsedLogs;
    }

    // Group filters by field
    const fieldFilters: Record<string, string[]> = {};
    const globalFilters: string[] = [];

    filters.forEach((filterTerm) => {
      const colonIndex = filterTerm.indexOf(":");

      if (colonIndex > 0) {
        const field = filterTerm.substring(0, colonIndex);
        const value = filterTerm.substring(colonIndex + 1);

        if (!fieldFilters[field]) {
          fieldFilters[field] = [];
        }
        fieldFilters[field].push(value);
      } else {
        globalFilters.push(filterTerm);
      }
    });

    return parsedLogs.filter((log) => {
      // Check field-specific filters (OR within field, AND across fields)
      for (const [field, values] of Object.entries(fieldFilters)) {
        const fieldValue =
          log[field as keyof typeof log]?.toString().toLowerCase() || "";
        const matches = values.some((value) => fieldValue.includes(value));
        if (!matches) return false;
      }

      // Check global filters (must match at least one field)
      for (const filterTerm of globalFilters) {
        const matches =
          log.domain.toLowerCase().includes(filterTerm) ||
          log.source.toLowerCase().includes(filterTerm) ||
          log.protocol.toLowerCase().includes(filterTerm) ||
          log.destination.toLowerCase().includes(filterTerm);
        if (!matches) return false;
      }

      return true;
    });
  }, [parsedLogs, filter]);

  const handleScroll = () => {
    const el = tableRef.current;
    if (el) {
      const isAtBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 50;
      setAutoScroll(isAtBottom);
    }
  };

  const handleDomainClick = (domain: string) => {
    const variants = generateDomainVariants(domain);
    setModalState({
      open: true,
      domain,
      variants,
      selected: variants[0] || domain,
    });
  };

  const handleModalClose = () => {
    setModalState({
      open: false,
      domain: "",
      variants: [],
      selected: "",
    });
  };

  const handleAddDomain = async () => {
    if (!modalState.selected) return;

    try {
      const response = await fetch("/api/geosite/domain", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          domain: modalState.selected,
        }),
      });

      if (response.ok) {
        const data = await response.json();
        setSnackbar({
          open: true,
          message: `Successfully added "${modalState.selected}" to manual domains`,
          severity: "success",
        });
        handleModalClose();
      } else {
        const error = await response.json();
        setSnackbar({
          open: true,
          message: `Failed to add domain: ${error.message}`,
          severity: "error",
        });
      }
    } catch (error) {
      setSnackbar({
        open: true,
        message: `Error adding domain: ${error}`,
        severity: "error",
      });
    }
  };

  return (
    <Container
      maxWidth={false}
      sx={{
        flex: 1,
        py: 3,
        px: 3,
        display: "flex",
        flexDirection: "column",
        overflow: "hidden",
      }}
    >
      <Paper
        elevation={0}
        variant="outlined"
        sx={{
          flex: 1,
          display: "flex",
          flexDirection: "column",
          overflow: "hidden",
          border: "1px solid",
          borderColor: paused ? colors.border.strong : colors.border.default,
          transition: "border-color 0.3s",
        }}
      >
        {/* Controls Bar */}
        <Box
          sx={{
            p: 2,
            borderBottom: "1px solid",
            borderColor: colors.border.light,
            bgcolor: colors.background.control,
          }}
        >
          <Stack direction="row" spacing={2} alignItems="center">
            <TextField
              size="small"
              placeholder="Filter entries (use `+` to combine, e.g. `tcp+domain2`, or `tcp+domain:exmpl1+domain:exmpl2`)"
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              sx={{ flex: 1 }}
              InputProps={{
                sx: {
                  bgcolor: colors.background.dark,
                  "& fieldset": {
                    borderColor: `${colors.border.default} !important`,
                  },
                },
              }}
            />
            <Stack direction="row" spacing={1} alignItems="center">
              <Chip
                label={`${parsedLogs.length} connections`}
                size="small"
                sx={{
                  bgcolor: colors.accent.secondary,
                  color: colors.secondary,
                  fontWeight: 600,
                }}
              />
              {filter && (
                <Chip
                  label={`${filtered.length} filtered`}
                  size="small"
                  sx={{
                    bgcolor: colors.accent.primary,
                    color: colors.primary,
                    borderColor: colors.primary,
                  }}
                  variant="outlined"
                />
              )}
            </Stack>
            <FormControlLabel
              control={
                <Switch
                  checked={paused}
                  onChange={(e) => setPaused(e.target.checked)}
                  sx={{
                    "& .MuiSwitch-switchBase.Mui-checked": {
                      color: colors.secondary,
                    },
                    "& .MuiSwitch-switchBase.Mui-checked + .MuiSwitch-track": {
                      backgroundColor: colors.secondary,
                    },
                  }}
                />
              }
              label={
                <Typography
                  sx={{
                    color: paused ? colors.secondary : "text.secondary",
                    fontWeight: paused ? 600 : 400,
                  }}
                >
                  {paused ? "Paused" : "Streaming"}
                </Typography>
              }
            />
            <IconButton
              color="inherit"
              onClick={() => {
                setLines([]);
                localStorage.removeItem(STORAGE_KEY);
              }}
              sx={{
                color: "text.secondary",
                "&:hover": {
                  color: colors.secondary,
                  bgcolor: colors.accent.secondaryHover,
                },
              }}
            >
              <RefreshIcon />
            </IconButton>
          </Stack>
        </Box>

        {/* Table Display */}
        <TableContainer
          ref={tableRef}
          onScroll={handleScroll}
          sx={{
            flex: 1,
            backgroundColor: colors.background.dark,
          }}
        >
          <Table stickyHeader size="small">
            <TableHead>
              <TableRow>
                <TableCell
                  sx={{
                    bgcolor: colors.background.paper,
                    color: colors.secondary,
                    fontWeight: 600,
                    borderBottom: `2px solid ${colors.border.default}`,
                  }}
                >
                  Time
                </TableCell>
                <TableCell
                  sx={{
                    bgcolor: colors.background.paper,
                    color: colors.secondary,
                    fontWeight: 600,
                    borderBottom: `2px solid ${colors.border.default}`,
                  }}
                >
                  Protocol
                </TableCell>
                <TableCell
                  sx={{
                    bgcolor: colors.background.paper,
                    color: colors.secondary,
                    fontWeight: 600,
                    borderBottom: `2px solid ${colors.border.default}`,
                    width: 80,
                  }}
                >
                  Target
                </TableCell>
                <TableCell
                  sx={{
                    bgcolor: colors.background.paper,
                    color: colors.secondary,
                    fontWeight: 600,
                    borderBottom: `2px solid ${colors.border.default}`,
                  }}
                >
                  Domain
                </TableCell>
                <TableCell
                  sx={{
                    bgcolor: colors.background.paper,
                    color: colors.secondary,
                    fontWeight: 600,
                    borderBottom: `2px solid ${colors.border.default}`,
                  }}
                >
                  Source
                </TableCell>
                <TableCell
                  sx={{
                    bgcolor: colors.background.paper,
                    color: colors.secondary,
                    fontWeight: 600,
                    borderBottom: `2px solid ${colors.border.default}`,
                  }}
                >
                  Destination
                </TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {filtered.length === 0 && parsedLogs.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={6}
                    sx={{
                      textAlign: "center",
                      py: 4,
                      color: "text.secondary",
                      fontStyle: "italic",
                      bgcolor: colors.background.dark,
                      borderBottom: "none",
                    }}
                  >
                    Waiting for connections...
                  </TableCell>
                </TableRow>
              ) : filtered.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={6}
                    sx={{
                      textAlign: "center",
                      py: 4,
                      color: "text.secondary",
                      fontStyle: "italic",
                      bgcolor: colors.background.dark,
                      borderBottom: "none",
                    }}
                  >
                    No connections match your filter
                  </TableCell>
                </TableRow>
              ) : (
                filtered.map((log, i) => (
                  <TableRow
                    key={i}
                    sx={{
                      "&:hover": {
                        bgcolor: colors.accent.primaryStrong,
                      },
                    }}
                  >
                    <TableCell
                      sx={{
                        color: "text.secondary",
                        fontFamily: "monospace",
                        fontSize: 12,
                        borderBottom: `1px solid ${colors.border.light}`,
                      }}
                    >
                      {log.timestamp.split(" ")[1]}
                    </TableCell>
                    <TableCell
                      sx={{
                        borderBottom: `1px solid ${colors.border.light}`,
                      }}
                    >
                      <Chip
                        label={log.protocol}
                        size="small"
                        icon={
                          log.protocol === "TCP" ? (
                            <TcpIcon color="primary" />
                          ) : (
                            <UdpIcon color="secondary" />
                          )
                        }
                        sx={{
                          bgcolor: colors.accent.primary,
                          color:
                            log.protocol === "TCP"
                              ? colors.primary
                              : colors.secondary,
                          fontWeight: 600,
                        }}
                      />
                    </TableCell>
                    <TableCell
                      sx={{
                        textAlign: "center",
                        borderBottom: `1px solid ${colors.border.light}`,
                      }}
                    >
                      {log.isTarget && (
                        <CheckCircleIcon
                          sx={{ color: colors.secondary, fontSize: 18 }}
                        />
                      )}
                    </TableCell>
                    <TableCell
                      sx={{
                        color: "text.primary",
                        fontWeight: 500,
                        borderBottom: `1px solid ${colors.border.light}`,
                        cursor: "pointer",
                        "&:hover": {
                          bgcolor: colors.accent.primary,
                          color: colors.secondary,
                        },
                      }}
                      onClick={() => handleDomainClick(log.domain)}
                    >
                      <Stack direction="row" spacing={1} alignItems="center">
                        <Typography>{log.domain}</Typography>
                        <AddIcon sx={{ fontSize: 16, opacity: 0.7 }} />
                      </Stack>
                    </TableCell>
                    <TableCell
                      sx={{
                        color: "text.secondary",
                        fontFamily: "monospace",
                        fontSize: 12,
                        borderBottom: `1px solid ${colors.border.light}`,
                      }}
                    >
                      {log.source}
                    </TableCell>
                    <TableCell
                      sx={{
                        color: "text.secondary",
                        fontFamily: "monospace",
                        fontSize: 12,
                        borderBottom: `1px solid ${colors.border.light}`,
                      }}
                    >
                      {log.destination}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </TableContainer>
      </Paper>

      {/* Domain Selection Modal */}
      <Dialog
        open={modalState.open}
        onClose={handleModalClose}
        maxWidth="sm"
        fullWidth
        PaperProps={{
          sx: {
            bgcolor: colors.background.paper,
            border: `1px solid ${colors.border.default}`,
          },
        }}
      >
        <DialogTitle sx={{ borderBottom: `1px solid ${colors.border.light}` }}>
          <Stack direction="row" spacing={1} alignItems="center">
            <DomainIcon sx={{ color: colors.primary }} />
            <Typography>Add Domain to Manual List</Typography>
          </Stack>
        </DialogTitle>
        <DialogContent sx={{ mt: 2 }}>
          <Alert severity="info" sx={{ mb: 2 }}>
            Select which domain pattern to add to the manual domains list. More
            specific patterns will only match exact subdomains, while broader
            patterns will match all subdomains.
          </Alert>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
            Original domain: <strong>{modalState.domain}</strong>
          </Typography>
          <List>
            {modalState.variants.map((variant, index) => (
              <ListItem key={variant} disablePadding>
                <ListItemButton
                  onClick={() =>
                    setModalState((prev) => ({ ...prev, selected: variant }))
                  }
                  selected={modalState.selected === variant}
                  sx={{
                    borderRadius: 1,
                    mb: 0.5,
                    "&.Mui-selected": {
                      bgcolor: colors.accent.primary,
                      "&:hover": {
                        bgcolor: colors.accent.primaryHover,
                      },
                    },
                  }}
                >
                  <ListItemIcon>
                    <Radio
                      checked={modalState.selected === variant}
                      sx={{
                        color: colors.border.default,
                        "&.Mui-checked": {
                          color: colors.primary,
                        },
                      }}
                    />
                  </ListItemIcon>
                  <ListItemText
                    primary={variant}
                    secondary={
                      index === 0
                        ? "Most specific - exact match only"
                        : index === modalState.variants.length - 1
                        ? "Broadest - matches all subdomains"
                        : "Intermediate specificity"
                    }
                  />
                </ListItemButton>
              </ListItem>
            ))}
          </List>
        </DialogContent>
        <DialogActions
          sx={{ borderTop: `1px solid ${colors.border.light}`, p: 2 }}
        >
          <Button onClick={handleModalClose} color="inherit">
            Cancel
          </Button>
          <Button
            onClick={handleAddDomain}
            variant="contained"
            startIcon={<AddIcon />}
            disabled={!modalState.selected}
            sx={{
              bgcolor: colors.primary,
              "&:hover": {
                bgcolor: colors.secondary,
              },
            }}
          >
            Add Domain
          </Button>
        </DialogActions>
      </Dialog>

      {/* Snackbar for notifications */}
      <Snackbar
        open={snackbar.open}
        autoHideDuration={6000}
        onClose={() => setSnackbar((prev) => ({ ...prev, open: false }))}
        anchorOrigin={{ vertical: "bottom", horizontal: "right" }}
      >
        <Alert
          onClose={() => setSnackbar((prev) => ({ ...prev, open: false }))}
          severity={snackbar.severity}
          sx={{ width: "100%" }}
        >
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Container>
  );
}
