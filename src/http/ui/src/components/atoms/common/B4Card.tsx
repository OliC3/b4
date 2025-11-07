import { Card, CardProps } from "@mui/material";
import { colors, radius } from "@design";

interface B4CardProps extends Omit<CardProps, "variant"> {
  variant?: "default" | "outlined" | "elevated";
}

export const B4Card: React.FC<B4CardProps> = ({
  variant = "outlined",
  children,
  sx,
  ...props
}) => {
  const variants = {
    default: {
      bgcolor: colors.background.paper,
      border: "none",
    },
    outlined: {
      bgcolor: colors.background.paper,
      border: `1px solid ${colors.border.default}`,
    },
    elevated: {
      bgcolor: colors.background.paper,
      boxShadow: `0 0 20px ${colors.primary}22`,
      border: `1px solid ${colors.border.default}`,
    },
  };

  return (
    <Card
      elevation={0}
      sx={{
        ...variants[variant],
        borderRadius: radius.md,
        ...sx,
      }}
      {...props}
    >
      {children}
    </Card>
  );
};
