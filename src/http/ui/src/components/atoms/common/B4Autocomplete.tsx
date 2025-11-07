import { Autocomplete, CircularProgress, IconButton, Box } from "@mui/material";
import { Add as AddIcon } from "@mui/icons-material";
import B4TextField from "./B4TextField";
import { colors } from "@design";

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
  const handleAdd = () => {
    if (value.trim() && onSelect) {
      onSelect(value.trim());
      onChange("");
    }
  };

  return (
    <Box sx={{ display: "flex", gap: 1, alignItems: "flex-start" }}>
      <Autocomplete
        fullWidth
        value={value}
        inputValue={value}
        onChange={(_, newValue) => {
          if (newValue && onSelect) {
            onSelect(newValue);
          }
        }}
        onInputChange={(_, newValue, reason) => {
          if (reason === "input") {
            onChange(newValue);
          }
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
              if ((e.key === "Enter" || e.key === "Tab") && value.trim()) {
                e.preventDefault();
                handleAdd();
              }
            }}
            slotProps={{
              input: {
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
                  bgcolor: colors.background.dark,
                  "&:hover": {
                    borderColor: colors.primary,
                  },
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
                  borderColor: colors.secondary,
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
          disabled={!value.trim() || disabled}
          sx={{
            bgcolor: colors.accent.secondary,
            color: colors.secondary,
            "&:hover": {
              bgcolor: colors.accent.secondaryHover,
            },
            "&:disabled": {
              bgcolor: colors.accent.secondaryHover,
              color: colors.accent.secondary,
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
