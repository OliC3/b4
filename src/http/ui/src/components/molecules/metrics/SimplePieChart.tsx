// src/http/ui/src/components/molecules/SimplePieChart.tsx
import React from "react";
import { Box, Typography } from "@mui/material";
import { colors } from "@design";
import { formatNumber } from "@utils";

interface SimplePieChartProps {
  data: Record<string, number>;
}

export const SimplePieChart: React.FC<SimplePieChartProps> = ({ data }) => {
  const total = Object.values(data).reduce((a, b) => a + b, 0);
  if (total === 0) return <Typography>No data</Typography>;

  let currentAngle = 0;
  const pieData = Object.entries(data).map(([name, value], index) => {
    const percentage = (value / total) * 100;
    const startAngle = currentAngle;
    currentAngle += (value / total) * 360;

    return {
      name,
      value,
      percentage,
      startAngle,
      endAngle: currentAngle,
      color: index === 0 ? colors.primary : colors.secondary,
    };
  });

  const radius = 80;
  const centerX = 100;
  const centerY = 100;

  return (
    <Box sx={{ display: "flex", alignItems: "center", gap: 2 }}>
      <svg width="200" height="200" viewBox="0 0 200 200">
        {pieData.map((slice, index) => {
          const startAngleRad = (slice.startAngle * Math.PI) / 180;
          const endAngleRad = (slice.endAngle * Math.PI) / 180;

          const x1 = centerX + radius * Math.cos(startAngleRad);
          const y1 = centerY + radius * Math.sin(startAngleRad);
          const x2 = centerX + radius * Math.cos(endAngleRad);
          const y2 = centerY + radius * Math.sin(endAngleRad);

          const largeArcFlag = slice.endAngle - slice.startAngle > 180 ? 1 : 0;

          const pathData = [
            `M ${centerX} ${centerY}`,
            `L ${x1} ${y1}`,
            `A ${radius} ${radius} 0 ${largeArcFlag} 1 ${x2} ${y2}`,
            "Z",
          ].join(" ");

          return (
            <path
              key={index}
              d={pathData}
              fill={slice.color}
              stroke={colors.background.paper}
              strokeWidth="2"
            />
          );
        })}
        {/* Center circle for donut effect */}
        <circle
          cx={centerX}
          cy={centerY}
          r="40"
          fill={colors.background.paper}
        />
      </svg>

      <Box>
        {pieData.map((slice) => (
          <Box
            key={slice.name}
            sx={{ display: "flex", alignItems: "center", mb: 1 }}
          >
            <Box
              sx={{
                width: 12,
                height: 12,
                bgcolor: slice.color,
                borderRadius: 1,
                mr: 1,
              }}
            />
            <Typography variant="body2" sx={{ color: colors.text.primary }}>
              {slice.name}: {formatNumber(slice.value)} (
              {slice.percentage.toFixed(1)}%)
            </Typography>
          </Box>
        ))}
      </Box>
    </Box>
  );
};
