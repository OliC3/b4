import React, { useState } from "react";
import {
  TextField,
  Autocomplete,
  CircularProgress,
  IconButton,
  Box,
} from "@mui/material";
import { Add as AddIcon } from "@mui/icons-material";
import B4TextField from "./B4TextField";
import { colors } from "../../../Theme";

interface SettingAutocompleteProps {
  label: string;
  value: string;
  options: string[];
  onChange: (value: string) => void;
  onSelect?: (value: string) => void;
  loading?: boolean;
  placeholder?: string;
  helperText?: string;
  disabled?: boolean;
}

const SettingAutocomplete: React.FC<SettingAutocompleteProps> = ({
  label,
  value,
  options,
  onChange,
  onSelect,
  loading = false,
  placeholder,
  helperText,
  disabled = false,
}) => {
  const [inputValue, setInputValue] = useState("");

  const handleAdd = () => {
    if (inputValue.trim() && onSelect) {
      onSelect(inputValue.trim());
      setInputValue("");
      onChange("");
    }
  };

  return (
    <Box sx={{ display: "flex", gap: 1, alignItems: "flex-start" }}>
      <Autocomplete
        fullWidth
        value={value}
        inputValue={inputValue}
        onInputChange={(_, newValue) => {
          setInputValue(newValue);
          onChange(newValue);
        }}
        options={options}
        loading={loading}
        disabled={disabled}
        freeSolo
        renderInput={(params) => (
          <B4TextField
            {...params}
            label={label}
            placeholder={placeholder}
            helperText={helperText}
            onKeyDown={(e) => {
              if ((e.key === "Enter" || e.key === "Tab") && inputValue.trim()) {
                e.preventDefault();
                handleAdd();
              }
            }}
            InputProps={{
              ...params.InputProps,
              endAdornment: (
                <>
                  {loading ? (
                    <CircularProgress color="inherit" size={20} />
                  ) : null}
                  {params.InputProps.endAdornment}
                </>
              ),
              sx: {
                bgcolor: colors.background.paper,
                "&:hover": {
                  bgcolor: colors.background.paper,
                },
              },
            }}
            sx={{
              "& .MuiOutlinedInput-root": {
                "& fieldset": {
                  borderColor: colors.border.default,
                },
                "&:hover fieldset": {
                  borderColor: colors.border.default,
                },
                "&.Mui-focused fieldset": {
                  borderColor: colors.primary,
                },
              },
              "& .MuiInputLabel-root": {
                color: colors.text.secondary,
                "&.Mui-focused": {
                  color: colors.primary,
                },
              },
            }}
          />
        )}
      />
      {onSelect && (
        <IconButton
          onClick={handleAdd}
          disabled={!inputValue.trim() || disabled}
          sx={{
            bgcolor: colors.accent.secondary,
            color: colors.secondary,
            "&:hover": {
              bgcolor: colors.accent.secondaryHover,
            },
            "&:disabled": {
              bgcolor: colors.background.paper,
              color: colors.text.primary,
            },
          }}
        >
          <AddIcon />
        </IconButton>
      )}
    </Box>
  );
};

export default SettingAutocomplete;
