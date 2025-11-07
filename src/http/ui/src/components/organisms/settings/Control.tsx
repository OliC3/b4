import React, { useState } from "react";
import { Button, Grid, Stack } from "@mui/material";
import SettingSection from "@molecules/common/B4Section";
import {
  RestartAlt as RestartIcon,
  ToggleOn as ToggleOnIcon,
  Hub as ControlIcon,
  Restore as RestoreIcon,
} from "@mui/icons-material";
import B4Config from "@models/Config";
import { RestartDialog } from "./RestartDialog";
import { colors } from "@design";
import { ResetDialog } from "./ResetDialog";

interface ControlSettingsProps {
  config: B4Config;
  onChange: (field: string, value: boolean) => void;
  loadConfig: () => void;
}

export const ControlSettings: React.FC<ControlSettingsProps> = ({
  config,
  onChange,
  loadConfig,
}) => {
  const [saving, setSaving] = useState(false);
  const [showRestartDialog, setShowRestartDialog] = useState(false);
  const [showResetDialog, setShowResetDialog] = useState(false);

  const handleResetSuccess = () => {
    loadConfig();
  };

  return (
    <SettingSection
      title="Core Controls"
      description="Control core service and config operations"
      icon={<ControlIcon />}
    >
      <Grid container spacing={2}>
        <Stack spacing={2}>
          <Button
            size="small"
            variant="outlined"
            startIcon={<RestartIcon />}
            onClick={() => setShowRestartDialog(true)}
            disabled={saving}
            sx={{
              borderColor: colors.secondary,
              color: colors.secondary,
              "&:hover": {
                borderColor: colors.primary,
                bgcolor: colors.accent.primaryHover,
              },
            }}
          >
            Restart B4 System Service
          </Button>
          <Button
            size="small"
            variant="outlined"
            startIcon={<RestoreIcon />}
            onClick={() => setShowResetDialog(true)}
            disabled={saving}
            sx={{
              borderColor: colors.primary,
              color: colors.primary,
              "&:hover": {
                borderColor: "#d32f2f",
                bgcolor: `${colors.primary}22`,
              },
            }}
          >
            Reset the configuration to default settings
          </Button>
        </Stack>
      </Grid>
      {/* Restart Dialog */}
      <RestartDialog
        open={showRestartDialog}
        onClose={() => setShowRestartDialog(false)}
      />

      {/* Reset Dialog */}
      <ResetDialog
        open={showResetDialog}
        onClose={() => setShowResetDialog(false)}
        onSuccess={handleResetSuccess}
      />
    </SettingSection>
  );
};
