import React from "react";
import {
  Button,
  Alert,
  Typography,
  List,
  ListItem,
  ListItemButton,
  ListItemText,
  ListItemIcon,
  Radio,
  Box,
  Stack,
  Chip,
} from "@mui/material";
import AddIcon from "@mui/icons-material/Add";
import DomainIcon from "@mui/icons-material/Language";
import {
  colors,
  button_primary,
  button_secondary,
  button_yellow_outline,
} from "@design";
import { B4Dialog } from "@molecules/common/B4Dialog";
import { B4Badge } from "@/components/atoms/common/B4Badge";
import { B4SetConfig, MAIN_SET_ID } from "@/models/Config";
import { SetSelector } from "@molecules/common/SetSelector";

interface IpInfo {
  ip: string;
  hostname?: string;
  org?: string;
  city?: string;
  region?: string;
  country?: string;
}

interface RipeNetworkInfo {
  asns: string[];
  prefix: string;
}

interface AddIpModalProps {
  open: boolean;
  ip: string;
  variants: string[];
  sets: B4SetConfig[];
  selected: string;
  ipInfoToken?: string;
  onClose: () => void;
  onSelectVariant: (variant: string) => void;
  onAdd: (setId: string) => void;
  onAddHostname?: (hostname: string) => void;
}

