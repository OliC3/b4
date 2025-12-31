import { Grid, FormControlLabel, Switch, Typography, Box } from "@mui/material";
import { DnsIcon } from "@b4.icons";
import { B4SetConfig, WindowMode, DesyncMode } from "@models/config";
import {
  B4Slider,
  B4Select,
  B4TextField,
  B4Section,
  B4Alert,
  B4FormHeader,
  B4PlusButton,
  B4ChipList,
} from "@b4.elements";
import { useState } from "react";

interface TcpSettingsProps {
  config: B4SetConfig;
  main: B4SetConfig;
  onChange: (
    field: string,
    value: string | number | boolean | number[]
  ) => void;
}

const desyncModeOptions: { label: string; value: DesyncMode }[] = [
  { label: "Disabled", value: "off" },
  { label: "RST Packets", value: "rst" },
  { label: "FIN Packets", value: "fin" },
  { label: "ACK Packets", value: "ack" },
  { label: "Combo (RST + FIN)", value: "combo" },
  { label: "Full (RST + FIN + ACK)", value: "full" },
];

const desyncModeDescriptions: Record<DesyncMode, string> = {
  off: "No desynchronization - packets sent normally",
  rst: "Inject fake RST packets with bad checksums to disrupt DPI state tracking",
  fin: "Inject fake FIN packets with past sequence numbers to confuse connection state",
  ack: "Inject fake ACK packets with random future sequence/ack numbers",
  combo: "Send RST + FIN + ACK sequence for stronger desync effect",
  full: "Full attack: fake SYN, overlapping RSTs, PSH, and URG packets",
};

const windowModeOptions: { label: string; value: WindowMode }[] = [
  { label: "Disabled", value: "off" },
  { label: "Zero Window", value: "zero" },
  { label: "Random Window", value: "random" },
  { label: "Oscillate", value: "oscillate" },
  { label: "Escalate", value: "escalate" },
];

const windowModeDescriptions: Record<WindowMode, string> = {
  off: "No window manipulation - use actual TCP window",
  zero: "Send fake packets: first with window=0, then window=65535",
  random: "Send 3-5 fake packets with random window sizes from your list",
  oscillate: "Cycle through your custom window values sequentially",
  escalate: "Gradually increase: 0 → 100 → 500 → 1460 → 8192 → 32768 → 65535",
};

