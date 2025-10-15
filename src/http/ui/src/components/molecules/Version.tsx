import { Box, Typography } from "@mui/material";
import { colors } from "../../Theme";

export default function Version() {
  return (
    <Box
      sx={{
        py: 2,
        textAlign: "center",
      }}
    >
      <Typography
        variant="caption"
        sx={{
          color: colors.secondary,
        }}
      >
        Version 1.0.0
      </Typography>
    </Box>
  );
}
