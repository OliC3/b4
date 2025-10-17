import React from "react";
import { Box, Grid, Paper, Typography } from "@mui/material";
import { SimpleLineChart } from "../../molecules/metrics/SimpleLineChart";
import { SimplePieChart } from "../../molecules/metrics/SimplePieChart";
import { colors } from "../../../Theme";

interface DashboardChartsProps {
  connectionRate: Array<{ timestamp: number; value: number }>;
  protocolDist: Record<string, number>;
}

export const DashboardCharts: React.FC<DashboardChartsProps> = ({
  connectionRate,
  protocolDist,
}) => {
  return (
    <Grid container spacing={3}>
      <Grid size={{ xs: 12, lg: 6 }}>
        <Paper
          sx={{
            p: 2,
            bgcolor: colors.background.paper,
            borderColor: colors.border.default,
          }}
          variant="outlined"
        >
          <Typography variant="h6" sx={{ mb: 2, color: colors.text.primary }}>
            Connection Rate (last 60s)
          </Typography>
          <Box sx={{ pl: 5 }}>
            <SimpleLineChart data={connectionRate} color={colors.secondary} />
          </Box>
        </Paper>
      </Grid>

      <Grid size={{ xs: 12, lg: 6 }}>
        <Paper
          sx={{
            p: 2,
            bgcolor: colors.background.paper,
            borderColor: colors.border.default,
          }}
          variant="outlined"
        >
          <Typography variant="h6" sx={{ mb: 2, color: colors.text.primary }}>
            Protocol Distribution
          </Typography>
          <SimplePieChart data={protocolDist} />
        </Paper>
      </Grid>
    </Grid>
  );
};
