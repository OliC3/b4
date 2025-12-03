import React, { useState } from "react";
import {
  Grid,
  Divider,
  Chip,
  IconButton,
  Box,
  Typography,
} from "@mui/material";
import { Security as SecurityIcon, Add as AddIcon } from "@mui/icons-material";
import SettingSection from "@molecules/common/B4Section";
import SettingSelect from "@atoms/common/B4Select";
import SettingTextField from "@atoms/common/B4TextField";
import SettingSwitch from "@atoms/common/B4Switch";
import B4Slider from "@atoms/common/B4Slider";
import { B4SetConfig, FakingPayloadType, MutationMode } from "@models/Config";
import { colors } from "@design";

interface FakingSettingsProps {
  config: B4SetConfig;
  onChange: (
    field: string,
    value: string | boolean | number | string[]
  ) => void;
}

const FAKE_STRATEGIES = [
  { value: "ttl", label: "TTL" },
  { value: "randseq", label: "Random Sequence" },
  { value: "pastseq", label: "Past Sequence" },
  { value: "tcp_check", label: "TCP Check" },
  { value: "md5sum", label: "MD5 Sum" },
];

const FAKE_PAYLOAD_TYPES = [
  { value: 0, label: "Random" },
  { value: 1, label: "Custom" },
  { value: 2, label: "Preset: Google (classic)" },
  { value: 3, label: "Preset: DuckDuckGo" },
];

const MUTATION_MODES: { value: MutationMode; label: string }[] = [
  { value: "off", label: "Disabled" },
  { value: "random", label: "Random" },
  { value: "grease", label: "GREASE Extensions" },
  { value: "padding", label: "Padding" },
  { value: "fakeext", label: "Fake Extensions" },
  { value: "fakesni", label: "Fake SNIs" },
  { value: "advanced", label: "Advanced (All)" },
];

const mutationModeDescriptions: Record<MutationMode, string> = {
  off: "No ClientHello mutation applied",
  random: "Randomize extension order and add noise",
  grease: "Insert GREASE extensions to confuse DPI",
  padding: "Add padding extension to reach target size",
  fakeext: "Insert fake/unknown TLS extensions",
  fakesni: "Add additional fake SNI entries",
  advanced: "Combine multiple mutation techniques",
};

