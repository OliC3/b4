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
} from "@mui/material";
import RefreshIcon from "@mui/icons-material/Refresh";
import CheckCircleIcon from "@mui/icons-material/CheckCircle";
import TcpIcon from "@mui/icons-material/SyncAlt";
import UdpIcon from "@mui/icons-material/TrendingFlat";

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

export default function Domains() {
  const [lines, setLines] = React.useState<string[]>([]);
  const [paused, setPaused] = React.useState(false);
  const [filter, setFilter] = React.useState("");
  const [autoScroll, setAutoScroll] = React.useState(true);
  const tableRef = React.useRef<HTMLDivElement | null>(null);

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
              onClick={() => setLines([])}
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
                            <UdpIcon color="error" />
                          )
                        }
                        sx={{
                          bgcolor:
                            log.protocol === "TCP"
                              ? colors.accent.primary
                              : colors.accent.tertiary,
                          color:
                            log.protocol === "TCP"
                              ? colors.primary
                              : colors.tertiary,
                          fontWeight: 600,
                          fontSize: 11,
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
                      }}
                    >
                      {log.domain}
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
    </Container>
  );
}
