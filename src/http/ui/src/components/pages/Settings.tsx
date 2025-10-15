import { Container, Paper, Typography, Stack } from "@mui/material";

export default function Settings() {
  return (
    <Container
      maxWidth="md"
      sx={{
        flex: 1,
        py: 3,
        px: 3,
        display: "flex",
        flexDirection: "column",
        overflow: "auto",
      }}
    >
      <Paper elevation={0} variant="outlined" sx={{ p: 4 }}>
        <Typography variant="h5" gutterBottom sx={{ color: "#F5AD18", mb: 3 }}>
          Settings
        </Typography>

        <Stack spacing={3}></Stack>
        <Typography>Under Development...</Typography>
      </Paper>
    </Container>
  );
}
