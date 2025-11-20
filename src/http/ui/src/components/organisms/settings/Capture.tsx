import React, { useState, useEffect } from "react";
import {
  Grid,
  Stack,
  Alert,
  Typography,
  Button,
  Box,
  Chip,
  List,
  ListItem,
  ListItemText,
  IconButton,
  Paper,
  CircularProgress,
  MenuItem,
  Divider,
  Tooltip,
} from "@mui/material";
import {
  CameraAlt as CaptureIcon,
  PlayArrow as StartIcon,
  Stop as StopIcon,
  Download as DownloadIcon,
  ContentCopy as CopyIcon,
  Info as InfoIcon,
} from "@mui/icons-material";
import B4Section from "@molecules/common/B4Section";
import B4TextField from "@atoms/common/B4TextField";
import B4Slider from "@atoms/common/B4Slider";
import { B4Dialog } from "@molecules/common/B4Dialog";
import { colors, button_primary, button_secondary, radius } from "@design";

interface CaptureSession {
  id: string;
  domain: string;
  protocol: string;
  max_packets: number;
  count: number;
  active: boolean;
  start_time: string;
  captures: CaptureData[];
}

interface CaptureData {
  protocol: string;
  domain: string;
  timestamp: string;
  size: number;
  filepath: string;
  hex_data?: string;
}

