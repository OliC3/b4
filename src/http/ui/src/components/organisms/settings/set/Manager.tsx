import React, { useState } from "react";
import {
  Box,
  Grid,
  Stack,
  Button,
  Typography,
  Alert,
  Chip,
  List,
  ListItem,
  ListItemText,
  IconButton,
  Collapse,
  Divider,
} from "@mui/material";
import {
  Add as AddIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  ContentCopy as CopyIcon,
  Star as StarIcon,
  ExpandMore as ExpandIcon,
  ExpandLess as CollapseIcon,
  Layers as LayersIcon,
  Warning as WarningIcon,
} from "@mui/icons-material";
import { v4 as uuidv4 } from "uuid";

import B4Section from "@molecules/common/B4Section";
import { B4Dialog } from "@molecules/common/B4Dialog";
import { B4Badge } from "@atoms/common/B4Badge";
import { SetEditor } from "./Editor";

import { colors, radius, button_secondary } from "@design";
import { B4Config, B4SetConfig } from "@models/Config";

interface SetsManagerProps {
  config: B4Config;
  onChange: (
    field: string,
    value: boolean | string | number | B4SetConfig[]
  ) => void;
}

export const SetsManager: React.FC<SetsManagerProps> = ({
  config,
  onChange,
}) => {
  const [expandedSet, setExpandedSet] = useState<string | null>(null);
  const [editDialog, setEditDialog] = useState<{
    open: boolean;
    set: B4SetConfig | null;
    isNew: boolean;
  }>({
    open: false,
    set: null,
    isNew: false,
  });
  const [deleteDialog, setDeleteDialog] = useState<{
    open: boolean;
    setId: string | null;
  }>({
    open: false,
    setId: null,
  });

  const mainSetId = "11111111-1111-1111-1111-111111111111";
  const sets = config.sets || [];

  const handleAddSet = () => {
    const newSet: B4SetConfig = {
      id: uuidv4(),
      name: `Set ${sets.length + 1}`,
      tcp: { conn_bytes_limit: 19, seg2delay: 0 },
      udp: {
        mode: "fake",
        fake_seq_length: 6,
        fake_len: 64,
        faking_strategy: "none",
        dport_min: 0,
        dport_max: 0,
        filter_quic: "disabled",
        filter_stun: true,
        conn_bytes_limit: 8,
      },
      fragmentation: {
        strategy: "tcp",
        sni_reverse: true,
        middle_sni: true,
        sni_position: 1,
      },
      faking: {
        sni: true,
        ttl: 8,
        strategy: "pastseq",
        seq_offset: 10000,
        sni_seq_length: 1,
        sni_type: 2,
        custom_payload: "",
      },
      domains: {
        sni_domains: [],
        geosite_categories: [],
        geoip_categories: [],
        block_domains: [],
        block_geosite_categories: [],
      },
    };

    setEditDialog({ open: true, set: newSet, isNew: true });
  };

  const handleEditSet = (set: B4SetConfig) => {
    setEditDialog({ open: true, set, isNew: false });
  };

  const handleSaveSet = (set: B4SetConfig) => {
    const isNew = !sets.find((s) => s.id === set.id);

    if (isNew) {
      sets.push(set);
    }
    onChange("sets", sets);

    setEditDialog({ open: false, set: null, isNew: false });
  };

  const handleDeleteSet = () => {
    if (deleteDialog.setId) {
      onChange(
        "sets",
        sets.filter((s) => s.id !== deleteDialog.setId)
      );
      // If deleting main set, set first remaining as main
      if (mainSetId === deleteDialog.setId && sets.length > 1) {
        const newMain = sets.find((s) => s.id !== deleteDialog.setId);
        if (newMain) {
          onChange("sets", sets);
        }
      }
    }
    setDeleteDialog({ open: false, setId: null });
  };

  const handleDuplicateSet = (set: B4SetConfig) => {
    const duplicated: B4SetConfig = {
      ...set,
      id: uuidv4(),
      name: `${set.name} (copy)`,
      domains: { ...set.domains },
      tcp: { ...set.tcp },
      udp: { ...set.udp },
      fragmentation: { ...set.fragmentation },
      faking: { ...set.faking },
    };

    onChange("sets", [...sets, duplicated]);
  };

  const getDomainCount = (set: B4SetConfig): number => {
    return (
      (set.domains?.sni_domains?.length || 0) +
      (set.domains?.geosite_categories?.length || 0)
    );
  };

  return (
    <Stack spacing={3}>
      {/* Info Alert */}
      <Alert severity="info" icon={<LayersIcon />}>
        <Typography variant="subtitle2" gutterBottom>
          Configuration Sets allow you to define different bypass strategies for
          different domains or scenarios.
        </Typography>
        <Typography variant="caption" color="text.secondary">
          The Main Set is used as the default configuration when no other set
          matches. Each set can have its own TCP/UDP limits, fragmentation, and
          faking strategies.
        </Typography>
      </Alert>

      {/* Sets List */}
      <B4Section
        title="Configuration Sets"
        description="Manage multiple bypass configurations for different scenarios"
        icon={<LayersIcon />}
      >
        <Box sx={{ display: "flex", justifyContent: "flex-end" }}>
          <Button
            startIcon={<AddIcon />}
            onClick={handleAddSet}
            size="small"
            variant="contained"
            sx={{
              bgcolor: colors.secondary,
              color: colors.background.default,
              "&:hover": { bgcolor: colors.primary },
              "&:disabled": {
                bgcolor: colors.accent.secondary,
                color: colors.text.secondary,
              },
            }}
          >
            Create New Set
          </Button>
        </Box>

        <List sx={{ p: 0 }}>
          {sets.map((set) => {
            const isMain = set.id === mainSetId;
            const isExpanded = expandedSet === set.id;
            const domainCount = getDomainCount(set);

            return (
              <React.Fragment key={set.id}>
                <ListItem
                  sx={{
                    bgcolor: isMain
                      ? colors.accent.primary
                      : colors.background.dark,
                    borderRadius: radius.md,
                    mb: 1,
                    border: `1px solid ${
                      isMain ? colors.primary : colors.border.default
                    }`,
                    p: 2,
                  }}
                  secondaryAction={
                    <Stack direction="row" spacing={0.5}>
                      <IconButton
                        size="small"
                        onClick={() =>
                          setExpandedSet(isExpanded ? null : set.id)
                        }
                        title={isExpanded ? "Collapse" : "Expand details"}
                      >
                        {isExpanded ? <CollapseIcon /> : <ExpandIcon />}
                      </IconButton>

                      <IconButton
                        size="small"
                        onClick={() => handleDuplicateSet(set)}
                        title="Duplicate set"
                      >
                        <CopyIcon />
                      </IconButton>
                      <IconButton
                        size="small"
                        onClick={() => handleEditSet(set)}
                        title="Edit set"
                        sx={{ color: colors.secondary }}
                      >
                        <EditIcon />
                      </IconButton>
                      {!isMain && (
                        <IconButton
                          size="small"
                          onClick={() =>
                            setDeleteDialog({ open: true, setId: set.id })
                          }
                          title="Delete set"
                          sx={{ color: colors.quaternary }}
                        >
                          <DeleteIcon />
                        </IconButton>
                      )}
                    </Stack>
                  }
                >
                  <ListItemText
                    primary={
                      <Stack direction="row" alignItems="center" spacing={1}>
                        <Typography variant="h6">{set.name}</Typography>
                        {isMain && (
                          <B4Badge
                            label="MAIN"
                            badgeVariant="primary"
                            icon={<StarIcon />}
                          />
                        )}
                        {domainCount > 0 && (
                          <Chip
                            size="small"
                            label={`${domainCount} domains`}
                            sx={{
                              bgcolor: colors.accent.tertiary,
                              color: colors.tertiary,
                            }}
                          />
                        )}
                      </Stack>
                    }
                    secondary={
                      <Stack direction="row" spacing={1} sx={{ mt: 1 }}>
                        <Chip
                          label={`TCP: ${set.tcp.conn_bytes_limit}B`}
                          size="small"
                          variant="outlined"
                        />
                        <Chip
                          label={`UDP: ${set.udp.mode}`}
                          size="small"
                          variant="outlined"
                        />
                        <Chip
                          label={`Frag: ${set.fragmentation.strategy}`}
                          size="small"
                          variant="outlined"
                        />
                        <Chip
                          label={`Fake: ${set.faking.strategy}`}
                          size="small"
                          variant="outlined"
                        />
                      </Stack>
                    }
                  />
                </ListItem>

                {/* Expanded Details */}
                <Collapse in={isExpanded}>
                  <Box
                    sx={{
                      ml: 2,
                      mr: 2,
                      mb: 2,
                      p: 2,
                      bgcolor: colors.background.paper,
                      border: `1px solid ${colors.border.default}`,
                      borderRadius: radius.sm,
                    }}
                  >
                    <Grid container spacing={2}>
                      <Grid size={{ xs: 12, md: 4 }}>
                        <Typography variant="subtitle2" color="text.secondary">
                          TCP Configuration
                        </Typography>
                        <Stack spacing={0.5} sx={{ mt: 1 }}>
                          <Typography variant="body2">
                            Conn Bytes: {set.tcp.conn_bytes_limit}
                          </Typography>
                          <Typography variant="body2">
                            Seg2 Delay: {set.tcp.seg2delay}ms
                          </Typography>
                        </Stack>
                      </Grid>
                      <Grid size={{ xs: 12, md: 4 }}>
                        <Typography variant="subtitle2" color="text.secondary">
                          Fragmentation
                        </Typography>
                        <Stack spacing={0.5} sx={{ mt: 1 }}>
                          <Typography variant="body2">
                            Strategy: {set.fragmentation.strategy}
                          </Typography>
                          <Typography variant="body2">
                            SNI Position: {set.fragmentation.sni_position}
                          </Typography>
                          <Typography variant="body2">
                            Reverse:{" "}
                            {set.fragmentation.sni_reverse ? "Yes" : "No"}
                          </Typography>
                        </Stack>
                      </Grid>
                      <Grid size={{ xs: 12, md: 4 }}>
                        <Typography variant="subtitle2" color="text.secondary">
                          Faking
                        </Typography>
                        <Stack spacing={0.5} sx={{ mt: 1 }}>
                          <Typography variant="body2">
                            SNI: {set.faking.sni ? "Enabled" : "Disabled"}
                          </Typography>
                          <Typography variant="body2">
                            Strategy: {set.faking.strategy}
                          </Typography>
                          <Typography variant="body2">
                            TTL: {set.faking.ttl}
                          </Typography>
                        </Stack>
                      </Grid>
                      {set.domains.sni_domains.length > 0 && (
                        <Grid size={12}>
                          <Divider sx={{ my: 1 }} />
                          <Typography
                            variant="subtitle2"
                            color="text.secondary"
                          >
                            Domains
                          </Typography>
                          <Box sx={{ mt: 1 }}>
                            {set.domains.sni_domains
                              .slice(0, 5)
                              .map((domain) => (
                                <Chip
                                  key={domain}
                                  label={domain}
                                  size="small"
                                  sx={{ mr: 0.5, mb: 0.5 }}
                                />
                              ))}
                            {set.domains.sni_domains.length > 5 && (
                              <Chip
                                label={`+${
                                  set.domains.sni_domains.length - 5
                                } more`}
                                size="small"
                                variant="outlined"
                              />
                            )}
                          </Box>
                        </Grid>
                      )}
                    </Grid>
                  </Box>
                </Collapse>
              </React.Fragment>
            );
          })}
        </List>
      </B4Section>

      {/* Set Editor Dialog */}
      <SetEditor
        open={editDialog.open}
        settings={config.system}
        set={editDialog.set}
        isNew={editDialog.isNew}
        onClose={() => setEditDialog({ open: false, set: null, isNew: false })}
        onSave={handleSaveSet}
      />

      {/* Delete Confirmation Dialog */}
      <B4Dialog
        open={deleteDialog.open}
        title="Delete Configuration Set"
        subtitle="This action cannot be undone"
        icon={<WarningIcon />}
        onClose={() => setDeleteDialog({ open: false, setId: null })}
        actions={
          <>
            <Button
              onClick={() => setDeleteDialog({ open: false, setId: null })}
              sx={{ ...button_secondary }}
            >
              Cancel
            </Button>
            <Box sx={{ flex: 1 }} />
            <Button
              onClick={handleDeleteSet}
              variant="contained"
              sx={{
                bgcolor: colors.quaternary,
                "&:hover": { bgcolor: "#d32f2f" },
              }}
            >
              Delete Set
            </Button>
          </>
        }
      >
        <Alert severity="warning" sx={{ mb: 2 }}>
          Are you sure you want to delete this configuration set? All settings
          and domain assignments for this set will be permanently removed.
        </Alert>
        <Typography variant="body2" color="text.secondary">
          {deleteDialog.setId &&
            `Set: ${sets.find((s) => s.id === deleteDialog.setId)?.name}`}
        </Typography>
      </B4Dialog>
    </Stack>
  );
};
