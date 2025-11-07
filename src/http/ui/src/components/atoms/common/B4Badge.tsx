import { Chip, ChipProps } from "@mui/material";
import { colors } from "@design";

type BadgeVariant =
  | "success"
  | "error"
  | "warning"
  | "info"
  | "primary"
  | "secondary";

interface B4BadgeProps extends Omit<ChipProps, "color" | "variant"> {
  badgeVariant?: BadgeVariant;
}

const variantStyles: Record<BadgeVariant, object> = {
  success: { bgcolor: "#4caf5033", color: "#4caf50", borderColor: "#4caf50" },
  error: {
    bgcolor: `${colors.quaternary}22`,
    color: colors.quaternary,
    borderColor: colors.quaternary,
  },
  warning: { bgcolor: "#ff980033", color: "#ff9800", borderColor: "#ff9800" },
  info: {
    bgcolor: colors.accent.tertiary,
    color: colors.tertiary,
    borderColor: colors.tertiary,
  },
  primary: {
    bgcolor: colors.accent.primary,
    borderColor: colors.primary,
  },
  secondary: {
    bgcolor: colors.accent.secondary,
    borderColor: colors.secondary,
  },
};

export const B4Badge: React.FC<B4BadgeProps> = ({
  badgeVariant = "primary",
  sx,
  ...props
}) => (
  <Chip
    size="small"
    sx={{
      ...variantStyles[badgeVariant],
      ...sx,
    }}
    {...props}
  />
);
