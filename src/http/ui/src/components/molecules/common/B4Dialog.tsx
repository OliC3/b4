import React from "react";
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  DialogProps,
  Stack,
  Box,
  Typography,
  Divider,
} from "@mui/material";
import { colors, spacing, radius } from "@design";

interface B4DialogProps extends Omit<DialogProps, "title"> {
  title: string;
  subtitle?: string;
  icon?: React.ReactNode;
  actions?: React.ReactNode;
  onClose: () => void;
}

export const B4Dialog: React.FC<B4DialogProps> = ({
  title,
  subtitle,
  icon,
  children,
  actions,
  onClose,
  ...props
}) => (
  <Dialog
    onClose={onClose}
    slotProps={{
      paper: {
        sx: {
          bgcolor: colors.background.paper,
          border: `2px solid ${colors.border.default}`,
          borderRadius: radius.md,
        },
      },
    }}
    {...props}
  >
    <DialogTitle
      sx={{
        bgcolor: colors.background.dark,
        color: colors.text.primary,
        borderBottom: `1px solid ${colors.border.default}`,
      }}
    >
      <Stack direction="row" alignItems="center" spacing={spacing.md}>
        {icon && (
          <Box
            sx={{
              p: 1.5,
              borderRadius: radius.md,
              bgcolor: colors.accent.secondary,
              color: colors.secondary,
              display: "flex",
            }}
          >
            {icon}
          </Box>
        )}
        <Box>
          <Typography sx={{ mt: 1.5, lineHeight: 0, mb: subtitle ? 0 : 1.5 }}>
            {title}
          </Typography>
          {subtitle && (
            <Typography variant="caption" sx={{ color: colors.text.secondary }}>
              {subtitle}
            </Typography>
          )}
        </Box>
      </Stack>
    </DialogTitle>

    <DialogContent sx={{ mt: spacing.md }}>{children}</DialogContent>

    {actions && (
      <>
        <Divider sx={{ borderColor: colors.border.default }} />
        <DialogActions sx={{ p: spacing.md }}>{actions}</DialogActions>
      </>
    )}
  </Dialog>
);
