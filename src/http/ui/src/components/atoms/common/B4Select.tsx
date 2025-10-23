import React from "react";
import {
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  SelectProps,
  FormHelperText,
} from "@mui/material";
import { colors } from "../../../Theme";

interface B4SelectProps extends Omit<SelectProps<string | number>, "variant"> {
  label: string;
  options: Array<{ value: string | number; label: string }>;
  helperText?: string;
}

export const B4Select: React.FC<B4SelectProps> = ({
  label,
  options,
  helperText,
  ...props
}) => {
  return (
    <FormControl fullWidth size="small">
      <InputLabel sx={{ color: colors.text.secondary }}>{label}</InputLabel>
      <Select
        {...props}
        label={label}
        sx={{
          bgcolor: colors.background.dark,
          "& .MuiOutlinedInput-notchedOutline": {
            borderColor: colors.border.default,
          },
          "&:hover .MuiOutlinedInput-notchedOutline": {
            borderColor: colors.border.medium,
          },
          "&.Mui-focused .MuiOutlinedInput-notchedOutline": {
            borderColor: colors.secondary,
          },

          ...props.sx,
        }}
      >
        {options.map((option) => (
          <MenuItem key={option.value} value={option.value}>
            {option.label}
          </MenuItem>
        ))}
      </Select>
      {helperText && (
        <FormHelperText sx={{ color: colors.text.secondary, ml: 0.1 }}>
          {helperText}
        </FormHelperText>
      )}
    </FormControl>
  );
};

export default B4Select;
