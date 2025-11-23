import React from "react";
import { Grid, Alert, Divider, Chip, Typography } from "@mui/material";
import { CallSplit as CallSplitIcon } from "@mui/icons-material";
import SettingSection from "@molecules/common/B4Section";
import SettingSelect from "@atoms/common/B4Select";
import SettingSwitch from "@atoms/common/B4Switch";
import B4TextField from "@atoms/common/B4TextField";
import B4Slider from "@atoms/common/B4Slider";
import { B4SetConfig, FragmentationStrategy } from "@models/Config";

interface FragmentationSettingsProps {
  config: B4SetConfig;
  onChange: (field: string, value: string | boolean | number) => void;
}

const fragmentationOptions: { label: string; value: FragmentationStrategy }[] =
  [
    { label: "TCP Segmentation", value: "tcp" },
    { label: "IP Fragmentation", value: "ip" },
    { label: "TLS Record Splitting", value: "tls" },
    { label: "OOB (Out-of-Band)", value: "oob" },
    { label: "No Fragmentation", value: "none" },
  ];

const strategyDescriptions = {
  tcp: "Splits packets at TCP layer - works with most servers, no MTU issues",
  ip: "Fragments at IP layer - bypasses some TCP-aware DPI but may cause MTU problems",
  tls: "Splits TLS ClientHello into multiple TLS records - bypasses DPI expecting single-record handshakes",
  oob: "Sends data with URG flag (Out-of-Band) - confuses stateful DPI inspection",
  none: "No fragmentation applied - packets sent as-is",
};

export const FragmentationSettings: React.FC<FragmentationSettingsProps> = ({
  config,
  onChange,
}) => {
  const strategy = config.fragmentation.strategy;
  const isTcpOrIp = strategy === "tcp" || strategy === "ip";
  const isOob = strategy === "oob";
  const isTls = strategy === "tls";
  const isActive = strategy !== "none";

  return (
    <SettingSection
      title="Fragmentation Strategy"
      description="Configure how packets are split to evade DPI detection"
      icon={<CallSplitIcon />}
    >
      <Grid container spacing={3}>
        {/* Strategy Selection */}
        <Grid size={{ xs: 12, md: 6 }}>
          <SettingSelect
            label="Fragmentation Method"
            value={strategy}
            options={fragmentationOptions}
            onChange={(e) =>
              onChange("fragmentation.strategy", e.target.value as string)
            }
            helperText={strategyDescriptions[strategy]}
          />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <SettingSwitch
            label="Reverse Fragment Order"
            checked={config.fragmentation.reverse_order}
            onChange={(checked) =>
              onChange("fragmentation.reverse_order", checked)
            }
            description="Send fragments in reverse order (applies to TCP/IP/TLS strategies)"
          />
        </Grid>
        {isActive && (
          <>
            <Grid size={{ xs: 12 }}>
              <Alert severity="info">
                <Typography variant="body2">
                  {strategy === "tcp" && (
                    <>
                      <strong>TCP Segmentation:</strong> Splits packets at TCP
                      layer. Most compatible, works with firewalls and NAT.
                    </>
                  )}
                  {strategy === "ip" && (
                    <>
                      <strong>IP Fragmentation:</strong> Splits at IP layer.
                      Bypasses TCP-aware DPI but may fail with strict MTU
                      limits.
                    </>
                  )}
                  {strategy === "tls" && (
                    <>
                      <strong>TLS Record Splitting:</strong> Splits ClientHello
                      into multiple TLS records. Highly effective against DPI
                      expecting single-record handshakes.
                    </>
                  )}
                  {strategy === "oob" && (
                    <>
                      <strong>OOB (Out-of-Band):</strong> Sends extra byte with
                      URG flag. Highly effective against stateful DPI, may
                      confuse older middleboxes.
                    </>
                  )}
                </Typography>
              </Alert>
            </Grid>
          </>
        )}

        {/* TCP/IP Fragmentation Settings */}
        {isTcpOrIp && (
          <>
            <Grid size={{ xs: 12 }}>
              <Divider sx={{ my: 2 }}>
                <Chip label="Split Configuration" size="small" />
              </Divider>
            </Grid>

            <Grid size={{ xs: 12, md: 6 }}>
              <B4Slider
                label="SNI Split Position"
                value={config.fragmentation.sni_position}
                onChange={(value) =>
                  onChange("fragmentation.sni_position", value)
                }
                min={0}
                max={10}
                step={1}
                helperText="Where to split SNI field (0=first byte)"
              />
            </Grid>

            <Grid size={{ xs: 12, md: 6 }}>
              <SettingSwitch
                label="Split in Middle of SNI"
                checked={config.fragmentation.middle_sni}
                onChange={(checked) =>
                  onChange("fragmentation.middle_sni", checked)
                }
                description="Split at SNI midpoint instead of start"
              />
            </Grid>
          </>
        )}

        {/* OOB Settings */}
        {isOob && (
          <>
            <Grid size={{ xs: 12 }}>
              <Divider sx={{ my: 2 }}>
                <Chip label="OOB Configuration" size="small" />
              </Divider>
            </Grid>

            <Grid size={{ xs: 12, md: 4 }}>
              <B4Slider
                label="OOB Split Position"
                value={config.fragmentation.oob_position || 1}
                onChange={(value) =>
                  onChange("fragmentation.oob_position", value)
                }
                min={1}
                max={10}
                step={1}
                helperText="Bytes before OOB insertion"
              />
            </Grid>

            <Grid size={{ xs: 12, md: 4 }}>
              <B4TextField
                label="OOB Character"
                value={String.fromCharCode(
                  config.fragmentation.oob_char || 120
                )}
                onChange={(e) => {
                  const char = e.target.value.slice(0, 1);
                  onChange(
                    "fragmentation.oob_char",
                    char ? char.charCodeAt(0) : 120
                  );
                }}
                placeholder="x"
                helperText="Byte sent with URG flag"
                inputProps={{ maxLength: 1 }}
              />
            </Grid>
          </>
        )}

        {/* TLS Record Splitting Settings */}
        {isTls && (
          <>
            <Grid size={{ xs: 12 }}>
              <Divider sx={{ my: 2 }}>
                <Chip label="TLS Record Configuration" size="small" />
              </Divider>
            </Grid>

            <Grid size={{ xs: 12, md: 6 }}>
              <B4Slider
                label="TLS Record Split Position"
                value={config.fragmentation.tlsrec_pos || 1}
                onChange={(value) =>
                  onChange("fragmentation.tlsrec_pos", value)
                }
                min={1}
                max={100}
                step={1}
                helperText="Where to split TLS ClientHello record (bytes into handshake data)"
              />
            </Grid>
          </>
        )}

        {/* None Strategy Info */}
        {strategy === "none" && (
          <Grid size={{ xs: 12 }}>
            <Alert severity="warning">
              <Typography variant="body2">
                <strong>No Fragmentation:</strong> Packets sent unmodified. Only
                fake packets (if enabled in Faking tab) will be used for DPI
                bypass.
              </Typography>
            </Alert>
          </Grid>
        )}
      </Grid>
    </SettingSection>
  );
};
