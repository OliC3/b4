import { Box, Typography } from "@mui/material";
import { colors } from "@design";
import DecryptedText from "@atoms/ReactBits/DecryptedText";

export default function Logo() {
  return (
    <Box sx={{ display: "flex", flexDirection: "column", gap: 0 }}>
      <Typography
        variant="h4"
        component="div"
        sx={{
          fontWeight: 800,
          color: colors.secondary,
          letterSpacing: "-0.08em",
          lineHeight: 1,
          background: `linear-gradient(135deg, ${colors.secondary} 0%, ${colors.primary} 100%)`,
          WebkitBackgroundClip: "text",
          WebkitTextFillColor: "transparent",
          backgroundClip: "text",
        }}
      >
        B<sup style={{ fontSize: "0.5em" }}>4</sup>
      </Typography>

      <Typography
        variant="caption"
        component="div"
        sx={{
          fontSize: "0.65rem",
          color: colors.text.secondary,
          opacity: 0.7,
          letterSpacing: "0.15em",
          textTransform: "uppercase",
          mt: -0.5,
        }}
      >
        <DecryptedText text="Bye Bye Big Bro" />
      </Typography>
    </Box>
  );
}