export const TcpSettings = ({ config, main, onChange }: TcpSettingsProps) => {
  const [newWinValue, setNewWinValue] = useState("");

  const winValues = config.tcp.win.values || [0, 1460, 8192, 65535];
  const showWinValues = ["oscillate", "random"].includes(config.tcp.win.mode);
  const isDesyncEnabled = config.tcp.desync.mode !== "off";

  const handleAddWinValue = () => {
    const val = parseInt(newWinValue, 10);
    if (!isNaN(val) && val >= 0 && val <= 65535 && !winValues.includes(val)) {
      onChange(
        "tcp.win_values",
        [...winValues, val].sort((a, b) => a - b)
      );
      setNewWinValue("");
    }
  };

  const handleRemoveWinValue = (val: number) => {
    onChange(
      "tcp.win_values",
      winValues.filter((v) => v !== val)
    );
  };

  return (
    <B4Section
      title="TCP Configuration"
      description="Configure TCP packet handling and DPI bypass techniques"
      icon={<DnsIcon />}
    >
      {/* Basic TCP Settings */}
      <Grid container spacing={3}>
        <Grid size={{ xs: 12, md: 6 }}>
          <B4Slider
            label="Connection Bytes Limit"
            value={config.tcp.conn_bytes_limit}
            onChange={(value: number) =>
              onChange("tcp.conn_bytes_limit", value)
            }
            min={1}
            max={main.id === config.id ? 100 : main.tcp.conn_bytes_limit}
            step={1}
            helperText={
              main.id === config.id
                ? "Main set limit (changing requires service restart to take effect)"
                : `Max: ${main.tcp.conn_bytes_limit} (limited by main set)`
            }
          />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <B4Slider
            label="Segment 2 Delay"
            value={config.tcp.seg2delay}
            onChange={(value: number) => onChange("tcp.seg2delay", value)}
            min={0}
            max={1000}
            step={10}
            valueSuffix=" ms"
            helperText="Delay between TCP segments (helps with timing-based DPI)"
          />
        </Grid>

        {/* SACK and SYN Fake */}
        <Grid size={{ xs: 12, md: 6 }}>
          <FormControlLabel
            control={
              <Switch
                checked={config.tcp.drop_sack || false}
                onChange={(e) => onChange("tcp.drop_sack", e.target.checked)}
                color="primary"
              />
            }
            label={
              <Box>
                <Typography variant="body1" fontWeight={500}>
                  Drop SACK Options
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  Strip Selective Acknowledgment from TCP headers to confuse
                  stateful DPI
                </Typography>
              </Box>
            }
          />
        </Grid>

        <Grid size={{ xs: 12, md: 6 }}>
          <FormControlLabel
            control={
              <Switch
                checked={config.tcp.syn_fake || false}
                onChange={(e) => onChange("tcp.syn_fake", e.target.checked)}
                color="primary"
              />
            }
            label={
              <Box>
                <Typography variant="body1" fontWeight={500}>
                  SYN Fake Packets
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  Send fake SYN packets during handshake (aggressive technique)
                </Typography>
              </Box>
            }
          />
        </Grid>

        {config.tcp.syn_fake && (
          <>
            <Grid size={{ xs: 12, md: 6 }}>
              <B4Slider
                label="SYN Fake Payload Length"
                value={config.tcp.syn_fake_len || 0}
                onChange={(value: number) =>
                  onChange("tcp.syn_fake_len", value)
                }
                min={0}
                max={1200}
                step={64}
                valueSuffix=" bytes"
                helperText="0 = header only, >0 = add fake TLS payload"
              />
            </Grid>
            <Grid size={{ xs: 12, md: 6 }}>
              <B4Slider
                label="SYN Fake TTL"
                value={config.tcp.syn_ttl || 0}
                onChange={(value: number) => onChange("tcp.syn_ttl", value)}
                min={1}
                max={100}
                step={1}
                valueSuffix=" ms"
                helperText="TTL value for SYN fake packets (default 3 if unset)"
              />
            </Grid>
          </>
        )}
      </Grid>

      {/* TCP Window Configuration */}
      <B4FormHeader label="TCP Window Manipulation" />
      <Grid container spacing={3}>
        <B4Alert>
          Window manipulation sends fake ACK packets with modified TCP window
          sizes before your real packet. These fakes use low TTL so they expire
          before reaching the server but confuse middlebox DPI.
        </B4Alert>

        <Grid size={{ xs: 12, md: 6 }}>
          <B4Select
            label="Window Mode"
            value={config.tcp.win.mode}
            options={windowModeOptions}
            onChange={(e) => onChange("tcp.win.mode", e.target.value as string)}
            helperText={windowModeDescriptions[config.tcp.win.mode]}
          />
        </Grid>

        {showWinValues && (
          <Grid size={{ xs: 12 }}>
            <Typography variant="subtitle2" gutterBottom>
              Custom Window Values
            </Typography>
            <Typography variant="caption" color="text.secondary" gutterBottom>
              {config.tcp.win.mode === "oscillate"
                ? "Packets will cycle through these values in order"
                : "Random values will be picked from this list"}
            </Typography>

            <Grid container spacing={2} alignItems="center">
              <Grid size={{ xs: 12, md: 6 }}>
                <Box
                  sx={{
                    display: "flex",
                    gap: 2,
                    mt: 1,
                    alignItems: "center",
                  }}
                >
                  <B4TextField
                    label="Add Value (0-65535)"
                    value={newWinValue}
                    onChange={(e) => setNewWinValue(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") {
                        e.preventDefault();
                        handleAddWinValue();
                      }
                    }}
                    type="number"
                  />

                  <B4PlusButton
                    onClick={handleAddWinValue}
                    disabled={!newWinValue}
                  />
                </Box>
              </Grid>
              <Grid size={{ xs: 12, md: 6 }}>
                <B4ChipList
                  items={winValues}
                  getKey={(v) => v}
                  getLabel={(v) => v.toLocaleString()}
                  onDelete={handleRemoveWinValue}
                  emptyMessage="No values configured - defaults will be used"
                  showEmpty
                />
              </Grid>
            </Grid>
          </Grid>
        )}
      </Grid>

      {/* TCP Desync Configuration */}
      <B4FormHeader label="TCP Desync Attack" />
      <Grid container spacing={3}>
        <B4Alert>
          Desync attacks inject fake TCP control packets (RST/FIN/ACK) with
          corrupted checksums and low TTL. These packets confuse stateful DPI
          systems but are discarded by the real server.
        </B4Alert>
        <Grid size={{ xs: 12, md: 4 }}>
          <B4Select
            label="Desync Mode"
            value={config.tcp.desync.mode}
            options={desyncModeOptions}
            onChange={(e) =>
              onChange("tcp.desync.mode", e.target.value as string)
            }
            helperText={desyncModeDescriptions[config.tcp.desync.mode]}
          />
        </Grid>

        <Grid size={{ xs: 12, md: 4 }}>
          <B4Slider
            label="Desync TTL"
            value={config.tcp.desync.ttl}
            onChange={(value: number) => onChange("tcp.desync.ttl", value)}
            min={1}
            max={50}
            step={1}
            disabled={!isDesyncEnabled}
            helperText={
              isDesyncEnabled
                ? "Low TTL ensures packets expire before reaching server"
                : "Enable desync mode first"
            }
          />
        </Grid>

        <Grid size={{ xs: 12, md: 4 }}>
          <B4Slider
            label="Desync Packet Count"
            value={config.tcp.desync.count}
            onChange={(value: number) => onChange("tcp.desync.count", value)}
            min={1}
            max={20}
            step={1}
            disabled={!isDesyncEnabled}
            helperText={
              isDesyncEnabled
                ? "Number of fake packets per desync attack"
                : "Enable desync mode first"
            }
          />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <FormControlLabel
            control={
              <Switch
                checked={config.tcp.desync.post_desync || false}
                onChange={(e) =>
                  onChange("tcp.desync.post_desync", e.target.checked)
                }
                color="primary"
              />
            }
            label={
              <Box>
                <Typography variant="body1" fontWeight={500}>
                  Post-ClientHello RST
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  Send fake RST after ClientHello to evict connection from DPI
                  tracking table
                </Typography>
              </Box>
            }
          />
        </Grid>
      </Grid>
    </B4Section>
  );
};
