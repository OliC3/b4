import React, { useState } from "react";
import {
  Grid,
  Alert,
  Divider,
  Chip,
  Typography,
  Box,
  Collapse,
  Button,
} from "@mui/material";
import {
  CallSplit as CallSplitIcon,
  ExpandMore as ExpandIcon,
  ExpandLess as CollapseIcon,
} from "@mui/icons-material";
import SettingSection from "@molecules/common/B4Section";
import SettingSelect from "@atoms/common/B4Select";
import SettingSwitch from "@atoms/common/B4Switch";
import B4Slider from "@atoms/common/B4Slider";
import { B4SetConfig, FragmentationStrategy } from "@models/Config";
import { colors } from "@design";
import { ComboSettings } from "./frags/Combo";
import { DisorderSettings } from "./frags/Disorder";

interface FragmentationSettingsProps {
  config: B4SetConfig;
  onChange: (field: string, value: string | boolean | number) => void;
}

const fragmentationOptions: { label: string; value: FragmentationStrategy }[] =
  [
    { label: "Combo", value: "combo" },
    { label: "Hybrid", value: "hybrid" },
    { label: "Disorder", value: "disorder" },
    { label: "Overlap", value: "overlap" },
    { label: "Extension Split", value: "extsplit" },
    { label: "First-Byte Desync", value: "firstbyte" },
    { label: "TCP Segmentation", value: "tcp" },
    { label: "IP Fragmentation", value: "ip" },
    { label: "TLS Record Splitting", value: "tls" },
    { label: "OOB (Out-of-Band)", value: "oob" },
    { label: "Disabled", value: "none" },
  ];

