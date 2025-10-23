import React from "react";
import { TextField, TextFieldProps } from "@mui/material";
import { colors } from "../../../Theme";

interface B4TextFieldProps extends Omit<TextFieldProps, "variant"> {
  helperText?: string;
}

export const B4TextField: React.FC<B4TextFieldProps> = ({
  helperText,
  ...props
}) => {
  return (
    <TextField
      {...props}
      variant="outlined"
      size="small"
      fullWidth
      helperText={helperText}
      sx={{
        "& .MuiOutlinedInput-root": {
          bgcolor: colors.background.dark,
          "& fieldset": {
            borderColor: colors.border.default,
          },
          "&:hover fieldset": {
            borderColor: colors.border.medium,
          },
          "&.Mui-focused fieldset": {
            borderColor: colors.secondary,
          },
        },
        "& .MuiFormHelperText-root": {
          ml: 0.1,
        },
        ...props.sx,
      }}
    />
  );
};

export default B4TextField;
