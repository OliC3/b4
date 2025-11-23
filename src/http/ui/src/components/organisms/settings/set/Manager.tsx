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
  Paper,
  Tooltip,
  Switch,
} from "@mui/material";
import {
  Add as AddIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  ContentCopy as CopyIcon,
  ExpandMore as ExpandIcon,
  ExpandLess as CollapseIcon,
  Layers as LayersIcon,
  Warning as WarningIcon,
  ArrowUpward as ArrowUpIcon,
  ArrowDownward as ArrowDownIcon,
  AirlineStops,
  CallSplit,
  Deblur,
  MultipleStop,
  Security as SecurityIcon,
  Category as CategoryIcon,
  Language as LanguageIcon,
} from "@mui/icons-material";
import { v4 as uuidv4 } from "uuid";

import B4Section from "@molecules/common/B4Section";
import { B4Dialog } from "@molecules/common/B4Dialog";
import { SetEditor } from "./Editor";

import { colors, radius, button_secondary } from "@design";
import { B4Config, B4SetConfig, MAIN_SET_ID } from "@models/Config";

export interface SetStats {
  manual_domains: number;
  manual_ips: number;
  geosite_domains: number;
  geoip_ips: number;
  total_domains: number;
  total_ips: number;
  geosite_category_breakdown?: Record<string, number>;
  geoip_category_breakdown?: Record<string, number>;
}

export interface SetWithStats extends B4SetConfig {
  stats: SetStats;
}
interface SetsManagerProps {
  config: B4Config & { sets?: SetWithStats[] };
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
  const setsData = config.sets || [];
  const sets = setsData.map((s) => ("set" in s ? s.set : s)) as B4SetConfig[];
  const setsStats = setsData.map((s) =>
    "stats" in s ? s.stats : null
  ) as SetStats[];

  const handleAddSet = () => {
    const newSet: B4SetConfig = {
      id: uuidv4(),
      name: `Set ${sets.length + 1}`,
      enabled: true,
      tcp: { conn_bytes_limit: 19, seg2delay: 0 } as B4SetConfig["tcp"],
      udp: {
        mode: "fake",
        fake_seq_length: 6,
        fake_len: 64,
        faking_strategy: "none",
        dport_filter: "",
        filter_quic: "disabled",
        filter_stun: true,
        conn_bytes_limit: 8,
      } as B4SetConfig["udp"],
      fragmentation: {
        strategy: "tcp",
        reverse_order: true,
        middle_sni: true,
        sni_position: 1,
        oob_position: 0,
        oob_char: 120,
        tlsrec_pos: 1,
      } as B4SetConfig["fragmentation"],
      faking: {
        sni: true,
        ttl: 8,
        strategy: "pastseq",
        seq_offset: 10000,
        sni_seq_length: 1,
        sni_type: 2,
        custom_payload: "",
      } as B4SetConfig["faking"],
      targets: {
        sni_domains: [],
        ip: [],
        geosite_categories: [],
        geoip_categories: [],
      } as B4SetConfig["targets"],
    };

    setEditDialog({ open: true, set: newSet, isNew: true });
  };

  const getDomainCount = (set: B4SetConfig, index: number): number => {
    if (setsStats[index]) {
      return setsStats[index].total_domains;
    }
    return (
      (set.targets?.sni_domains?.length || 0) +
      (set.targets?.geosite_categories?.length || 0)
    );
  };

  const getIpCount = (set: B4SetConfig, index: number): number => {
    if (setsStats[index]) {
      return setsStats[index].total_ips;
    }
    return set.targets?.ip?.length || 0;
  };

  const handleEditSet = (set: B4SetConfig) => {
    setEditDialog({ open: true, set, isNew: false });
  };

  const handleSaveSet = (set: B4SetConfig) => {
    const existingIndex = sets.findIndex((s) => s.id === set.id);

    let updatedSets: B4SetConfig[];
    if (existingIndex >= 0) {
      // Update existing
      updatedSets = [
        ...sets.slice(0, existingIndex),
        set,
        ...sets.slice(existingIndex + 1),
      ];
    } else {
      // Add new
      updatedSets = [...sets, set];
    }

    onChange("sets", updatedSets);
    setEditDialog({ open: false, set: null, isNew: false });
  };

