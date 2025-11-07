import React from "react";
import {
  Grid,
  Paper,
  Typography,
  List,
  ListItem,
  ListItemText,
  Stack,
  Chip,
  Divider,
} from "@mui/material";
import { formatNumber } from "@utils";
import { colors } from "@design";
import { ProtocolChip } from "@atoms/common/ProtocolChip";

interface Connection {
  timestamp: string;
  protocol: "TCP" | "UDP";
  domain: string;
  source: string;
  destination: string;
  is_target: boolean;
}

interface DashboardActivityPanelsProps {
  topDomains: Record<string, number>;
  recentConnections: Connection[];
}

export const DashboardActivityPanels: React.FC<
  DashboardActivityPanelsProps
> = ({ topDomains, recentConnections }) => {
  const topDomainsData = Object.entries(topDomains)
    .sort((a, b) => b[1] - a[1])
    .slice(0, 10);

  return (
    <Grid container spacing={3}>
      <Grid size={{ xs: 12, md: 6 }}>
        <Paper
          sx={{
            p: 2,
            bgcolor: colors.background.paper,
            borderColor: colors.border.default,
          }}
          variant="outlined"
        >
          <Typography variant="h6" sx={{ mb: 2, color: colors.text.primary }}>
            Top Domains
          </Typography>
          {topDomainsData.length > 0 ? (
            <List dense>
              {topDomainsData.map(([domain, count], index) => (
                <ListItem key={domain}>
                  <ListItemText
                    primary={
                      <Stack
                        direction="row"
                        justifyContent="space-between"
                        alignItems="center"
                      >
                        <Typography
                          variant="body2"
                          sx={{ color: colors.text.primary }}
                        >
                          {index + 1}. {domain}
                        </Typography>
                        <Chip
                          label={formatNumber(count)}
                          size="small"
                          sx={{
                            bgcolor: colors.accent.primary,
                            color: colors.primary,
                          }}
                        />
                      </Stack>
                    }
                  />
                  <Divider />
                </ListItem>
              ))}
            </List>
          ) : (
            <Typography
              sx={{ color: colors.text.secondary, textAlign: "center", py: 4 }}
            >
              No domain data available yet
            </Typography>
          )}
        </Paper>
      </Grid>

      <Grid size={{ xs: 12, md: 6 }}>
        <Paper
          sx={{
            p: 2,
            bgcolor: colors.background.paper,
            borderColor: colors.border.default,
            height: "100%",
          }}
          variant="outlined"
        >
          <Typography variant="h6" sx={{ mb: 2, color: colors.text.primary }}>
            Recent Activity
          </Typography>
          <List dense sx={{ maxHeight: 400, overflow: "auto" }}>
            {recentConnections.map((conn, index) => (
              <ListItem key={index}>
                <ListItemText
                  primary={
                    <Stack direction="row" spacing={1} alignItems="center">
                      <ProtocolChip protocol={conn.protocol} />
                      <Typography
                        variant="body2"
                        sx={{ color: colors.text.primary }}
                      >
                        {conn.domain}
                      </Typography>
                      {conn.is_target && (
                        <Chip
                          label="TARGET"
                          size="small"
                          sx={{
                            bgcolor: "#4caf5033",
                            color: "#4caf50",
                            fontWeight: 600,
                          }}
                        />
                      )}
                    </Stack>
                  }
                  secondary={
                    <Typography
                      variant="caption"
                      sx={{ color: colors.text.secondary }}
                    >
                      {conn.source} → {conn.destination} •{" "}
                      {new Date(conn.timestamp).toLocaleTimeString()}
                    </Typography>
                  }
                />
              </ListItem>
            ))}
            {recentConnections.length === 0 && (
              <Typography
                sx={{
                  color: colors.text.secondary,
                  textAlign: "center",
                  py: 4,
                }}
              >
                No recent connections
              </Typography>
            )}
          </List>
        </Paper>
      </Grid>
    </Grid>
  );
};
