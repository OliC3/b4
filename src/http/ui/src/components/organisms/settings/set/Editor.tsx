import React, { useState } from "react";
import { Box, Stack, Button, Tabs, Tab, Paper } from "@mui/material";
import {
  Settings as SettingsIcon,
  Security as SecurityIcon,
  Dns as DnsIcon,
  CallSplit as FragIcon,
  Language as DomainIcon,
  Layers as LayersIcon,
} from "@mui/icons-material";

import { B4Dialog } from "@molecules/common/B4Dialog";
import B4TextField from "@atoms/common/B4TextField";

import { colors, button_primary, button_secondary } from "@design";
import { B4SetConfig, SystemConfig } from "@models/Config";

import { DomainSettings } from "@/components/organisms/settings/set/Domain";
import { TcpSettings } from "./Tcp";
import { UdpSettings } from "./Udp";
import { FragmentationSettings } from "./Fragmentation";
import { FakingSettings } from "./Faking";

export interface SetEditorProps {
  open: boolean;
  settings: SystemConfig;
  set: B4SetConfig | null;
  isNew: boolean;
  onClose: () => void;
  onSave: (set: B4SetConfig) => void;
}

export const SetEditor: React.FC<SetEditorProps> = ({
  open,
  set: initialSet,
  isNew,
  settings,
  onClose,
  onSave,
}) => {
  enum TABS {
    DOMAINS = 0,
    TCP,
    UDP,
    FRAGMENTATION,
    FAKING,
  }

  const [activeTab, setActiveTab] = useState<TABS>(TABS.DOMAINS);
  const [editedSet, setEditedSet] = useState<B4SetConfig | null>(initialSet);

  React.useEffect(() => {
    setEditedSet(initialSet);
    setActiveTab(0);
  }, [initialSet]);

  const handleChange = (
    field: string,
    value: string | number | boolean | string[] | null | undefined
  ) => {
    if (!editedSet) return;

    const keys = field.split(".");

    if (keys.length === 1) {
      setEditedSet({ ...editedSet, [field]: value });
    } else {
      const newConfig = { ...editedSet };
      let current: Record<string, unknown> = newConfig;

      for (let i = 0; i < keys.length - 1; i++) {
        current[keys[i]] = { ...(current[keys[i]] as object) };
        current = current[keys[i]] as Record<string, unknown>;
      }

      current[keys[keys.length - 1]] = value;
      setEditedSet(newConfig);
    }
  };

  const handleSave = () => {
    if (editedSet) {
      onSave(editedSet);
    }
  };

  if (!editedSet) return null;

  const dialogContent = (
    <Stack spacing={3} sx={{ mt: 2 }}>
      <Paper
        elevation={0}
        sx={{
          bgcolor: colors.background.paper,
          borderRadius: 2,
          border: `1px solid ${colors.border.default}`,
        }}
      >
        <Box sx={{ mt: 2, p: 3 }}>
          <B4TextField
            label="Set Name"
            value={editedSet.name}
            onChange={(e) => handleChange("name", e.target.value)}
            placeholder="e.g., YouTube Bypass, Gaming, Streaming"
            helperText="Give this set a descriptive name"
            required
          />
        </Box>
        {/* Configuration Tabs */}
        <Tabs
          value={activeTab}
          onChange={(_, v: number) => setActiveTab(v)}
          variant="scrollable"
          scrollButtons="auto"
          sx={{
            borderBottom: `1px solid ${colors.border.light}`,
            "& .MuiTab-root": {
              color: colors.text.secondary,
              textTransform: "none",
              minHeight: 48,
              "&.Mui-selected": {
                color: colors.secondary,
              },
            },
            "& .MuiTabs-indicator": {
              bgcolor: colors.secondary,
            },
          }}
        >
          <Tab label="Domains" icon={<DomainIcon />} />
          <Tab label="TCP" icon={<SettingsIcon />} />
          <Tab label="UDP" icon={<DnsIcon />} />
          <Tab label="Fragmentation" icon={<FragIcon />} />
          <Tab label="Faking" icon={<SecurityIcon />} />
        </Tabs>
      </Paper>
      <Box>
        {/* TCP Settings */}
        <Box hidden={activeTab !== TABS.TCP}>
          <Stack spacing={2}>
            <TcpSettings config={editedSet} onChange={handleChange} />
          </Stack>
        </Box>

        {/* UDP Settings */}
        <Box hidden={activeTab !== TABS.UDP}>
          <Stack spacing={2}>
            <UdpSettings config={editedSet} onChange={handleChange} />
          </Stack>
        </Box>

        {/* Fragmentation Settings */}
        <Box hidden={activeTab !== TABS.FRAGMENTATION}>
          <Stack spacing={2}>
            <FragmentationSettings config={editedSet} onChange={handleChange} />
          </Stack>
        </Box>

        {/* Faking Settings */}
        <Box hidden={activeTab !== TABS.FAKING}>
          <Stack spacing={2}>
            <FakingSettings config={editedSet} onChange={handleChange} />
          </Stack>
        </Box>

        {/* Domain Settings */}
        <Box hidden={activeTab !== TABS.DOMAINS}>
          <Stack spacing={2}>
            <DomainSettings
              geo={settings.geo}
              config={editedSet}
              onChange={handleChange}
            />
          </Stack>
        </Box>
      </Box>
    </Stack>
  );

  return (
    <B4Dialog
      title={isNew ? "Create New Set" : `Edit Set: ${editedSet.name}`}
      open={open}
      onClose={onClose}
      icon={<LayersIcon />}
      fullWidth={true}
      maxWidth="lg"
      actions={
        <>
          <Button onClick={onClose} sx={{ ...button_secondary }}>
            Cancel
          </Button>
          <Box sx={{ flex: 1 }} />
          <Button
            variant="contained"
            onClick={handleSave}
            disabled={!editedSet.name.trim()}
            sx={{ ...button_primary }}
          >
            {isNew ? "Create Set" : "Save Changes"}
          </Button>
        </>
      }
    >
      {dialogContent}
    </B4Dialog>
  );
};
