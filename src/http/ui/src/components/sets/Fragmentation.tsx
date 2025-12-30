import { Grid, Typography, Box } from "@mui/material";
import { FragIcon } from "@b4.icons";
import {
  B4Section,
  B4Switch,
  B4Select,
  B4Slider,
  B4Alert,
  B4FormHeader,
} from "@b4.elements";
import { B4SetConfig, FragmentationStrategy } from "@models/config";
import { ComboSettings } from "./frags/Combo";
import { DisorderSettings } from "./frags/Disorder";
import { ExtSplitSettings } from "./frags/ExtSplit";
import { FirstByteSettings } from "./frags/FirstByte";
import { TcpIpSettings } from "./frags/TcpIp";

interface FragmentationSettingsProps {
  config: B4SetConfig;
  onChange: (
    field: string,
    value: string | boolean | number | string[]
  ) => void;
}

const fragmentationOptions: { label: string; value: FragmentationStrategy }[] =
  [
    { label: "Combo", value: "combo" },
    { label: "Hybrid", value: "hybrid" },
    { label: "Disorder", value: "disorder" },
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
  const strategy = config.fragmentation.strategy;
  const isTcpOrIp = strategy === "tcp" || strategy === "ip";
  const isOob = strategy === "oob";
  const isTls = strategy === "tls";
  const isActive = strategy !== "none";

  return (
    <B4Section
      title="Fragmentation Strategy"
      description="Split packets to evade DPI pattern matching"
      icon={<FragIcon />}
    >
      <Grid container spacing={3}>
        {/* Strategy Selection */}
        <Grid size={{ xs: 12, md: 6 }}>
          <B4Select
            label="Method"
            value={strategy}
            options={fragmentationOptions}
            onChange={(e) =>
              onChange("fragmentation.strategy", e.target.value as string)
            }
          />
        </Grid>

        <Grid size={{ xs: 12, md: 6 }}>
          <B4Switch
            label="Reverse Fragment Order"
            checked={config.fragmentation.reverse_order}
            onChange={(checked: boolean) =>
              onChange("fragmentation.reverse_order", checked)
            }
            description="Send second fragment first"
          />
        </Grid>

        {isTcpOrIp && <TcpIpSettings config={config} onChange={onChange} />}

        {strategy === "combo" && (
          <ComboSettings config={config} onChange={onChange} />
        )}

        {strategy === "disorder" && (
          <DisorderSettings config={config} onChange={onChange} />
        )}
        {strategy === "extsplit" && <ExtSplitSettings />}

        {strategy === "firstbyte" && <FirstByteSettings config={config} />}

        {isOob && (
          <>
            <B4FormHeader label="OOB (Out-of-Band) Strategy" />

            <B4Alert>
              Inserts a byte with TCP URG flag. Server ignores it, but stateful
              DPI gets confused.
            </B4Alert>

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
            <B4FormHeader label="TLS Record Splitting Strategy" />

            <B4Alert>
              Splits ClientHello into multiple TLS records. DPI expecting
              single-record handshake fails to match.
            </B4Alert>

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

        {!isActive && (
          <B4Alert severity="warning">
            Fragmentation disabled. Only fake packets (if enabled) will be used
            for bypass.
          </B4Alert>
        )}
      </Grid>
    </B4Section>
  );
};
