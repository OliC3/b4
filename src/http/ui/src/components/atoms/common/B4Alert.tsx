import { Alert, AlertProps } from "@mui/material";
import { colors } from "@design";

export const B4Alert: React.FC<AlertProps> = ({ sx, ...props }) => (
  <Alert
    sx={{
      bgcolor: colors.background.default,
      border: `1px solid ${colors.border.default}`,
      ...sx,
    }}
    {...props}
  />
);