export const FragmentationSettings = ({
  config,
  onChange,
}: FragmentationSettingsProps) => {
  const [showAdvanced, setShowAdvanced] = useState(false);

  const strategy = config.fragmentation.strategy;
  const isTcpOrIp = strategy === "tcp" || strategy === "ip";
  const isOob = strategy === "oob";
  const isTls = strategy === "tls";
  const isActive = strategy !== "none";

  // Determine effective split mode for display
  const getSplitModeDescription = () => {
    if (config.fragmentation.middle_sni) {
      if (config.fragmentation.sni_position > 0) {
        return "3 segments: split at fixed position AND middle of SNI";
      }
      return "2 segments: split at middle of SNI hostname";
    }
    return `2 segments: split at byte ${config.fragmentation.sni_position} of TLS payload`;
  };

  return (
    <SettingSection
      title="Fragmentation Strategy"
      description="Split packets to evade DPI pattern matching"
      icon={<CallSplitIcon />}
    >
      <Grid container spacing={3}>
        {/* Strategy Selection */}
        <Grid size={{ xs: 12, md: 6 }}>
          <SettingSelect
            label="Method"
            value={strategy}
            options={fragmentationOptions}
            onChange={(e) =>
              onChange("fragmentation.strategy", e.target.value as string)
            }
          />
        </Grid>

        <Grid size={{ xs: 12, md: 6 }}>
          <SettingSwitch
            label="Reverse Fragment Order"
            checked={config.fragmentation.reverse_order}
            onChange={(checked: boolean) =>
              onChange("fragmentation.reverse_order", checked)
            }
            description="Send second fragment first"
          />
        </Grid>

        {/* TCP/IP: Simplified SNI Split */}
        {isTcpOrIp && (
          <>
            <Grid size={{ xs: 12 }}>
              <Divider sx={{ my: 1 }}>
                <Chip label="Where to Split" size="small" />
              </Divider>
            </Grid>

            <Grid size={{ xs: 12 }}>
              <SettingSwitch
                label="Smart SNI Split"
                checked={config.fragmentation.middle_sni}
                onChange={(checked: boolean) =>
                  onChange("fragmentation.middle_sni", checked)
                }
                description="Automatically split in the middle of the SNI hostname (recommended)"
              />
            </Grid>

            {/* Visual explanation */}
            <Grid size={{ xs: 12 }}>
              <Box
                sx={{
                  p: 2,
                  bgcolor: colors.background.paper,
                  borderRadius: 1,
                  border: `1px solid ${colors.border.default}`,
                }}
              >
                <Typography
                  variant="caption"
                  color="text.secondary"
                  component="div"
                  sx={{ mb: 1 }}
                >
                  TCP PACKET STRUCTURE EXAMPLE
                </Typography>
                <Box
                  sx={{
                    display: "flex",
                    gap: 0.5,
                    fontFamily: "monospace",
                    fontSize: "0.75rem",
                  }}
                >
                  <Box
                    sx={{
                      p: 1,
                      bgcolor: colors.accent.primary,
                      borderRadius: 0.5,
                      textAlign: "center",
                      minWidth: 60,
                    }}
                  >
                    TLS Header
                  </Box>
                  <Box
                    sx={{
                      p: 1,
                      bgcolor: colors.accent.secondary,
                      borderRadius: 0.5,
                      textAlign: "center",
                      flex: 1,
                      position: "relative",
                    }}
                  >
                    {/* Fixed position split line */}
                    {config.fragmentation.sni_position > 0 && (
                      <Box
                        component="span"
                        sx={{
                          position: "absolute",
                          left: "20%",
                          top: 0,
                          bottom: 0,
                          width: 2,
                          bgcolor: colors.tertiary,
                          transform: "translateX(-50%)",
                        }}
                      />
                    )}
                    {/* Middle SNI split line */}
                    {config.fragmentation.middle_sni && (
                      <Box
                        component="span"
                        sx={{
                          position: "absolute",
                          left: "50%",
                          top: 0,
                          bottom: 0,
                          width: 2,
                          bgcolor: colors.quaternary,
                          transform: "translateX(-50%)",
                        }}
                      />
                    )}
                    SNI: youtube.com
                  </Box>
                  <Box
                    sx={{
                      p: 1,
                      bgcolor: colors.accent.primary,
                      borderRadius: 0.5,
                      textAlign: "center",
                      minWidth: 80,
                    }}
                  >
                    Extensions...
                  </Box>
                </Box>
                <Typography
                  variant="caption"
                  color="text.secondary"
                  sx={{ mt: 1, display: "block" }}
                >
                  {getSplitModeDescription()}
                </Typography>
              </Box>
            </Grid>

            {/* Advanced toggle */}
            <Grid size={{ xs: 12 }}>
              <Button
                size="small"
                onClick={() => setShowAdvanced(!showAdvanced)}
                endIcon={showAdvanced ? <CollapseIcon /> : <ExpandIcon />}
                sx={{ color: colors.text.secondary, textTransform: "none" }}
              >
                {showAdvanced ? "Hide" : "Show"} manual position control
              </Button>
            </Grid>

            <Grid size={{ xs: 12 }}>
              <Collapse in={showAdvanced}>
                <Box
                  sx={{
                    p: 2,
                    bgcolor: colors.background.dark,
                    borderRadius: 1,
                  }}
                >
                  <Typography
                    variant="caption"
                    color="warning.main"
                    gutterBottom
                    component="div"
                  >
                    Manual override — use if Smart SNI Split doesn't work for
                    your ISP
                  </Typography>
                  <Grid container spacing={2} sx={{ mt: 1 }}>
                    <Grid size={{ xs: 12, md: 12 }}>
                      <B4Slider
                        label="Fixed Split Position"
                        value={config.fragmentation.sni_position}
                        onChange={(value: number) =>
                          onChange("fragmentation.sni_position", value)
                        }
                        min={0}
                        max={10}
                        step={1}
                        helperText="Bytes from TLS payload start (0 = disabled)"
                      />
                    </Grid>
                  </Grid>
                  {config.fragmentation.sni_position > 0 &&
                    config.fragmentation.middle_sni && (
                      <Alert severity="info" sx={{ mt: 2 }}>
                        Both enabled → packet splits into 3 segments
                      </Alert>
                    )}
                </Box>
              </Collapse>
            </Grid>

            {/* QUIC note */}
            <Grid size={{ xs: 12 }}>
              <Alert severity="success" icon={<CallSplitIcon />}>
                <strong>QUIC:</strong> Automatically detects and splits at SNI
                inside encrypted packets. No configuration needed.
              </Alert>
            </Grid>
          </>
        )}

        {/* Combo Settings */}
        {strategy === "combo" && (
          <ComboSettings config={config} onChange={onChange} />
        )}

        {/* Disorder Settings */}
        {strategy === "disorder" && (
          <DisorderSettings config={config} onChange={onChange} />
        )}

        {/* OOB Settings */}
        {isOob && (
          <>
            <Grid size={{ xs: 12 }}>
              <Divider sx={{ my: 1 }}>
                <Chip label="OOB Configuration" size="small" />
              </Divider>
            </Grid>

            <Grid size={{ xs: 12 }}>
              <Alert severity="info">
                Inserts a byte with TCP URG flag. Server ignores it, but
                stateful DPI gets confused.
              </Alert>
            </Grid>

            <Grid size={{ xs: 12, md: 6 }}>
              <B4Slider
                label="Insert Position"
                value={config.fragmentation.oob_position || 1}
                onChange={(value: number) =>
                  onChange("fragmentation.oob_position", value)
                }
                min={1}
                max={50}
                step={1}
                helperText="Bytes before OOB insertion"
              />
            </Grid>

            <Grid size={{ xs: 12, md: 6 }}>
              <Box>
                <Typography variant="body2" gutterBottom>
                  OOB Byte:{" "}
                  <code>
                    {String.fromCharCode(config.fragmentation.oob_char || 120)}
                  </code>{" "}
                  (0x
                  {(config.fragmentation.oob_char || 120)
                    .toString(16)
                    .padStart(2, "0")}
                  )
                </Typography>
              </Box>
            </Grid>
          </>
        )}

        {/* TLS Record Settings */}
        {isTls && (
          <>
            <Grid size={{ xs: 12 }}>
              <Divider sx={{ my: 1 }}>
                <Chip label="TLS Record Configuration" size="small" />
              </Divider>
            </Grid>

            <Grid size={{ xs: 12 }}>
              <Alert severity="info">
                Splits ClientHello into multiple TLS records. DPI expecting
                single-record handshake fails to match.
              </Alert>
            </Grid>

            <Grid size={{ xs: 12, md: 6 }}>
              <B4Slider
                label="Record Split Position"
                value={config.fragmentation.tlsrec_pos || 1}
                onChange={(value: number) =>
                  onChange("fragmentation.tlsrec_pos", value)
                }
                min={1}
                max={100}
                step={1}
                helperText="First TLS record size in bytes"
              />
            </Grid>
          </>
        )}

        {/* Disabled state */}
        {!isActive && (
          <Grid size={{ xs: 12 }}>
            <Alert severity="warning">
              Fragmentation disabled. Only fake packets (if enabled) will be
              used for bypass.
            </Alert>
          </Grid>
        )}
      </Grid>
    </SettingSection>
  );
};