  const handleDeleteSet = () => {
    if (!deleteDialog.setId) return;

    const filteredSets = sets.filter((s) => s.id !== deleteDialog.setId);
    onChange("sets", filteredSets);

    setDeleteDialog({ open: false, setId: null });
  };

  const handleDuplicateSet = (set: B4SetConfig) => {
    const duplicated: B4SetConfig = {
      ...set,
      id: uuidv4(),
      name: `${set.name} (copy)`,
      targets: { ...set.targets },
      tcp: { ...set.tcp },
      udp: { ...set.udp },
      fragmentation: { ...set.fragmentation },
      faking: { ...set.faking },
    };

    onChange("sets", [...sets, duplicated]);
  };

  const handleMoveSetUp = (index: number) => {
    if (index <= 0) return;
    const newSets = [...sets];
    [newSets[index - 1], newSets[index]] = [newSets[index], newSets[index - 1]];
    onChange("sets", newSets);
  };

  const handleMoveSetDown = (index: number) => {
    if (index >= sets.length - 1) return;
    const newSets = [...sets];
    [newSets[index], newSets[index + 1]] = [newSets[index + 1], newSets[index]];
    onChange("sets", newSets);
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
          <Stack spacing={2}>
            {sets.map((set, index) => {
              const isMain = set.id === MAIN_SET_ID;
              const isExpanded = expandedSet === set.id;
              const domainCount = getDomainCount(set, index);
              const ipCount = getIpCount(set, index);
              const hasTargets = domainCount > 0 || ipCount > 0;

              return (
                <Paper
                  key={set.id}
                  elevation={isMain ? 2 : 1}
                  sx={{
                    opacity: set.enabled ? 1 : 0.6,
                    position: "relative",
                    overflow: "hidden",
                    border: `1px solid ${
                      isMain ? colors.primary : colors.border.default
                    }`,
                    borderRadius: radius.md,
                    bgcolor: isMain
                      ? `${colors.accent.primary}44`
                      : colors.background.paper,
                    transition: "all 0.3s ease",
                    "&:hover": {
                      borderColor: isMain ? colors.secondary : colors.primary,
                      transform: "translateY(-2px)",
                      boxShadow: `0 4px 12px ${colors.accent.primary}`,
                    },
                  }}
                >
                  <Box
                    sx={{
                      position: "absolute",
                      left: 0,
                      top: 0,
                      bottom: 0,
                      width: 4,
                      bgcolor: isMain
                        ? colors.secondary
                        : `${colors.primary}${(100 - index * 20).toString(16)}`,
                    }}
                  />

                  <Box sx={{ p: 2, pl: 3 }}>
                    <Stack
                      direction="row"
                      alignItems="center"
                      justifyContent="space-between"
                    >
                      <Stack direction="row" alignItems="center" spacing={2}>
                        <Tooltip
                          title={set.enabled ? "Disable set" : "Enable set"}
                        >
                          <Switch
                            checked={set.enabled}
                            onChange={(e) => {
                              const updatedSet = {
                                ...set,
                                enabled: e.target.checked,
                              };
                              const updatedSets = sets.map((s) =>
                                s.id === set.id ? updatedSet : s
                              );
                              onChange("sets", updatedSets);
                            }}
                            size="small"
                            sx={{
                              "& .MuiSwitch-switchBase.Mui-checked": {
                                color: colors.secondary,
                              },
                              "& .MuiSwitch-switchBase.Mui-checked + .MuiSwitch-track":
                                {
                                  backgroundColor: colors.secondary,
                                },
                            }}
                          />
                        </Tooltip>
                        <Chip
                          size="small"
                          label={isMain ? "MAIN" : `#${index + 1}`}
                          sx={{
                            minWidth: 48,
                            fontWeight: 600,
                            bgcolor: isMain
                              ? colors.secondary
                              : colors.accent.tertiary,
                            color: isMain
                              ? colors.background.default
                              : colors.text.primary,
                          }}
                        />

                        <Typography
                          variant="h6"
                          sx={{
                            fontWeight: isMain ? 600 : 500,
                            color: colors.text.primary,
                          }}
                        >
                          {set.name}
                        </Typography>

                        <Stack direction="row" spacing={1}>
                          {hasTargets && (
                            <Tooltip
                              title={
                                <Box>
                                  <Typography variant="caption">
                                    Domains:{" "}
                                    {setsStats[index]?.total_domains ||
                                      domainCount}
                                    {setsStats[index]?.manual_domains > 0 &&
                                      ` (${setsStats[index].manual_domains} manual)`}
                                    {setsStats[index]?.geosite_domains > 0 &&
                                      ` (${setsStats[index].geosite_domains} from geosite)`}
                                  </Typography>
                                  <br />
                                  <Typography variant="caption">
                                    IPs:{" "}
                                    {setsStats[index]?.total_ips || ipCount}
                                    {setsStats[index]?.manual_ips > 0 &&
                                      ` (${setsStats[index].manual_ips} manual)`}
                                    {setsStats[index]?.geoip_ips > 0 &&
                                      ` (${setsStats[index].geoip_ips} from geoip)`}
                                  </Typography>
                                </Box>
                              }
                            >
                              <Chip
                                icon={<LanguageIcon />}
                                label={`${
                                  setsStats[index]?.total_domains || domainCount
                                }/${
                                  setsStats[index]?.total_ips ||
                                  set.targets.ip.length
                                }`}
                                size="small"
                                variant="outlined"
                                sx={{
                                  borderColor: colors.secondary,
                                  color: colors.secondary,
                                }}
                              />
                            </Tooltip>
                          )}

                          {set.faking.sni && (
                            <Tooltip title="SNI Faking Enabled">
                              <Chip
                                icon={<SecurityIcon />}
                                label="SNI"
                                size="small"
                                sx={{
                                  bgcolor: `${colors.secondary}22`,
                                  color: colors.secondary,
                                }}
                              />
                            </Tooltip>
                          )}
                        </Stack>
                      </Stack>

                      <Stack direction="row" spacing={0.5}>
                        <Tooltip title="Move up">
                          <IconButton
                            size="small"
                            onClick={() => handleMoveSetUp(index)}
                            disabled={index === 0}
                            sx={{
                              opacity: index === 0 ? 0.3 : 1,
                              "&:hover": { color: colors.secondary },
                            }}
                          >
                            <ArrowUpIcon />
                          </IconButton>
                        </Tooltip>

                        <Tooltip title="Move down">
                          <IconButton
                            size="small"
                            onClick={() => handleMoveSetDown(index)}
                            disabled={index === sets.length - 1}
                            sx={{
                              opacity: index === sets.length - 1 ? 0.3 : 1,
                              "&:hover": { color: colors.secondary },
                            }}
                          >
                            <ArrowDownIcon />
                          </IconButton>
                        </Tooltip>

                        <Divider
                          orientation="vertical"
                          flexItem
                          sx={{ mx: 0.5 }}
                        />

                        <Tooltip
                          title={isExpanded ? "Collapse" : "View details"}
                        >
                          <IconButton
                            size="small"
                            onClick={() =>
                              setExpandedSet(isExpanded ? null : set.id)
                            }
                            sx={{ "&:hover": { color: colors.primary } }}
                          >
                            {isExpanded ? <CollapseIcon /> : <ExpandIcon />}
                          </IconButton>
                        </Tooltip>

                        <Tooltip title="Duplicate set">
                          <IconButton
                            size="small"
                            onClick={() => handleDuplicateSet(set)}
                            sx={{ "&:hover": { color: colors.tertiary } }}
                          >
                            <CopyIcon />
                          </IconButton>
                        </Tooltip>

                        <Tooltip title="Edit set">
                          <IconButton
                            size="small"
                            onClick={() => handleEditSet(set)}
                            sx={{ "&:hover": { color: colors.secondary } }}
                          >
                            <EditIcon />
                          </IconButton>
                        </Tooltip>

                        {!isMain && (
                          <Tooltip title="Delete set">
                            <IconButton
                              size="small"
                              onClick={() =>
                                setDeleteDialog({ open: true, setId: set.id })
                              }
                              sx={{ "&:hover": { color: colors.quaternary } }}
                            >
                              <DeleteIcon />
                            </IconButton>
                          </Tooltip>
                        )}
                      </Stack>
                    </Stack>

                    <Grid container spacing={3} sx={{ mt: 2 }}>
                      <Grid
                        size={{ xs: 12, sm: 6, md: 3 }}
                        sx={{ display: "flex" }}
                      >
                        <Box
                          sx={{
                            width: "100%",
                            p: 1,
                            borderRadius: radius.sm,
                            bgcolor: colors.background.dark,
                            border: `1px solid ${colors.border.light}`,
                          }}
                        >
                          <Stack spacing={0.5}>
                            <Stack
                              direction="row"
                              alignItems="center"
                              spacing={0.5}
                            >
                              <AirlineStops
                                sx={{
                                  fontSize: 16,
                                  color: colors.text.secondary,
                                }}
                              />
                              <Typography
                                variant="caption"
                                color="text.secondary"
                              >
                                TCP
                              </Typography>
                            </Stack>
                            <Typography variant="body2" fontWeight={500}>
                              {set.tcp.conn_bytes_limit}B limit
                            </Typography>
                            {set.tcp.seg2delay > 0 && (
                              <Typography
                                variant="caption"
                                color="text.secondary"
                              >
                                {set.tcp.seg2delay}ms delay
                              </Typography>
                            )}
                          </Stack>
                        </Box>
                      </Grid>

                      <Grid
                        size={{ xs: 12, sm: 6, md: 3 }}
                        sx={{ display: "flex" }}
                      >
                        <Box
                          sx={{
                            width: "100%",
                            p: 1,
                            borderRadius: radius.sm,
                            bgcolor: colors.background.dark,
                            border: `1px solid ${colors.border.light}`,
                          }}
                        >
                          <Stack spacing={0.5}>
                            <Stack
                              direction="row"
                              alignItems="center"
                              spacing={0.5}
                            >
                              <MultipleStop
                                sx={{
                                  fontSize: 16,
                                  color: colors.text.secondary,
                                }}
                              />
                              <Typography
                                variant="caption"
                                color="text.secondary"
                              >
                                UDP
                              </Typography>
                            </Stack>
                            <Typography variant="body2" fontWeight={500}>
                              Mode: {set.udp.mode}
                            </Typography>
                            <Typography
                              variant="caption"
                              color="text.secondary"
                            >
                              QUIC: {set.udp.filter_quic}
                            </Typography>
                          </Stack>
                        </Box>
                      </Grid>

                      <Grid
                        size={{ xs: 12, sm: 6, md: 3 }}
                        sx={{ display: "flex" }}
                      >
                        <Box
                          sx={{
                            width: "100%",
                            p: 1,
                            borderRadius: radius.sm,
                            bgcolor: colors.background.dark,
                            border: `1px solid ${colors.border.light}`,
                          }}
                        >
                          <Stack spacing={0.5}>
                            <Stack
                              direction="row"
                              alignItems="center"
                              spacing={0.5}
                            >
                              <CallSplit
                                sx={{
                                  fontSize: 16,
                                  color: colors.text.secondary,
                                }}
                              />
                              <Typography
                                variant="caption"
                                color="text.secondary"
                              >
                                Fragment
                              </Typography>
                            </Stack>
                            <Typography variant="body2" fontWeight={500}>
                              {set.fragmentation.strategy.toUpperCase()}
                            </Typography>
                            <Stack direction="row" spacing={0.5}>
                              {set.fragmentation.reverse_order && (
                                <Chip
                                  label="REV"
                                  size="small"
                                  sx={{ height: 16, fontSize: "0.65rem" }}
                                />
                              )}
                              {set.fragmentation.middle_sni && (
                                <Chip
                                  label="MID"
                                  size="small"
                                  sx={{ height: 16, fontSize: "0.65rem" }}
                                />
                              )}
                            </Stack>
                          </Stack>
                        </Box>
                      </Grid>
                      <Grid
                        size={{ xs: 12, sm: 6, md: 3 }}
                        sx={{ display: "flex" }}
                      >
                        <Box
                          sx={{
                            width: "100%",
                            p: 1,
                            borderRadius: radius.sm,
                            bgcolor: colors.background.dark,
                            border: `1px solid ${colors.border.light}`,
                          }}
                        >
                          <Stack spacing={0.5}>
                            <Stack
                              direction="row"
                              alignItems="center"
                              spacing={0.5}
                            >
                              <Deblur
                                sx={{
                                  fontSize: 16,
                                  color: colors.text.secondary,
                                }}
                              />
                              <Typography
                                variant="caption"
                                color="text.secondary"
                              >
                                Faking
                              </Typography>
                            </Stack>
                            <Typography variant="body2" fontWeight={500}>
                              {set.faking.strategy}
                            </Typography>
                            <Typography
                              variant="caption"
                              color="text.secondary"
                            >
                              TTL: {set.faking.ttl}
                            </Typography>
                          </Stack>
                        </Box>
                      </Grid>
                    </Grid>

                    <Collapse in={isExpanded}>
                      <Divider sx={{ my: 2 }} />

                      {(set.targets.sni_domains.length > 0 ||
                        set.targets.geosite_categories.length > 0) && (
                        <Box sx={{ mb: 2 }}>
                          <Typography
                            variant="subtitle2"
                            sx={{ mb: 1, color: colors.text.secondary }}
                          >
                            Target Domains & Categories
                          </Typography>
                          <Stack direction="row" flexWrap="wrap" gap={0.5}>
                            {set.targets.geosite_categories.map((cat) => (
                              <Chip
                                key={cat}
                                label={cat}
                                size="small"
                                icon={<CategoryIcon />}
                                sx={{
                                  bgcolor: `${colors.tertiary}22`,
                                  color: colors.tertiary,
                                }}
                              />
                            ))}
                            {set.targets.sni_domains
                              .slice(0, 5)
                              .map((domain) => (
                                <Chip
                                  key={domain}
                                  label={domain}
                                  size="small"
                                  sx={{
                                    bgcolor: `${colors.secondary}22`,
                                    color: colors.secondary,
                                  }}
                                />
                              ))}
                            {set.targets.sni_domains.length > 5 && (
                              <Chip
                                label={`+${
                                  set.targets.sni_domains.length - 5
                                } more`}
                                size="small"
                                variant="outlined"
                              />
                            )}
                          </Stack>
                        </Box>
                      )}

                      {/* Advanced Settings */}
                      <Grid container spacing={2}>
                        <Grid size={{ xs: 12, md: 6 }}>
                          <Typography variant="caption" color="text.secondary">
                            Advanced TCP/UDP Settings
                          </Typography>
                          <List dense disablePadding>
                            <ListItem>
                              <ListItemText
                                primary="UDP Fake Length"
                                secondary={`${set.udp.fake_len} bytes`}
                              />
                            </ListItem>
                            <ListItem>
                              <ListItemText
                                primary="UDP Fake Strategy"
                                secondary={set.udp.faking_strategy}
                              />
                            </ListItem>
                          </List>
                        </Grid>
                        <Grid size={{ xs: 12, md: 6 }}>
                          <Typography variant="caption" color="text.secondary">
                            Faking Details
                          </Typography>
                          <List dense disablePadding>
                            <ListItem>
                              <ListItemText
                                primary="SNI Type"
                                secondary={
                                  ["Random", "Custom", "Default"][
                                    set.faking.sni_type
                                  ]
                                }
                              />
                            </ListItem>
                            <ListItem>
                              <ListItemText
                                primary="Sequence Offset"
                                secondary={set.faking.seq_offset}
                              />
                            </ListItem>
                          </List>
                        </Grid>
                      </Grid>
                    </Collapse>
                  </Box>
                </Paper>
              );
            })}
          </Stack>
        </List>
      </B4Section>

      <SetEditor
        open={editDialog.open}
        settings={config.system}
        set={editDialog.set!}
        isNew={editDialog.isNew}
        stats={
          setsStats[sets.findIndex((s) => s.id === editDialog.set?.id)] ||
          undefined
        }
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
