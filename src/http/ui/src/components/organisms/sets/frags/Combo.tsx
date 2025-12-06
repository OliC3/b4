import React from "react";
import { Grid, Alert, Divider, Chip, Box, Typography } from "@mui/material";
import SettingSelect from "@atoms/common/B4Select";
import SettingSwitch from "@atoms/common/B4Switch";
import B4Slider from "@atoms/common/B4Slider";
import { B4SetConfig, ComboShuffleMode } from "@models/Config";
import { colors } from "@design";

interface ComboSettingsProps {
  config: B4SetConfig;
  onChange: (field: string, value: string | boolean | number) => void;
}

const shuffleModeOptions: { label: string; value: ComboShuffleMode }[] = [
  { label: "Middle Only", value: "middle" },
  { label: "Full Shuffle", value: "full" },
  { label: "Reverse Order", value: "reverse" },
];

export const ComboSettings: React.FC<ComboSettingsProps> = ({
  config,
  onChange,
}) => {
  const combo = config.fragmentation.combo;
  const middleSni = config.fragmentation.middle_sni;

  const enabledSplits = [
    combo.first_byte_split && "1st byte",
    combo.extension_split && "ext",
    middleSni && "SNI",
  ].filter(Boolean);

  return (
    <>
      <Grid size={{ xs: 12 }}>
        <Divider sx={{ my: 1 }}>
          <Chip label="Combo Strategy" size="small" />
        </Divider>
      </Grid>

      <Grid size={{ xs: 12 }}>
        <Alert severity="info">
          Combo combines multiple split points and sends segments out of order
          with timing jitter to confuse stateful DPI.
        </Alert>
      </Grid>

      {/* Split Points */}
      <Grid size={{ xs: 12 }}>
        <Typography
          variant="subtitle2"
          sx={{ mb: 1, color: colors.text.secondary }}
        >
          Split Points
        </Typography>
      </Grid>

      <Grid size={{ xs: 12, md: 4 }}>
        <SettingSwitch
          label="First Byte"
          checked={combo.first_byte_split}
          onChange={(checked: boolean) =>
            onChange("fragmentation.combo.first_byte_split", checked)
          }
          description="Split after 1st byte (timing desync)"
        />
      </Grid>

      <Grid size={{ xs: 12, md: 4 }}>
        <SettingSwitch
          label="Extension Split"
          checked={combo.extension_split}
          onChange={(checked: boolean) =>
            onChange("fragmentation.combo.extension_split", checked)
          }
          description="Split before SNI extension"
        />
      </Grid>

      <Grid size={{ xs: 12, md: 4 }}>
        <SettingSwitch
          label="SNI Split"
          checked={middleSni}
          onChange={(checked: boolean) =>
            onChange("fragmentation.middle_sni", checked)
          }
          description="Split in middle of SNI hostname"
        />
      </Grid>

      {/* Visual */}
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
            SEGMENT VISUALIZATION
          </Typography>
          <Box
            sx={{
              display: "flex",
              gap: 0.5,
              fontFamily: "monospace",
              fontSize: "0.7rem",
              flexWrap: "wrap",
            }}
          >
            {combo.first_byte_split && (
              <Box
                sx={{
                  p: 1,
                  bgcolor: colors.tertiary,
                  borderRadius: 0.5,
                  textAlign: "center",
                  minWidth: 40,
                }}
              >
                1B
              </Box>
            )}
            {combo.extension_split && (
              <Box
                sx={{
                  p: 1,
                  bgcolor: colors.accent.primary,
                  borderRadius: 0.5,
                  textAlign: "center",
                  flex: 1,
                  minWidth: 60,
                }}
              >
                Pre-SNI Ext
              </Box>
            )}
            {middleSni && (
              <>
                <Box
                  sx={{
                    p: 1,
                    bgcolor: colors.accent.secondary,
                    borderRadius: 0.5,
                    textAlign: "center",
                    minWidth: 50,
                  }}
                >
                  SNI₁
                </Box>
                <Box
                  sx={{
                    p: 1,
                    bgcolor: colors.accent.secondary,
                    borderRadius: 0.5,
                    textAlign: "center",
                    minWidth: 50,
                  }}
                >
                  SNI₂
                </Box>
              </>
            )}
            <Box
              sx={{
                p: 1,
                bgcolor: colors.quaternary,
                borderRadius: 0.5,
                textAlign: "center",
                flex: 1,
                minWidth: 60,
              }}
            >
              Rest...
            </Box>
          </Box>
          <Typography
            variant="caption"
            color="text.secondary"
            sx={{ mt: 1, display: "block" }}
          >
            {enabledSplits.length > 0
              ? `Active splits: ${enabledSplits.join(" → ")} → creates ${
                  enabledSplits.length + 1
                } segments`
              : "No splits enabled - packet sent as single segment"}
          </Typography>
        </Box>
      </Grid>

      {/* Shuffle Mode */}
      <Grid size={{ xs: 12, md: 6 }}>
        <SettingSelect
          label="Shuffle Mode"
          value={combo.shuffle_mode}
          options={shuffleModeOptions}
          onChange={(e) =>
            onChange(
              "fragmentation.combo.shuffle_mode",
              e.target.value as string
            )
          }
          helperText="How to reorder segments before sending"
        />
      </Grid>

      <Grid size={{ xs: 12, md: 6 }}>
        <Alert severity="info">
          {combo.shuffle_mode === "middle" &&
            "Middle: Keep first & last in place, shuffle middle segments"}
          {combo.shuffle_mode === "full" &&
            "Full: Randomly shuffle all segments"}
          {combo.shuffle_mode === "reverse" &&
            "Reverse: Send segments in reverse order"}
        </Alert>
      </Grid>

      {/* Timing */}
      <Grid size={{ xs: 12 }}>
        <Divider sx={{ my: 1 }}>
          <Chip label="Timing" size="small" />
        </Divider>
      </Grid>

      <Grid size={{ xs: 12, md: 6 }}>
        <B4Slider
          label="First Segment Delay"
          value={combo.first_delay_ms}
          onChange={(value: number) =>
            onChange("fragmentation.combo.first_delay_ms", value)
          }
          min={10}
          max={500}
          step={10}
          helperText="Delay after first segment (ms)"
        />
      </Grid>

      <Grid size={{ xs: 12, md: 6 }}>
        <B4Slider
          label="Jitter Max"
          value={combo.jitter_max_us}
          onChange={(value: number) =>
            onChange("fragmentation.combo.jitter_max_us", value)
          }
          min={100}
          max={10000}
          step={100}
          helperText="Max random delay between other segments (μs)"
        />
      </Grid>

      {!combo.first_byte_split && !combo.extension_split && !middleSni && (
        <Grid size={{ xs: 12 }}>
          <Alert severity="warning">
            No split points enabled. Enable at least one for Combo to work
            effectively.
          </Alert>
        </Grid>
      )}
    </>
  );
};
