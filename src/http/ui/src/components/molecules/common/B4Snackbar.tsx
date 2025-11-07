import { Snackbar, Alert } from "@mui/material";

export interface B4SnackbarProps {
  open: boolean;
  message: string;
  severity: "error" | "warning" | "info" | "success";
}

export function B4Snackbar({ ...snackbar }: Readonly<B4SnackbarProps>) {
  function handleClose() {
    snackbar.open = false;
  }

  return (
    <Snackbar
      open={snackbar.open}
      autoHideDuration={6000}
      onClose={handleClose}
      anchorOrigin={{ vertical: "bottom", horizontal: "right" }}
    >
      <Alert
        onClose={handleClose}
        severity={snackbar.severity}
        sx={{ width: "100%" }}
      >
        {snackbar.message}
      </Alert>
    </Snackbar>
  );
}