export const FakingSettings: React.FC<FakingSettingsProps> = ({
  config,
  onChange,
}) => {
  const [newFakeSni, setNewFakeSni] = useState("");

  const mutation = config.faking.sni_mutation || {
    mode: "off" as MutationMode,
    grease_count: 3,
    padding_size: 2048,
    fake_ext_count: 5,
    fake_snis: [],
  };

  const isMutationEnabled = mutation.mode !== "off";
  const showGreaseSettings = ["grease", "advanced"].includes(mutation.mode);
  const showPaddingSettings = ["padding", "advanced"].includes(mutation.mode);
  const showFakeExtSettings = ["fakeext", "advanced"].includes(mutation.mode);
  const showFakeSniSettings = ["fakesni", "advanced"].includes(mutation.mode);

  const handleAddFakeSni = () => {
    if (newFakeSni.trim()) {
      const current = mutation.fake_snis || [];
      if (!current.includes(newFakeSni.trim())) {
        onChange("faking.sni_mutation.fake_snis", [
          ...current,
          newFakeSni.trim(),
        ]);
      }
      setNewFakeSni("");
    }
  };

  const handleRemoveFakeSni = (sni: string) => {
    const current = mutation.fake_snis || [];
    onChange(
      "faking.sni_mutation.fake_snis",
      current.filter((s) => s !== sni)
    );
  };

  return (
    <>
      <SettingSection
        title="Fake SNI Configuration"
        description="Configure fake SNI packets to confuse DPI"
        icon={<SecurityIcon />}
      >
        <Grid container spacing={2}>
          <Grid size={{ xs: 12 }}>
            <SettingSwitch
              label="Enable Fake SNI"
              checked={config.faking.sni}
              onChange={(checked: boolean) => onChange("faking.sni", checked)}
              description="Send fake SNI packets before real ClientHello"
            />
          </Grid>
          <Grid size={{ xs: 12, md: 6 }}>
            <SettingSelect
              label="Fake Strategy"
              value={config.faking.strategy}
              options={FAKE_STRATEGIES}
              onChange={(e) =>
                onChange("faking.strategy", e.target.value as string)
              }
              helperText="How to make fake packets unprocessable by server"
              disabled={!config.faking.sni}
            />
          </Grid>
          <Grid size={{ xs: 12, md: 6 }}>
            <SettingSelect
              label="Fake Payload Type"
              value={config.faking.sni_type}
              options={FAKE_PAYLOAD_TYPES}
              onChange={(e) =>
                onChange("faking.sni_type", Number(e.target.value))
              }
              helperText="Content of fake packets"
              disabled={!config.faking.sni}
            />
          </Grid>
          <Grid size={{ xs: 12, md: 4 }}>
            <B4Slider
              label="Fake TTL"
              value={config.faking.ttl}
              onChange={(value: number) => onChange("faking.ttl", value)}
              min={1}
              max={64}
              step={1}
              helperText="TTL for fake packets (should expire before server)"
              disabled={!config.faking.sni}
            />
          </Grid>
          <Grid size={{ xs: 12, md: 4 }}>
            <SettingTextField
              label="Sequence Offset"
              type="number"
              value={config.faking.seq_offset}
              onChange={(e) =>
                onChange("faking.seq_offset", Number(e.target.value))
              }
              helperText="TCP sequence number offset for pastseq strategy"
              disabled={!config.faking.sni}
            />
          </Grid>
          <Grid size={{ xs: 12, md: 4 }}>
            <B4Slider
              label="Fake Packet Count"
              value={config.faking.sni_seq_length}
              onChange={(value: number) =>
                onChange("faking.sni_seq_length", value)
              }
              min={1}
              max={20}
              step={1}
              helperText="Number of fake packets to send"
              disabled={!config.faking.sni}
            />
          </Grid>
          {config.faking.sni_type === FakingPayloadType.CUSTOM && (
            <Grid size={{ xs: 12 }}>
              <SettingTextField
                label="Custom Payload (Hex)"
                value={config.faking.custom_payload}
                onChange={(e) =>
                  onChange("faking.custom_payload", e.target.value)
                }
                helperText="Hex-encoded payload for fake packets (use Capture feature to get real payloads)"
                disabled={!config.faking.sni}
                multiline
                rows={2}
              />
            </Grid>
          )}
        </Grid>
      </SettingSection>

      {/* SNI Mutation Section */}
      <SettingSection
        title="ClientHello Mutation"
        description="Modify TLS ClientHello structure to evade fingerprinting"
        icon={<SecurityIcon />}
      >
        <Grid container spacing={2}>
          <Grid size={{ xs: 12, md: 6 }}>
            <SettingSelect
              label="Mutation Mode"
              value={mutation.mode}
              options={MUTATION_MODES}
              onChange={(e) =>
                onChange("faking.sni_mutation.mode", e.target.value as string)
              }
              helperText={mutationModeDescriptions[mutation.mode]}
            />
          </Grid>

          {isMutationEnabled && (
            <>
              {showGreaseSettings && (
                <>
                  <Grid size={{ xs: 12 }}>
                    <Divider sx={{ my: 1 }}>
                      <Chip label="GREASE Configuration" size="small" />
                    </Divider>
                  </Grid>
                  <Grid size={{ xs: 12 }}>
                    <B4Slider
                      label="GREASE Extension Count"
                      value={mutation.grease_count}
                      onChange={(value: number) =>
                        onChange("faking.sni_mutation.grease_count", value)
                      }
                      min={1}
                      max={10}
                      step={1}
                      helperText="Number of GREASE extensions to insert"
                    />
                  </Grid>
                </>
              )}

              {showPaddingSettings && (
                <>
                  <Grid size={{ xs: 12 }}>
                    <Divider sx={{ my: 1 }}>
                      <Chip label="Padding Configuration" size="small" />
                    </Divider>
                  </Grid>
                  <Grid size={{ xs: 12 }}>
                    <B4Slider
                      label="Padding Size"
                      value={mutation.padding_size}
                      onChange={(value: number) =>
                        onChange("faking.sni_mutation.padding_size", value)
                      }
                      min={256}
                      max={16384}
                      step={256}
                      valueSuffix=" bytes"
                      helperText="Target ClientHello size with padding"
                    />
                  </Grid>
                </>
              )}

              {showFakeExtSettings && (
                <>
                  <Grid size={{ xs: 12 }}>
                    <Divider sx={{ my: 1 }}>
                      <Chip
                        label="Fake Extensions Configuration"
                        size="small"
                      />
                    </Divider>
                  </Grid>
                  <Grid size={{ xs: 12 }}>
                    <B4Slider
                      label="Fake Extension Count"
                      value={mutation.fake_ext_count}
                      onChange={(value: number) =>
                        onChange("faking.sni_mutation.fake_ext_count", value)
                      }
                      min={1}
                      max={15}
                      step={1}
                      helperText="Number of fake TLS extensions to insert"
                    />
                  </Grid>
                </>
              )}

              {showFakeSniSettings && (
                <>
                  <Grid size={{ xs: 12 }}>
                    <Divider sx={{ my: 1 }}>
                      <Chip label="Fake SNI Configuration" size="small" />
                    </Divider>
                  </Grid>
                  <Grid size={{ xs: 12, md: 6 }}>
                    <Box
                      sx={{ display: "flex", gap: 1, alignItems: "flex-start" }}
                    >
                      <SettingTextField
                        label="Add Fake SNI"
                        value={newFakeSni}
                        onChange={(e) => setNewFakeSni(e.target.value)}
                        onKeyDown={(e) => {
                          if (e.key === "Enter") {
                            e.preventDefault();
                            handleAddFakeSni();
                          }
                        }}
                        placeholder="e.g., ya.ru, vk.com"
                        helperText="Additional SNI values to inject into ClientHello"
                      />
                      <IconButton
                        onClick={handleAddFakeSni}
                        sx={{
                          bgcolor: colors.accent.secondary,
                          color: colors.secondary,
                          "&:hover": { bgcolor: colors.accent.secondaryHover },
                        }}
                      >
                        <AddIcon />
                      </IconButton>
                    </Box>
                  </Grid>
                  {(mutation.fake_snis?.length ?? 0) > 0 && (
                    <Grid size={{ xs: 12, md: 6 }}>
                      <Typography variant="subtitle2" gutterBottom>
                        Active Fake SNIs
                      </Typography>
                      <Box
                        sx={{
                          display: "flex",
                          flexWrap: "wrap",
                          gap: 1,
                          p: 1,
                          border: `1px solid ${colors.border.default}`,
                          borderRadius: 1,
                          bgcolor: colors.background.paper,
                        }}
                      >
                        {mutation.fake_snis.map((sni) => (
                          <Chip
                            key={sni}
                            label={sni}
                            onDelete={() => handleRemoveFakeSni(sni)}
                            size="small"
                            sx={{
                              bgcolor: colors.accent.primary,
                              color: colors.secondary,
                              "& .MuiChip-deleteIcon": {
                                color: colors.secondary,
                              },
                            }}
                          />
                        ))}
                      </Box>
                    </Grid>
                  )}
                </>
              )}
            </>
          )}
        </Grid>
      </SettingSection>
    </>
  );
};