export const AddIpModal: React.FC<AddIpModalProps> = ({
  open,
  ip,
  sets,
  variants: initialVariants,
  selected,
  ipInfoToken,
  onClose,
  onSelectVariant,
  onAdd,
  onAddHostname,
}) => {
  const [selectedSetId, setSelectedSetId] = React.useState<string>("");
  const [ipInfo, setIpInfo] = React.useState<IpInfo | null>(null);
  const [loadingInfo, setLoadingInfo] = React.useState(false);
  const [loadingPrefixes, setLoadingPrefixes] = React.useState(false);
  const [variants, setVariants] = React.useState<string[]>(initialVariants);
  const [asn, setAsn] = React.useState<string>("");
  const [prefixes, setPrefixes] = React.useState<string[]>([]);
  const [addMode, setAddMode] = React.useState<"single" | "all">("single");

  React.useEffect(() => {
    if (open) {
      setIpInfo(null);
      setAsn("");
      setVariants(initialVariants);
      setPrefixes([]);
      setAddMode("single");
      if (sets.length > 0) {
        setSelectedSetId(MAIN_SET_ID);
      }
    }
  }, [open, sets, initialVariants]);

  const loadIpInfo = async () => {
    setLoadingInfo(true);
    try {
      const cleanIp = ip.split(":")[0].replace(/[[\]]/g, "");
      const response = await fetch(
        `/api/integration/ipinfo?ip=${encodeURIComponent(cleanIp)}`
      );
      if (response.ok) {
        const data = (await response.json()) as IpInfo;
        setIpInfo(data);

        // Extract ASN from org field (e.g., "AS13335 Cloudflare, Inc.")
        if (data.org) {
          const asnMatch = data.org.match(/AS(\d+)/);
          if (asnMatch) {
            setAsn(asnMatch[1]);
          }
        }
      }
    } catch (error) {
      console.error("Failed to load IP info:", error);
    } finally {
      setLoadingInfo(false);
    }
  };

  const loadRipeNetworkInfo = async () => {
    setLoadingInfo(true);
    try {
      const cleanIp = ip.split(":")[0].replace(/[[\]]/g, "");
      const response = await fetch(
        `/api/integration/ripestat?ip=${encodeURIComponent(cleanIp)}`
      );
      if (response.ok) {
        const data = (await response.json()) as { data: RipeNetworkInfo };
        const networkData = data.data;
        if (networkData.asns && networkData.asns.length > 0) {
          setAsn(networkData.asns[0]);
          setIpInfo({
            ip: cleanIp,
            org: `AS${networkData.asns[0]}`,
          });
        }
      }
    } catch (error) {
      console.error("Failed to load RIPE network info:", error);
    } finally {
      setLoadingInfo(false);
    }
  };

  const loadRipePrefixes = async () => {
    if (!asn) return;

    setLoadingPrefixes(true);
    try {
      const response = await fetch(
        `/api/integration/ripestat/asn?asn=${encodeURIComponent(asn)}`
      );
      if (response.ok) {
        const data = (await response.json()) as {
          data: { prefixes: Array<{ prefix: string }> };
        };
        const loadedPrefixes = data.data.prefixes.map((p) => p.prefix);
        setPrefixes(loadedPrefixes);
        setAddMode("all");
      }
    } catch (error) {
      console.error("Failed to load RIPE prefixes:", error);
    } finally {
      setLoadingPrefixes(false);
    }
  };

  const handleAdd = () => {
    if (addMode === "all" && prefixes.length > 0) {
      onAdd(selectedSetId);
    } else {
      onAdd(selectedSetId);
    }
  };

  React.useEffect(() => {
    if (asn && open) {
      void loadRipePrefixes();
    }
  }, [asn, open]);

  const handleAddHostname = () => {
    if (ipInfo?.hostname && onAddHostname) {
      onAddHostname(ipInfo.hostname);
      onClose();
    }
  };

  return (
    <B4Dialog
      title="Add IP/CIDR to Manual List"
      icon={<DomainIcon />}
      open={open}
      onClose={onClose}
      maxWidth="md"
      actions={
        <>
          <Button
            onClick={onClose}
            variant="outlined"
            sx={{ ...button_secondary }}
          >
            Cancel
          </Button>
          <Box sx={{ flex: 1 }} />
          <Button
            onClick={handleAdd}
            variant="contained"
            startIcon={<AddIcon />}
            disabled={!selected && prefixes.length === 0}
            sx={{ ...button_primary }}
          >
            {addMode === "all" && prefixes.length > 0
              ? `Add All ${prefixes.length} Prefixes`
              : "Add IP/CIDR"}
          </Button>
        </>
      }
    >
      <>
        <Alert severity="info" sx={{ mb: 2 }}>
          Select the desired IP or CIDR range. You can enrich with network
          information to load all ASN prefixes.
        </Alert>

        {/* Original IP and Enrichment section */}
        <Box sx={{ mb: 3 }}>
          {!ipInfo ? (
            <Stack direction="row" spacing={2} alignItems="center">
              <Typography variant="body2" color="text.secondary">
                Original IP: <B4Badge label={ip} badgeVariant="secondary" />
              </Typography>
              <Box sx={{ flex: 1 }} />
              {ipInfoToken && (
                <Button
                  variant="outlined"
                  size="small"
                  onClick={() => void loadIpInfo()}
                  disabled={loadingInfo}
                  sx={{ ...button_yellow_outline }}
                >
                  {loadingInfo ? "Loading..." : "Enrich with IPInfo"}
                </Button>
              )}
              <Button
                variant="outlined"
                size="small"
                onClick={() => void loadRipeNetworkInfo()}
                disabled={loadingInfo}
                sx={{ ...button_yellow_outline }}
              >
                {loadingInfo ? "Loading..." : "Load Network Info"}
              </Button>
            </Stack>
          ) : (
            <>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                Original IP: <B4Badge label={ip} badgeVariant="secondary" />
              </Typography>
              <Box
                sx={{
                  p: 2,
                  bgcolor: colors.background.dark,
                  borderRadius: 1,
                  border: `1px solid ${colors.border.default}`,
                }}
              >
                <Stack direction="row" spacing={2} alignItems="center">
                  <Box sx={{ flex: 1 }}>
                    {ipInfo.org && (
                      <Typography variant="body2" color="text.primary">
                        <strong>Org:</strong> {ipInfo.org}
                      </Typography>
                    )}
                    {ipInfo.hostname && (
                      <Typography variant="body2" color="text.secondary">
                        <strong>Hostname:</strong> {ipInfo.hostname}
                      </Typography>
                    )}
                    {(ipInfo.city || ipInfo.region || ipInfo.country) && (
                      <Typography variant="body2" color="text.secondary">
                        <strong>Location:</strong>{" "}
                        {[ipInfo.city, ipInfo.region, ipInfo.country]
                          .filter(Boolean)
                          .join(", ")}
                      </Typography>
                    )}
                    {asn && loadingPrefixes && (
                      <Typography
                        variant="body2"
                        color={colors.secondary}
                        sx={{ mt: 1 }}
                      >
                        Loading AS{asn} prefixes...
                      </Typography>
                    )}
                  </Box>
                  {ipInfo.hostname && onAddHostname && (
                    <Button
                      size="small"
                      onClick={handleAddHostname}
                      sx={{ ...button_yellow_outline }}
                    >
                      Add Hostname
                    </Button>
                  )}
                </Stack>
              </Box>
            </>
          )}
        </Box>

        {sets.length > 0 && (
          <SetSelector
            sets={sets}
            value={selectedSetId}
            onChange={setSelectedSetId}
          />
        )}

        {prefixes.length > 0 ? (
          <>
            <Typography
              variant="body2"
              color="text.secondary"
              sx={{ mb: 1, mt: 2 }}
            >
              Loaded {prefixes.length} prefixes from AS{asn}
            </Typography>
            <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
              <Chip
                label={`Add single IP/CIDR (${selected})`}
                onClick={() => setAddMode("single")}
                sx={{
                  bgcolor:
                    addMode === "single"
                      ? colors.accent.primary
                      : colors.background.dark,
                  color: addMode === "single" ? colors.primary : "text.primary",
                  cursor: "pointer",
                  "&:hover": {
                    bgcolor:
                      addMode === "single"
                        ? colors.accent.primaryHover
                        : colors.accent.primaryStrong,
                  },
                }}
              />
              <Chip
                label={`Add all ${prefixes.length} prefixes`}
                onClick={() => setAddMode("all")}
                sx={{
                  bgcolor:
                    addMode === "all"
                      ? colors.accent.primary
                      : colors.background.dark,
                  color: addMode === "all" ? colors.primary : "text.primary",
                  cursor: "pointer",
                  "&:hover": {
                    bgcolor:
                      addMode === "all"
                        ? colors.accent.primaryHover
                        : colors.accent.primaryStrong,
                  },
                }}
              />
            </Stack>
          </>
        ) : (
          <>
            <Typography
              variant="body2"
              color="text.secondary"
              sx={{ mb: 1, mt: 2 }}
            >
              CIDR variants:
            </Typography>
            <List sx={{ maxHeight: 400, overflow: "auto" }}>
              {variants.map((variant) => (
                <ListItem key={variant} disablePadding>
                  <ListItemButton
                    onClick={() => onSelectVariant(variant)}
                    selected={selected === variant}
                    sx={{
                      borderRadius: 1,
                      mb: 0.5,
                      "&.Mui-selected": {
                        bgcolor: colors.accent.primary,
                        "&:hover": { bgcolor: colors.accent.primaryHover },
                      },
                    }}
                  >
                    <ListItemIcon>
                      <Radio
                        checked={selected === variant}
                        sx={{
                          color: colors.border.default,
                          "&.Mui-checked": { color: colors.primary },
                        }}
                      />
                    </ListItemIcon>
                    <ListItemText
                      primary={variant}
                      secondary={(() => {
                        const [, cidr] = variant.split("/");
                        if (cidr === "32" || cidr === "128") return "Single IP";
                        if (cidr === "24") return "~256 IPs - local subnet";
                        if (cidr === "16") return "~65K IPs - network block";
                        if (cidr === "8") return "~16M IPs - class A";
                        if (cidr === "64") return "IPv6 subnet";
                        if (cidr === "48") return "IPv6 site";
                        return "IPv6 ISP range";
                      })()}
                    />
                  </ListItemButton>
                </ListItem>
              ))}
            </List>
          </>
        )}
      </>
    </B4Dialog>
  );
};
