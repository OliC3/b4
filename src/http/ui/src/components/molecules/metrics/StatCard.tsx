import { B4Card } from "@atoms/common/B4Card";
import { Box, Stack, Typography } from "@mui/material";
import { colors, spacing } from "@design";

interface StatCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  icon: React.ReactNode;
  color?: string;
  variant?: "default" | "outlined" | "elevated";
  onClick?: () => void;
  trend?: {
    value: number;
    label?: string;
  };
}

export const StatCard: React.FC<StatCardProps> = ({
  title,
  value,
  subtitle,
  icon,
  color = colors.primary,
  variant = "outlined",
  onClick,
  trend,
}) => (
  <B4Card
    variant={variant}
    sx={{
      border: `1px solid ${color}33`,
      cursor: onClick ? "pointer" : "default",
      transition: "all 0.2s ease",
      "&:hover": onClick
        ? {
            borderColor: `${color}66`,
            boxShadow: `0 0 20px ${color}22`,
            transform: "translateY(-2px)",
          }
        : {
            borderColor: `${color}66`,
            boxShadow: `0 0 20px ${color}22`,
          },
    }}
    onClick={onClick}
  >
    <Box sx={{ p: spacing.lg }}>
      <Stack
        direction="row"
        justifyContent="space-between"
        alignItems="flex-start"
      >
        <Box sx={{ flex: 1 }}>
          <Typography
            variant="caption"
            sx={{
              color: colors.text.secondary,
              textTransform: "uppercase",
              letterSpacing: "0.5px",
            }}
          >
            {title}
          </Typography>
          <Typography
            variant="h4"
            sx={{
              color: colors.text.primary,
              fontWeight: 600,
              mt: 0.5,
              mb: 0.5,
            }}
          >
            {value}
          </Typography>
          {subtitle && (
            <Typography variant="caption" sx={{ color: colors.text.secondary }}>
              {subtitle}
            </Typography>
          )}
          {trend && (
            <Box
              sx={{ display: "flex", alignItems: "center", gap: 0.5, mt: 0.5 }}
            >
              <Typography
                variant="caption"
                sx={{
                  color: trend.value > 0 ? "#4caf50" : "#f44336",
                  fontWeight: 600,
                }}
              >
                {trend.value > 0 ? "+" : ""}
                {trend.value.toFixed(1)}%
              </Typography>
              {trend.label && (
                <Typography
                  variant="caption"
                  sx={{ color: colors.text.secondary }}
                >
                  {trend.label}
                </Typography>
              )}
            </Box>
          )}
        </Box>
        <Box
          sx={{
            p: 1.5,
            borderRadius: spacing.md,
            bgcolor: `${color}22`,
            color,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            minWidth: 56,
            minHeight: 56,
          }}
        >
          {icon}
        </Box>
      </Stack>
    </Box>
  </B4Card>
);
