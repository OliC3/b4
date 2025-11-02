import React from "react";
import {
  Box,
  Stack,
  TextField,
  Chip,
  FormControlLabel,
  Switch,
  Typography,
  IconButton,
} from "@mui/material";
import RefreshIcon from "@mui/icons-material/DeleteForever";
import { colors } from "../../../Theme";

interface DomainsControlBarProps {
  filter: string;
  onFilterChange: (filter: string) => void;
  totalCount: number;
  filteredCount: number;
  sortColumn: string | null;
  paused: boolean;
  onPauseChange: (paused: boolean) => void;
  onClearSort: () => void;
  onReset: () => void;
}

export const DomainsControlBar: React.FC<DomainsControlBarProps> = ({
  filter,
  onFilterChange,
  totalCount,
  filteredCount,
  sortColumn,
  paused,
  onPauseChange,
  onClearSort,
  onReset,
}) => {
  return (
    <Box
      sx={{
        p: 2,
        borderBottom: "1px solid",
        borderColor: colors.border.light,
        bgcolor: colors.background.control,
      }}
    >
      <Stack direction="row" spacing={2} alignItems="center">
        <TextField
          size="small"
          placeholder="Filter entries (use `+` to combine, e.g. `tcp+domain2`, or `tcp+domain:exmpl1+domain:exmpl2`)"
          value={filter}
          onChange={(e) => onFilterChange(e.target.value)}
          sx={{ flex: 1 }}
          InputProps={{
            sx: {
              bgcolor: colors.background.dark,
              "& fieldset": {
                borderColor: `${colors.border.default} !important`,
              },
            },
          }}
        />
        <Stack direction="row" spacing={1} alignItems="center">
          <Chip
            label={`${totalCount} connections`}
            size="small"
            sx={{
              bgcolor: colors.accent.secondary,
              color: colors.secondary,
              fontWeight: 600,
            }}
          />
          {filter && (
            <Chip
              label={`${filteredCount} filtered`}
              size="small"
              sx={{
                bgcolor: colors.accent.primary,
                color: colors.primary,
                borderColor: colors.primary,
              }}
              variant="outlined"
            />
          )}
          {sortColumn && (
            <Chip
              label={`Sorted by ${sortColumn}`}
              size="small"
              onDelete={onClearSort}
              sx={{
                bgcolor: colors.accent.tertiary,
                color: colors.tertiary,
                borderColor: colors.tertiary,
              }}
              variant="outlined"
            />
          )}
        </Stack>
        <FormControlLabel
          control={
            <Switch
              checked={paused}
              onChange={(e) => onPauseChange(e.target.checked)}
              sx={{
                "& .MuiSwitch-switchBase.Mui-checked": {
                  color: colors.secondary,
                },
                "& .MuiSwitch-switchBase.Mui-checked + .MuiSwitch-track": {
                  backgroundColor: colors.secondary,
                },
              }}
            />
          }
          label={
            <Typography
              sx={{
                color: paused ? colors.secondary : "text.secondary",
                fontWeight: paused ? 600 : 400,
              }}
            >
              {paused ? "Paused" : "Streaming"}
            </Typography>
          }
        />
        <IconButton
          color="inherit"
          onClick={onReset}
          sx={{
            color: "text.secondary",
            "&:hover": {
              color: colors.secondary,
              bgcolor: colors.accent.secondaryHover,
            },
          }}
        >
          <RefreshIcon />
        </IconButton>
      </Stack>
    </Box>
  );
};
