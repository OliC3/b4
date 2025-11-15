import React, { useEffect, useState } from "react";
import {
  Button,
  Alert,
  Typography,
  Box,
  CircularProgress,
  Stack,
} from "@mui/material";
import InfoIcon from "@mui/icons-material/Info";
import AddIcon from "@mui/icons-material/Add";
import { button_primary, button_secondary } from "@design";
import { B4Dialog } from "@molecules/common/B4Dialog";
import { B4Badge } from "@/components/atoms/common/B4Badge";

interface IpInfo {
  ip: string;
  hostname?: string;
  city?: string;
  region?: string;
  country?: string;
  loc?: string;
  org?: string;
  postal?: string;
  timezone?: string;
}

interface IpInfoModalProps {
  open: boolean;
  ip: string;
  token: string;
  onClose: () => void;
  onAddHostname?: (hostname: string) => void;
}

export const IpInfoModal: React.FC<IpInfoModalProps> = ({
  open,
  ip,
  token,
  onClose,
  onAddHostname,
}) => {
  const [ipInfo, setIpInfo] = useState<IpInfo | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open || !ip || !token) return;

    const fetchIpInfo = async () => {
      setLoading(true);
      setError(null);
      try {
        const cleanIp = ip.split(":")[0].replace(/[[\]]/g, "");
        const response = await fetch(
          `/api/integration/ipinfo?ip=${encodeURIComponent(cleanIp)}`
        );
        if (!response.ok) throw new Error("Failed to fetch IP info");
        const data = (await response.json()) as IpInfo;
        setIpInfo(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Unknown error");
      } finally {
        setLoading(false);
      }
    };

    void fetchIpInfo();
  }, [open, ip, token]);

  const handleAddHostname = () => {
    if (ipInfo?.hostname && onAddHostname) {
      onAddHostname(ipInfo.hostname);
      onClose();
    }
  };

  return (
    <B4Dialog
      title="IP Information"
      icon={<InfoIcon />}
      open={open}
      onClose={onClose}
      actions={
        <>
          {ipInfo?.hostname && onAddHostname && (
            <Button
              onClick={handleAddHostname}
              variant="contained"
              startIcon={<AddIcon />}
              sx={{ ...button_primary }}
            >
              Add Hostname
            </Button>
          )}
          <Box sx={{ flex: 1 }} />
          <Button
            onClick={onClose}
            variant="outlined"
            sx={{ ...button_secondary }}
          >
            Close
          </Button>
        </>
      }
    >
      <>
        {loading && (
          <Box sx={{ display: "flex", justifyContent: "center", py: 4 }}>
            <CircularProgress />
          </Box>
        )}

        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        {ipInfo && !loading && (
          <Stack spacing={2}>
            {ipInfo.org && (
              <Box>
                <Typography variant="caption" color="text.secondary">
                  Organization
                </Typography>
                <Typography variant="body1">{ipInfo.org}</Typography>
              </Box>
            )}

            {ipInfo.hostname && (
              <Box>
                <Typography variant="caption" color="text.secondary">
                  Hostname
                </Typography>
                <Typography variant="body1" fontFamily="monospace">
                  <B4Badge label={ipInfo.hostname} badgeVariant="secondary" />
                </Typography>
              </Box>
            )}

            <Box>
              <Typography variant="caption" color="text.secondary">
                IP Address
              </Typography>
              <Typography variant="body1" fontFamily="monospace">
                {ipInfo.ip}
              </Typography>
            </Box>

            {(ipInfo.city || ipInfo.region || ipInfo.country) && (
              <Box>
                <Typography variant="caption" color="text.secondary">
                  Location
                </Typography>
                <Typography variant="body1">
                  {[ipInfo.city, ipInfo.region, ipInfo.country]
                    .filter(Boolean)
                    .join(", ")}
                </Typography>
              </Box>
            )}

            {ipInfo.loc && (
              <Box>
                <Typography variant="caption" color="text.secondary">
                  Coordinates
                </Typography>
                <Typography variant="body1" fontFamily="monospace">
                  {ipInfo.loc}
                </Typography>
              </Box>
            )}

            {ipInfo.timezone && (
              <Box>
                <Typography variant="caption" color="text.secondary">
                  Timezone
                </Typography>
                <Typography variant="body1">{ipInfo.timezone}</Typography>
              </Box>
            )}

            {ipInfo.postal && (
              <Box>
                <Typography variant="caption" color="text.secondary">
                  Postal Code
                </Typography>
                <Typography variant="body1">{ipInfo.postal}</Typography>
              </Box>
            )}
          </Stack>
        )}
      </>
    </B4Dialog>
  );
};