export const CaptureSettings: React.FC = () => {
  const [sessions, setSessions] = useState<CaptureSession[]>([]);
  const [loading, setLoading] = useState(false);
  const [newSession, setNewSession] = useState({
    domain: "*",
    protocol: "both",
    max_packets: 5,
  });

  const [hexDialog, setHexDialog] = useState<{
    open: boolean;
    capture: CaptureData | null;
  }>({ open: false, capture: null });

  useEffect(() => {
    void loadSessions();
    const interval = setInterval(() => {
      void loadSessions();
    }, 2000); // Auto-refresh active sessions
    return () => clearInterval(interval);
  }, []);

  const loadSessions = async () => {
    try {
      const response = await fetch("/api/capture/status");
      if (response.ok) {
        const data = (await response.json()) as CaptureSession[];
        setSessions(data);
      }
    } catch (error) {
      console.error("Failed to load capture sessions:", error);
    }
  };

  const startCapture = async () => {
    setLoading(true);
    try {
      const response = await fetch("/api/capture/start", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(newSession),
      });

      if (response.ok) {
        await loadSessions();
        setNewSession({ domain: "*", protocol: "both", max_packets: 5 });
      }
    } catch (error) {
      console.error("Failed to start capture:", error);
    } finally {
      setLoading(false);
    }
  };

  const stopCapture = async (sessionId: string) => {
    try {
      await fetch(`/api/capture/stop?id=${sessionId}`, {
        method: "POST",
      });
      await loadSessions();
    } catch (error) {
      console.error("Failed to stop capture:", error);
    }
  };

  const downloadCapture = (capture: CaptureData) => {
    const date = new Date(capture.timestamp);
    const dateStr = date.toISOString().split("T")[0].replace(/-/g, "");
    const timeStr = date.toTimeString().split(" ")[0].replace(/:/g, "");

    const customName = `${capture.protocol}_${capture.domain.replace(
      /\./g,
      "_"
    )}_${dateStr}_${timeStr}.bin`;

    const url = `/api/capture/download?file=${encodeURIComponent(
      capture.filepath
    )}&name=${encodeURIComponent(customName)}`;

    // Create a temporary link to trigger download
    const link = document.createElement("a");
    link.href = url;
    link.download = customName; // Also hint the browser
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };
  const copyHexToClipboard = (hexData: string) => {
    void navigator.clipboard.writeText(hexData);
  };

  const activeSessions = sessions.filter((s) => s.active);
  const completedSessions = sessions.filter((s) => !s.active);

  return (
    <Stack spacing={3}>
      {/* Info Alert */}
      <Alert severity="info" icon={<InfoIcon />}>
        <Typography variant="subtitle2" gutterBottom>
          Capture real protocol handshakes from live traffic for custom payload
          generation.
        </Typography>
        <Typography variant="caption" color="text.secondary">
          Captured payloads can be used in the Faking configuration as custom
          payloads.
        </Typography>
      </Alert>

      {/* New Capture Section */}
      <B4Section
        title="Start New Capture"
        description="Capture TLS ClientHello or QUIC Initial packets from specific domains"
        icon={<CaptureIcon />}
      >
        <Grid container spacing={2}>
          <Grid size={{ xs: 12, md: 4 }}>
            <B4TextField
              label="Domain Filter"
              value={newSession.domain}
              onChange={(e) =>
                setNewSession({ ...newSession, domain: e.target.value })
              }
              placeholder="*.youtube.com or * for all"
              helperText="Use * to capture from any domain"
            />
          </Grid>

          <Grid size={{ xs: 12, md: 3 }}>
            <B4TextField
              select
              label="Protocol"
              value={newSession.protocol}
              onChange={(e) =>
                setNewSession({ ...newSession, protocol: e.target.value })
              }
              helperText="Which protocols to capture"
            >
              <MenuItem value="both">Both TLS & QUIC</MenuItem>
              <MenuItem value="tls">TLS Only</MenuItem>
              <MenuItem value="quic">QUIC Only</MenuItem>
            </B4TextField>
          </Grid>

          <Grid size={{ xs: 12, md: 3 }}>
            <B4Slider
              label="Max Packets"
              value={newSession.max_packets}
              onChange={(value) =>
                setNewSession({ ...newSession, max_packets: value })
              }
              min={1}
              max={20}
              step={1}
              helperText="Stop after capturing N packets"
            />
          </Grid>

          <Grid size={{ xs: 12, md: 2 }}>
            <Button
              fullWidth
              variant="contained"
              startIcon={
                loading ? <CircularProgress size={16} /> : <StartIcon />
              }
              onClick={() => void startCapture()}
              disabled={loading || activeSessions.length >= 3}
              sx={{ ...button_primary, height: 56 }}
            >
              {loading ? "Starting..." : "Start Capture"}
            </Button>
          </Grid>
        </Grid>

        {activeSessions.length >= 3 && (
          <Alert severity="warning" sx={{ mt: 2 }}>
            Maximum 3 concurrent capture sessions allowed
          </Alert>
        )}
      </B4Section>

      {/* Active Sessions */}
      {activeSessions.length > 0 && (
        <B4Section
          title="Active Capture Sessions"
          description="Currently capturing packets"
          icon={<CaptureIcon />}
        >
          <Stack spacing={2}>
            {activeSessions.map((session) => (
              <Paper
                key={session.id}
                elevation={1}
                sx={{
                  p: 2,
                  border: `1px solid ${colors.secondary}`,
                  borderRadius: radius.md,
                  position: "relative",
                  overflow: "hidden",
                }}
              >
                <Box
                  sx={{
                    position: "absolute",
                    top: 0,
                    left: 0,
                    right: 0,
                    height: 3,
                    bgcolor: colors.secondary,
                    animation: "pulse 2s ease-in-out infinite",
                    "@keyframes pulse": {
                      "0%, 100%": { opacity: 1 },
                      "50%": { opacity: 0.3 },
                    },
                  }}
                />

                <Grid container alignItems="center" spacing={2}>
                  <Grid size={{ xs: 12, md: 3 }}>
                    <Typography variant="body2" color="text.secondary">
                      Domain
                    </Typography>
                    <Typography variant="body1" fontWeight={500}>
                      {session.domain}
                    </Typography>
                  </Grid>

                  <Grid size={{ xs: 12, md: 2 }}>
                    <Typography variant="body2" color="text.secondary">
                      Protocol
                    </Typography>
                    <Chip
                      label={session.protocol.toUpperCase()}
                      size="small"
                      sx={{
                        bgcolor: colors.accent.secondary,
                        color: colors.secondary,
                      }}
                    />
                  </Grid>

                  <Grid size={{ xs: 12, md: 3 }}>
                    <Typography variant="body2" color="text.secondary">
                      Progress
                    </Typography>
                    <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                      <Typography variant="body1">
                        {session.count} / {session.max_packets}
                      </Typography>
                      <Box
                        sx={{
                          flex: 1,
                          height: 6,
                          borderRadius: 3,
                          bgcolor: colors.background.dark,
                          overflow: "hidden",
                        }}
                      >
                        <Box
                          sx={{
                            width: `${
                              (session.count / session.max_packets) * 100
                            }%`,
                            height: "100%",
                            bgcolor: colors.secondary,
                            borderRadius: 3,
                            transition: "width 0.3s ease",
                          }}
                        />
                      </Box>
                    </Box>
                  </Grid>

                  <Grid size={{ xs: 12, md: 2 }}>
                    <Typography variant="body2" color="text.secondary">
                      Started
                    </Typography>
                    <Typography variant="caption">
                      {new Date(session.start_time).toLocaleTimeString()}
                    </Typography>
                  </Grid>

                  <Grid size={{ xs: 12, md: 2 }}>
                    <Button
                      variant="outlined"
                      startIcon={<StopIcon />}
                      onClick={() => void stopCapture(session.id)}
                      sx={{ ...button_secondary }}
                    >
                      Stop
                    </Button>
                  </Grid>
                </Grid>

                {/* Live captures preview */}
                {session.captures.length > 0 && (
                  <Box
                    sx={{
                      mt: 2,
                      pt: 2,
                      borderTop: `1px solid ${colors.border.light}`,
                    }}
                  >
                    <Typography variant="caption" color="text.secondary">
                      Recent captures:
                    </Typography>
                    <Stack direction="row" spacing={1} sx={{ mt: 1 }}>
                      {session.captures.slice(-3).map((capture, idx) => (
                        <Chip
                          key={idx}
                          label={`${capture.protocol.toUpperCase()} - ${
                            capture.domain
                          }`}
                          size="small"
                          variant="outlined"
                        />
                      ))}
                    </Stack>
                  </Box>
                )}
              </Paper>
            ))}
          </Stack>
        </B4Section>
      )}

      {/* Completed Sessions */}
      {completedSessions.length > 0 && (
        <B4Section
          title="Captured Payloads"
          description="Ready to use in your faking configuration"
          icon={<DownloadIcon />}
        >
          <List>
            {completedSessions.map((session) => (
              <Box key={session.id}>
                <ListItem sx={{ px: 0 }}>
                  <ListItemText
                    primary={
                      <Typography variant="subtitle1" fontWeight={500}>
                        {session.domain} - {session.protocol.toUpperCase()}
                      </Typography>
                    }
                    secondary={`${session.count} packets captured • ${new Date(
                      session.start_time
                    ).toLocaleString()}`}
                  />
                </ListItem>

                <Grid container spacing={1} sx={{ ml: 2, mb: 2 }}>
                  {session.captures.map((capture, idx) => (
                    <Grid size={{ xs: 12, sm: 6, md: 4 }} key={idx}>
                      <Paper
                        elevation={0}
                        sx={{
                          p: 1.5,
                          border: `1px solid ${colors.border.default}`,
                          borderRadius: radius.sm,
                          cursor: "pointer",
                          transition: "all 0.2s",
                          "&:hover": {
                            borderColor: colors.secondary,
                            transform: "translateY(-2px)",
                          },
                        }}
                        onClick={() => setHexDialog({ open: true, capture })}
                      >
                        <Stack spacing={0.5}>
                          <Typography variant="caption" fontWeight={600}>
                            {capture.protocol.toUpperCase()} • {capture.domain}
                          </Typography>
                          <Typography variant="caption" color="text.secondary">
                            {capture.size} bytes
                          </Typography>
                          <Stack direction="row" spacing={0.5}>
                            <Tooltip title="Download binary">
                              <IconButton
                                size="small"
                                onClick={(e) => {
                                  e.stopPropagation();
                                  downloadCapture(capture);
                                }}
                              >
                                <DownloadIcon fontSize="small" />
                              </IconButton>
                            </Tooltip>
                            <Tooltip title="Copy as hex">
                              <IconButton
                                size="small"
                                onClick={(e) => {
                                  e.stopPropagation();
                                  if (capture.hex_data) {
                                    copyHexToClipboard(capture.hex_data);
                                  }
                                }}
                              >
                                <CopyIcon fontSize="small" />
                              </IconButton>
                            </Tooltip>
                          </Stack>
                        </Stack>
                      </Paper>
                    </Grid>
                  ))}
                </Grid>
                <Divider />
              </Box>
            ))}
          </List>
        </B4Section>
      )}

      {/* Hex Data Dialog */}
      <B4Dialog
        title="Payload Data"
        subtitle="Copy this hex string to use as custom payload"
        icon={<CaptureIcon />}
        open={hexDialog.open}
        onClose={() => setHexDialog({ open: false, capture: null })}
        maxWidth="md"
        fullWidth
        actions={
          <Button
            variant="contained"
            onClick={() => {
              if (hexDialog.capture?.hex_data) {
                copyHexToClipboard(hexDialog.capture.hex_data);
              }
              setHexDialog({ open: false, capture: null });
            }}
            sx={{ ...button_primary }}
          >
            Copy & Close
          </Button>
        }
      >
        {hexDialog.capture && (
          <Stack spacing={2}>
            <Alert severity="info">
              Use this in Faking Settings → Custom Payload field
            </Alert>
            <Box
              sx={{
                p: 2,
                bgcolor: colors.background.dark,
                borderRadius: radius.sm,
                fontFamily: "monospace",
                fontSize: "0.8rem",
                wordBreak: "break-all",
                maxHeight: 400,
                overflow: "auto",
              }}
            >
              {hexDialog.capture.hex_data}
            </Box>
          </Stack>
        )}
      </B4Dialog>
    </Stack>
  );
};
