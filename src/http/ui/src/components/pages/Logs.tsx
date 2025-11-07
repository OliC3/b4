import React, { useEffect } from "react";
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
} from "@mui/material";
import RefreshIcon from "@mui/icons-material/DeleteForever";
import KeyboardArrowDownIcon from "@mui/icons-material/KeyboardArrowDown";

export default function Logs() {
  const [lines, setLines] = React.useState<string[]>([]);
  const [paused, setPaused] = React.useState(false);
  const [filter, setFilter] = React.useState("");
  const [autoScroll, setAutoScroll] = React.useState(true);
  const [showScrollBtn, setShowScrollBtn] = React.useState(false);
  const logRef = React.useRef<HTMLDivElement | null>(null);

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
    const el = logRef.current;
    if (el && autoScroll) {
      el.scrollTop = el.scrollHeight;
    }
  }, [lines, autoScroll]);

  const handleScroll = () => {
    const el = logRef.current;
    if (el) {
      const isAtBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 50;
      setAutoScroll(isAtBottom);
      setShowScrollBtn(!isAtBottom);
    }
  };

  const scrollToBottom = () => {
    const el = logRef.current;
    if (el) {
      el.scrollTop = el.scrollHeight;
      setAutoScroll(true);
      setShowScrollBtn(false);
    }
  };

  const filtered = React.useMemo(() => {
    const f = filter.trim().toLowerCase();
    return f ? lines.filter((l) => l.toLowerCase().includes(f)) : lines;
  }, [lines, filter]);

  const handleHotkeysDown = (e: KeyboardEvent) => {
    // Ignore shortcuts when typing in input fields
    const target = e.target as HTMLElement;
    if (
      target.tagName === "INPUT" ||
      target.tagName === "TEXTAREA" ||
      target.isContentEditable
    ) {
      return;
    }

    if ((e.ctrlKey && e.key === "x") || e.key === "Delete") {
      e.preventDefault();
      setLines([]);
    } else if (e.key === "p" || e.key === "Pause") {
      e.preventDefault();
      setPaused((prev) => !prev);
    }
  };

  useEffect(() => {
    globalThis.window.addEventListener("keydown", handleHotkeysDown);
    return () => {
      globalThis.window.removeEventListener("keydown", handleHotkeysDown);
    };
  }, [handleHotkeysDown]);

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
          borderColor: paused
            ? "rgba(245, 173, 24, 0.5)"
            : "rgba(245, 173, 24, 0.24)",
          transition: "border-color 0.3s",
        }}
      >
        {/* Controls Bar */}
        <Box
          sx={{
            p: 2,
            borderBottom: "1px solid",
            borderColor: "rgba(245, 173, 24, 0.12)",
            bgcolor: "rgba(31, 18, 24, 0.6)",
          }}
        >
          <Stack direction="row" spacing={2} alignItems="center">
            <TextField
              size="small"
              placeholder="Filter logs..."
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              sx={{ flex: 1 }}
              slotProps={{
                input: {
                  sx: {
                    bgcolor: "rgba(15, 10, 14, 0.5)",
                    "& fieldset": {
                      borderColor: "rgba(245, 173, 24, 0.24) !important",
                    },
                  },
                },
              }}
            />
            <Stack direction="row" spacing={1} alignItems="center">
              <Chip
                label={`${lines.length} lines`}
                size="small"
                sx={{
                  bgcolor: "rgba(245, 173, 24, 0.2)",
                  color: "#F5AD18",
                  fontWeight: 600,
                }}
              />
              {filter && (
                <Chip
                  label={`${filtered.length} filtered`}
                  size="small"
                  sx={{
                    bgcolor: "rgba(158, 28, 96, 0.3)",
                    color: "#9E1C60",
                    borderColor: "#9E1C60",
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
                      color: "#F5AD18",
                    },
                    "& .MuiSwitch-switchBase.Mui-checked + .MuiSwitch-track": {
                      backgroundColor: "#F5AD18",
                    },
                  }}
                />
              }
              label={
                <Typography
                  sx={{
                    color: paused ? "#F5AD18" : "text.secondary",
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
                  color: "#F5AD18",
                  bgcolor: "rgba(245, 173, 24, 0.1)",
                },
              }}
            >
              <RefreshIcon />
            </IconButton>
          </Stack>
        </Box>

        {/* Log Display */}
        <Box
          ref={logRef}
          onScroll={handleScroll}
          sx={{
            flex: 1,
            overflowY: "auto",
            position: "relative",
            p: 2,
            fontFamily:
              'ui-monospace, SFMono-Regular, Menlo, Consolas, "Liberation Mono", monospace',
            fontSize: 13,
            lineHeight: 1.6,
            whiteSpace: "pre-wrap",
            wordBreak: "break-word",
            backgroundColor: "#0f0a0e",
            color: "text.primary",
          }}
        >
          {(() => {
            if (filtered.length === 0 && lines.length === 0) {
              return (
                <Typography
                  sx={{
                    color: "text.secondary",
                    textAlign: "center",
                    mt: 4,
                    fontStyle: "italic",
                  }}
                >
                  Waiting for logs...
                </Typography>
              );
            } else if (filtered.length === 0) {
              return (
                <Typography
                  sx={{
                    color: "text.secondary",
                    textAlign: "center",
                    mt: 4,
                    fontStyle: "italic",
                  }}
                >
                  No logs match your filter
                </Typography>
              );
            } else {
              return filtered.map((l, i) => (
                <Typography
                  key={l + "_" + i}
                  component="div"
                  sx={{
                    fontFamily: "inherit",
                    fontSize: "inherit",
                    "&:hover": {
                      bgcolor: "rgba(158, 28, 96, 0.1)",
                    },
                  }}
                >
                  {l}
                </Typography>
              ));
            }
          })()}

          {/* Scroll to Bottom Button */}
          {showScrollBtn && (
            <IconButton
              onClick={scrollToBottom}
              sx={{
                position: "absolute",
                bottom: 16,
                right: 16,
                bgcolor: "#9E1C60",
                color: "#fff",
                boxShadow: "0 4px 12px rgba(158, 28, 96, 0.4)",
                "&:hover": {
                  bgcolor: "#811844",
                },
              }}
              size="small"
            >
              <KeyboardArrowDownIcon />
            </IconButton>
          )}
        </Box>
      </Paper>
    </Container>
  );
}
