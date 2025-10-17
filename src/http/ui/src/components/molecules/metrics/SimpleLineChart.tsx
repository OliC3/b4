// src/http/ui/src/components/molecules/SimpleLineChart.tsx
import React from "react";
import { Box, Typography } from "@mui/material";
import { colors } from "../../../Theme";

interface SimpleChartProps {
  data: Array<{ timestamp: number; value: number }>;
  height?: number;
  color?: string;
}

export const SimpleLineChart: React.FC<SimpleChartProps> = ({
  data,
  height = 200,
  color = colors.secondary,
}) => {
  if (data.length === 0) return <Typography>No data</Typography>;

  const maxValue = Math.max(...data.map((d) => d.value));
  const minValue = Math.min(...data.map((d) => d.value));
  const range = maxValue - minValue || 1;

  // Calculate SVG points
  const width = 100;
  const points = data
    .map((d, i) => {
      const x = (i / (data.length - 1)) * width;
      const y = height - ((d.value - minValue) / range) * height;
      return `${x},${y}`;
    })
    .join(" ");

  return (
    <Box sx={{ position: "relative", width: "100%", height }}>
      <svg
        width="100%"
        height={height}
        viewBox={`0 0 ${width} ${height}`}
        preserveAspectRatio="none"
      >
        {/* Grid lines */}
        {[0, 0.25, 0.5, 0.75, 1].map((y) => (
          <line
            key={y}
            x1="0"
            y1={y * height}
            x2={width}
            y2={y * height}
            stroke={colors.border.light}
            strokeWidth="0.5"
          />
        ))}

        {/* Area under line */}
        <polygon
          points={`0,${height} ${points} ${width},${height}`}
          fill={color}
          fillOpacity="0.1"
        />

        {/* Line */}
        <polyline points={points} fill="none" stroke={color} strokeWidth="2" />
      </svg>

      {/* Y-axis labels */}
      <Box
        sx={{
          position: "absolute",
          top: 0,
          left: -40,
          height: "100%",
          display: "flex",
          flexDirection: "column",
          justifyContent: "space-between",
        }}
      >
        <Typography variant="caption" sx={{ color: colors.text.secondary }}>
          {maxValue.toFixed(1)}
        </Typography>
        <Typography variant="caption" sx={{ color: colors.text.secondary }}>
          {minValue.toFixed(1)}
        </Typography>
      </Box>
    </Box>
  );
};
