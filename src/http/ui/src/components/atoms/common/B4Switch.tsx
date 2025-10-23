import React from "react";
import {
  FormControlLabel,
  Switch,
  SwitchProps,
  Typography,
  Box,
} from "@mui/material";
import { colors } from "../../../Theme";

interface B4SwitchProps extends Omit<SwitchProps, "checked" | "onChange"> {
  label: string;
  checked: boolean;
  description?: string;
  onChange: (checked: boolean) => void;
}

export const B4Switch: React.FC<B4SwitchProps> = ({
  label,
  checked,
  description,
  onChange,
  disabled,
  ...props
}) => {
  return (
    <Box>
      <FormControlLabel
        control={
          <Switch
            checked={checked}
            onChange={(e) => onChange(e.target.checked)}
            disabled={disabled}
            sx={{
              "& .MuiSwitch-switchBase.Mui-checked": {
                color: colors.secondary,
              },
              "& .MuiSwitch-switchBase.Mui-checked + .MuiSwitch-track": {
                backgroundColor: colors.secondary,
              },
            }}
            {...props}
          />
        }
        label={
          <Typography sx={{ color: colors.text.primary, fontWeight: 500 }}>
            {label}
          </Typography>
        }
      />
      {description && (
        <Typography
          variant="caption"
          sx={{
            display: "block",
            color: colors.text.secondary,
            ml: 6,
            mt: -1,
          }}
        >
          {description}
        </Typography>
      )}
    </Box>
  );
};

export default B4Switch;
