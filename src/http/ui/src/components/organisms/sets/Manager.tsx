import { useMemo, useState } from "react";
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
  TextField,
  Snackbar,
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
  AirlineStops,
  CallSplit,
  Deblur,
  MultipleStop,
  Security as SecurityIcon,
  Category as CategoryIcon,
  Language as LanguageIcon,
  CompareArrows as CompareIcon,
  DragIndicator as DragIcon,
} from "@mui/icons-material";
import {
  DndContext,
  closestCenter,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
  DragStartEvent,
  DragOverlay,
} from "@dnd-kit/core";
import {
  SortableContext,
  verticalListSortingStrategy,
  useSortable,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { v4 as uuidv4 } from "uuid";

import B4Section from "@molecules/common/B4Section";
import { B4Dialog } from "@molecules/common/B4Dialog";
import { SetEditor } from "./Editor";

import { colors, radius, button_secondary } from "@design";
import { B4Config, B4SetConfig, MAIN_SET_ID } from "@models/Config";
import { SetCompare } from "./Compare";
import { useSets } from "@hooks/useSets";
import React from "react";

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
  onRefresh: () => void;
}

export const SetsManager = ({ config, onRefresh }: SetsManagerProps) => {
  const {
    createSet,
    updateSet,
    deleteSet,
    duplicateSet,
    reorderSets,
    loading: saving,
  } = useSets();

  const [filterText, setFilterText] = useState("");
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
  const [snackbar, setSnackbar] = useState<{
    open: boolean;
    message: string;
    severity: "success" | "error";
  }>({
    open: false,
    message: "",
    severity: "success",
  });

  const setsData = config.sets || [];
  const sets = setsData.map((s) => ("set" in s ? s.set : s)) as B4SetConfig[];
  const setsStats = setsData.map((s) =>
    "stats" in s ? s.stats : null
  ) as SetStats[];

  const [compareDialog, setCompareDialog] = useState<{
    open: boolean;
    setA: B4SetConfig | null;
    setB: B4SetConfig | null;
  }>({ open: false, setA: null, setB: null });

  const [activeId, setActiveId] = useState<string | null>(null);

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: { distance: 8 },
    })
  );

  const handleDragStart = (event: DragStartEvent) => {
    setActiveId(event.active.id as string);
  };

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    setActiveId(null);

    if (!over || active.id === over.id) return;

    const oldIndex = sets.findIndex((s) => s.id === active.id);
    const newIndex = sets.findIndex((s) => s.id === over.id);

    if (oldIndex === -1 || newIndex === -1) return;

    const newOrder = [...sets];
    const [removed] = newOrder.splice(oldIndex, 1);
    newOrder.splice(newIndex, 0, removed);

    void (async () => {
      const result = await reorderSets(newOrder.map((s) => s.id));
      if (result.success) onRefresh();
    })();
  };

  const activeSet = activeId ? sets.find((s) => s.id === activeId) : null;

  const handleAddSet = () => {
    const newSet: B4SetConfig = {
      id: uuidv4(),
      name: `Set ${sets.length + 1}`,
      enabled: true,
      tcp: {
        conn_bytes_limit: 19,
        seg2delay: 0,
        syn_fake: false,
        syn_fake_len: 0,
        drop_sack: false,
        win_mode: "off",
        win_values: [0, 1460, 8192, 65535],
        desync_mode: "off",
        desync_ttl: 3,
        desync_count: 3,
      } as B4SetConfig["tcp"],
      udp: {
        mode: "fake",
        fake_seq_length: 6,
        fake_len: 64,
        faking_strategy: "none",
        dport_filter: "",
        filter_quic: "disabled",
        filter_stun: true,
        conn_bytes_limit: 8,
        seg2delay: 0,
      } as B4SetConfig["udp"],
      fragmentation: {
        strategy: "tcp",
        reverse_order: true,
        middle_sni: true,
        sni_position: 1,
        oob_position: 0,
        oob_char: 120,
        tlsrec_pos: 0,
        combo: {
          extension_split: true,
          first_byte_split: true,
          shuffle_mode: "middle",
          first_delay_ms: 100,
          jitter_max_us: 2000,
        },
        disorder: {
          shuffle_mode: "full",
          min_jitter_us: 1000,
          max_jitter_us: 3000,
        },
      } as B4SetConfig["fragmentation"],
      faking: {
        sni: true,
        ttl: 8,
        strategy: "pastseq",
        seq_offset: 10000,
        sni_seq_length: 1,
        sni_type: 2,
        custom_payload: "",
        sni_mutation: {
          mode: "off",
          grease_count: 3,
          padding_size: 2048,
          fake_ext_count: 5,
          fake_snis: ["ya.ru", "vk.com", "max.ru"],
        },
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
    void (async () => {
      const result = editDialog.isNew
        ? await createSet(set)
        : await updateSet(set);

      if (result.success) {
        setSnackbar({
          open: true,
          message: editDialog.isNew ? "Set created" : "Set updated",
          severity: "success",
        });
        setEditDialog({ open: false, set: null, isNew: false });
        onRefresh();
      } else {
        setSnackbar({
          open: true,
          message: result.error || "Failed",
          severity: "error",
        });
      }
    })();
  };

  const handleDeleteSet = () => {
    if (!deleteDialog.setId) return;
    void (async () => {
      const result = await deleteSet(deleteDialog.setId!);
      if (result.success) {
        setSnackbar({
          open: true,
          message: "Set deleted",
          severity: "success",
        });
        setDeleteDialog({ open: false, setId: null });
        onRefresh();
      } else {
        setSnackbar({
          open: true,
          message: result.error || "Failed to delete",
          severity: "error",
        });
      }
    })();
  };

  const handleDuplicateSet = (set: B4SetConfig) => {
    void (async () => {
      const result = await duplicateSet(set);
      if (result.success) {
        setSnackbar({
          open: true,
          message: "Set duplicated",
          severity: "success",
        });
        onRefresh();
      } else {
        setSnackbar({
          open: true,
          message: result.error || "Failed",
          severity: "error",
        });
      }
    })();
  };

  const handleToggleEnabled = (set: B4SetConfig, enabled: boolean) => {
    void (async () => {
      const updatedSet = { ...set, enabled };
      const result = await updateSet(updatedSet);
      if (result.success) {
        onRefresh();
      } else {
        setSnackbar({
          open: true,
          message: result.error || "Failed to update",
          severity: "error",
        });
      }
    })();
  };

  const filteredSets = useMemo(() => {
    if (!filterText.trim()) return sets;
    const lower = filterText.toLowerCase();
    return sets.filter((set) => {
      if (set.name.toLowerCase().includes(lower)) return true;
      if (
        set.targets?.sni_domains?.some((d) => d.toLowerCase().includes(lower))
      )
        return true;
      if (
        set.targets?.geosite_categories?.some((c) =>
          c.toLowerCase().includes(lower)
        )
      )
        return true;
      if (
        set.targets?.geoip_categories?.some((c) =>
          c.toLowerCase().includes(lower)
        )
      )
        return true;
      return false;
    });
  }, [sets, filterText]);

  interface SortableSetCardProps {
    set: B4SetConfig;
    children: React.ReactNode;
  }

  const SortableSetCard: React.FC<SortableSetCardProps> = ({
    set,
    children,
  }) => {
    const {
      attributes,
      listeners,
      setNodeRef,
      transform,
      transition,
      isDragging,
    } = useSortable({ id: set.id });

    const style = {
      transform: CSS.Transform.toString(transform),
      transition,
      opacity: isDragging ? 0.4 : 1,
      zIndex: isDragging ? 1 : 0,
    };

    return (
      <Box ref={setNodeRef} style={style} {...attributes} {...listeners}>
        {children}
      </Box>
    );
  };

  return (
    <Stack spacing={3}>
      {/* Info Alert */}
      <Alert
        severity="info"
        sx={{
          bgcolor: colors.accent.primary,
          border: `1px solid ${colors.secondary}44`,
        }}
        icon={<LayersIcon />}
      >
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
        <Stack direction="row" justifyContent="space-between" mb={0}>
          <TextField
            size="small"
            placeholder="Filter by name, domain, or geosite category..."
            value={filterText}
            onChange={(e) => setFilterText(e.target.value)}
            sx={{
              "& .MuiOutlinedInput-root": {
                bgcolor: colors.background.default,
              },
            }}
          />
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
        </Stack>

        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragStart={handleDragStart}
          onDragEnd={handleDragEnd}
        >
          <SortableContext
            items={filteredSets.map((s) => s.id)}
            strategy={verticalListSortingStrategy}
          >
            <List sx={{ p: 0 }}>
              <Stack spacing={2}>
                {filteredSets.map((set) => {
                  const index = sets.findIndex((s) => s.id === set.id);
                  const isMain = set.id === MAIN_SET_ID;
                  const isExpanded = expandedSet === set.id;
                  const domainCount = getDomainCount(set, index);
                  const ipCount = getIpCount(set, index);
                  const hasTargets = domainCount > 0 || ipCount > 0;

                  return (
                    <SortableSetCard key={set.id} set={set}>
                      <Paper
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
                          cursor: "grab",
                          transition: "all 0.2s ease",
                          "&:hover": {
                            borderColor: isMain
                              ? colors.secondary
                              : colors.primary,
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
                              : `${colors.primary}${(100 - index * 20).toString(
                                  16
                                )}`,
                          }}
                        />

                        <Box sx={{ p: 2, pl: 3 }}>
                          <Stack
                            direction="row"
                            alignItems="center"
                            justifyContent="space-between"
                          >
                            <Stack
                              direction="row"
                              alignItems="center"
                              spacing={2}
                            >
                              <DragIcon
                                sx={{
                                  color: colors.text.secondary,
                                  cursor: "grab",
                                  "&:hover": { color: colors.primary },
                                }}
                              />
                              <Tooltip
                                title={
                                  set.enabled ? "Disable set" : "Enable set"
                                }
                              >
                                <Switch
                                  checked={set.enabled}
                                  onChange={(e) =>
                                    handleToggleEnabled(set, e.target.checked)
                                  }
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
                                          {setsStats[index]?.manual_domains >
                                            0 &&
                                            ` (${setsStats[index].manual_domains} manual)`}
                                          {setsStats[index]?.geosite_domains >
                                            0 &&
                                            ` (${setsStats[index].geosite_domains} from geosite)`}
                                        </Typography>
                                        <br />
                                        <Typography variant="caption">
                                          IPs:{" "}
                                          {setsStats[index]?.total_ips ||
                                            ipCount}
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
                                        setsStats[index]?.total_domains ||
                                        domainCount
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
                                  {isExpanded ? (
                                    <CollapseIcon />
                                  ) : (
                                    <ExpandIcon />
                                  )}
                                </IconButton>
                              </Tooltip>
                              <Divider orientation="vertical" flexItem />
                              <Tooltip title="Duplicate set">
                                <IconButton
                                  size="small"
                                  onClick={() => handleDuplicateSet(set)}
                                  sx={{ "&:hover": { color: colors.tertiary } }}
                                >
                                  <CopyIcon />
                                </IconButton>
                              </Tooltip>

                              <Tooltip title="Compare with another set">
                                <IconButton
                                  size="small"
                                  onClick={() =>
                                    setCompareDialog({
                                      open: true,
                                      setA: set,
                                      setB: null,
                                    })
                                  }
                                  disabled={sets.length < 2}
                                  sx={{ "&:hover": { color: colors.primary } }}
                                >
                                  <CompareIcon />
                                </IconButton>
                              </Tooltip>

                              <Tooltip title="Edit set">
                                <IconButton
                                  size="small"
                                  onClick={() => handleEditSet(set)}
                                  sx={{
                                    "&:hover": { color: colors.secondary },
                                  }}
                                >
                                  <EditIcon />
                                </IconButton>
                              </Tooltip>

                              {!isMain && (
                                <Tooltip title="Delete set">
                                  <IconButton
                                    size="small"
                                    onClick={() =>
                                      setDeleteDialog({
                                        open: true,
                                        setId: set.id,
                                      })
                                    }
                                    sx={{
                                      "&:hover": { color: colors.quaternary },
                                    }}
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
                                <Stack
                                  direction="row"
                                  flexWrap="wrap"
                                  gap={0.5}
                                >
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
                                <Typography
                                  variant="caption"
                                  color="text.secondary"
                                >
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
                                <Typography
                                  variant="caption"
                                  color="text.secondary"
                                >
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
                    </SortableSetCard>
                  );
                })}

                {filteredSets.length === 0 && filterText && (
                  <Paper
                    elevation={0}
                    sx={{
                      p: 3,
                      textAlign: "center",
                      border: `1px dashed ${colors.border.default}`,
                      borderRadius: radius.md,
                    }}
                  >
                    <Typography color="text.secondary">
                      No sets match "{filterText}"
                    </Typography>
                  </Paper>
                )}
              </Stack>
            </List>
          </SortableContext>
          <DragOverlay>
            {activeSet ? (
              <Paper
                elevation={8}
                sx={{
                  p: 2,
                  bgcolor: colors.background.paper,
                  border: `2px solid ${colors.secondary}`,
                  borderRadius: radius.md,
                  opacity: 0.9,
                  cursor: "grabbing",
                }}
              >
                <Stack direction="row" alignItems="center" spacing={2}>
                  <DragIcon sx={{ color: colors.secondary }} />
                  <Typography variant="h6" fontWeight={500}>
                    {activeSet.name}
                  </Typography>
                  <Chip
                    size="small"
                    label={activeSet.id === MAIN_SET_ID ? "MAIN" : `Set`}
                    sx={{
                      bgcolor: colors.accent.tertiary,
                      color: colors.text.primary,
                    }}
                  />
                </Stack>
              </Paper>
            ) : null}
          </DragOverlay>
        </DndContext>
      </B4Section>

      <SetEditor
        open={editDialog.open}
        settings={config.system}
        set={editDialog.set!}
        isNew={editDialog.isNew}
        saving={saving}
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

      {/* Compare Dialog - Set Selection */}
      <B4Dialog
        open={compareDialog.open && !compareDialog.setB}
        onClose={() =>
          setCompareDialog({ open: false, setA: null, setB: null })
        }
        title="Select Set to Compare"
        subtitle={`Comparing with: ${compareDialog.setA?.name}`}
        icon={<CompareIcon />}
      >
        <List>
          {sets
            .filter((s) => s.id !== compareDialog.setA?.id)
            .map((s) => (
              <ListItem
                key={s.id}
                component="div"
                onClick={() =>
                  setCompareDialog((prev) => ({ ...prev, setB: s }))
                }
                sx={{
                  cursor: "pointer",
                  borderRadius: 1,
                  "&:hover": { bgcolor: colors.accent.primary },
                }}
              >
                <ListItemText primary={s.name} />
              </ListItem>
            ))}
        </List>
      </B4Dialog>

      <SetCompare
        open={compareDialog.open && !!compareDialog.setB}
        setA={compareDialog.setA}
        setB={compareDialog.setB}
        onClose={() =>
          setCompareDialog({ open: false, setA: null, setB: null })
        }
      />

      {/* Snackbar for notifications */}
      <Snackbar
        open={snackbar.open}
        autoHideDuration={4000}
        onClose={() => setSnackbar({ ...snackbar, open: false })}
        anchorOrigin={{ vertical: "bottom", horizontal: "right" }}
      >
        <Alert
          onClose={() => setSnackbar({ ...snackbar, open: false })}
          severity={snackbar.severity}
          sx={{ width: "100%" }}
        >
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Stack>
  );
};
