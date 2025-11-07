import { Box, Grid, Typography } from "@mui/material";
import { colors, spacing } from "@design";
import React from "react";

interface B4FormGroupProps {
  label: string;
  description?: string;
  icon?: React.ReactNode;
  children: React.ReactNode;
  columns?: 1 | 2;
}

export const B4FormGroup: React.FC<B4FormGroupProps> = ({
  label,
  description,
  icon,
  children,
  columns = 2,
}) => (
  <Box sx={{ mb: spacing.lg }}>
    <Box sx={{ display: "flex", alignItems: "center", gap: 1, mb: spacing.md }}>
      {icon}
      <Box>
        <Typography variant="h6" sx={{ color: colors.text.primary }}>
          {label}
        </Typography>
        {description && (
          <Typography variant="caption" sx={{ color: colors.text.secondary }}>
            {description}
          </Typography>
        )}
      </Box>
    </Box>

    <Grid container spacing={spacing.md}>
      {React.Children.map(children, (child) => (
        <Grid size={{ xs: 12, md: 12 / columns }}>{child}</Grid>
      ))}
    </Grid>
  </Box>
);
