import { Container, Alert, Stack } from "@mui/material";
import { TestRunner } from "@organisms/check/Runner";
import { colors } from "@design";
import { Link as RouterLink } from "react-router-dom";

export default function Test() {
  return (
    <Container
      maxWidth={false}
      sx={{
        height: "100%",
        display: "flex",
        flexDirection: "column",
        overflow: "auto",
        py: 3,
      }}
    >
      <Stack spacing={3}>
        <Alert
          severity="info"
          sx={{
            bgcolor: colors.accent.primary,
            border: `1px solid ${colors.secondary}44`,
          }}
        >
          <strong>About this test:</strong> This test measures download speeds
          from your configured domains to verify DPI bypass is working
          correctly. It tests all domains from your manual SNI domains list plus
          any additional domains specified in the checker configuration. Higher
          speeds indicate successful bypass - if speeds are slow or connections
          fail, your DPI circumvention may need adjustment. Configure domains to
          test in{" "}
          <RouterLink color="primary" to="/settings/checker">
            Testing Settings
          </RouterLink>
          .
        </Alert>
        <TestRunner />
      </Stack>
    </Container>
  );
}
