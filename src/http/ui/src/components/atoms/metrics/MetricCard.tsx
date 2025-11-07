// src/http/ui/src/components/atoms/MetricCard.tsx
import React from "react";
import { Box, Card, CardContent, Stack, Typography } from "@mui/material";
import { TrendingUp as TrendingUpIcon } from "@mui/icons-material";
import { colors } from "@design";

interface MetricCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  icon: React.ReactNode;
  color?: string;
  trend?: number;
}

export const MetricCard: React.FC<MetricCardProps> = ({
  title,
  value,
  subtitle,
  icon,
  color = colors.primary,
  trend,
}) => (
  <Card
    sx={{
      bgcolor: colors.background.paper,
      border: `1px solid ${color}33`,
      position: "relative",
      overflow: "visible",
      "&:hover": {
        borderColor: color + "66",
        boxShadow: `0 0 20px ${color}22`,
      },
    }}
  >
    <CardContent>
      <Stack
        direction="row"
        justifyContent="space-between"
        alignItems="flex-start"
      >
        <Box>
          <Typography
            variant="caption"
            sx={{ color: colors.text.secondary, textTransform: "uppercase" }}
          >
            {title}
          </Typography>
          <Typography
            variant="h4"
            sx={{ color: colors.text.primary, fontWeight: 600, mt: 0.5 }}
          >
            {value}
          </Typography>
          {subtitle && (
            <Typography variant="caption" sx={{ color: colors.text.secondary }}>
              {subtitle}
            </Typography>
          )}
          {trend !== undefined && (
            <Box sx={{ display: "flex", alignItems: "center", mt: 0.5 }}>
              <TrendingUpIcon
                sx={{
                  fontSize: 16,
                  color: trend > 0 ? "#4caf50" : "#f44336",
                  mr: 0.5,
                }}
              />
              <Typography
                variant="caption"
                sx={{ color: trend > 0 ? "#4caf50" : "#f44336" }}
              >
                {trend > 0 ? "+" : ""}
                {trend.toFixed(1)}%
              </Typography>
            </Box>
          )}
        </Box>
        <Box
          sx={{
            p: 1,
            borderRadius: 2,
            bgcolor: color + "22",
            color: color,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
          }}
        >
          {icon}
        </Box>
      </Stack>
    </CardContent>
  </Card>
);
