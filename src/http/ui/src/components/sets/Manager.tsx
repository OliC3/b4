import { useMemo, useState } from "react";
import {
  Box,
  Grid,
  Stack,
  Button,
  Typography,
  List,
  ListItem,
  ListItemText,
  Paper,
  TextField,
  InputAdornment,
} from "@mui/material";

import {
  AddIcon,
  SetsIcon,
  DomainIcon,
  WarningIcon,
  CompareIcon,
  CheckIcon,
} from "@b4.icons";
import SearchOutlinedIcon from "@mui/icons-material/SearchOutlined";

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
  rectSortingStrategy,
  useSortable,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { v4 as uuidv4 } from "uuid";

import { B4Section, B4Dialog } from "@b4.elements";
import { useSnackbar } from "@context/SnackbarProvider";

import { SetEditor } from "./Editor";
import { SetCard } from "./SetCard";
import { SetCompare } from "./Compare";

import { colors, radius } from "@design";
import { B4Config, B4SetConfig } from "@models/config";
import { useSets } from "@hooks/useSets";

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

interface SortableCardWrapperProps {
  id: string;
  children:
    | React.ReactNode
    | ((props: React.HTMLAttributes<HTMLDivElement>) => JSX.Element);
}

const SortableCardWrapper = ({ id, children }: SortableCardWrapperProps) => {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id });

  return (
    <Box
      ref={setNodeRef}
      style={{
        transform: CSS.Transform.toString(transform),
        transition,
        opacity: isDragging ? 0.4 : 1,
        zIndex: isDragging ? 1 : 0,
      }}
    >
      {/* Pass drag handle props to child */}
      {typeof children === "function"
        ? children({ ...attributes, ...listeners })
        : children}
    </Box>
  );
};

export const SetsManager = ({ config, onRefresh }: SetsManagerProps) => {
  const { showSuccess, showError } = useSnackbar();
  const {
    createSet,
    updateSet,
    deleteSet,
    duplicateSet,
    reorderSets,
    loading: saving,
  } = useSets();

  const [filterText, setFilterText] = useState("");
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
  const [compareDialog, setCompareDialog] = useState<{
    open: boolean;
    setA: B4SetConfig | null;
    setB: B4SetConfig | null;
  }>({ open: false, setA: null, setB: null });

  const [activeId, setActiveId] = useState<string | null>(null);

  const setsData = config.sets || [];
  const sets = setsData.map((s) => ("set" in s ? s.set : s)) as B4SetConfig[];
  const setsStats = setsData.map((s) =>
    "stats" in s ? s.stats : null
  ) as (SetStats | null)[];

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: { distance: 8 },
    })
  );

  // Summary stats
  const summaryStats = useMemo(() => {
    const enabledCount = sets.filter((s) => s.enabled).length;
    const totalDomains = setsStats.reduce(
      (acc, s) => acc + (s?.total_domains || 0),
      0
    );
    const totalIps = setsStats.reduce((acc, s) => acc + (s?.total_ips || 0), 0);
    return {
      total: sets.length,
      enabled: enabledCount,
      totalDomains,
      totalIps,
    };
  }, [sets, setsStats]);

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
      return false;
    });
  }, [sets, filterText]);

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
        syn_ttl: 3,
        drop_sack: false,
        win_mode: "off",
        win_values: [0, 1460, 8192, 65535],
        desync_mode: "off",
        desync_ttl: 3,
        desync_count: 3,
        post_desync: false,
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
      dns: {
        enabled: false,
        target_dns: "",
        fragment_query: false,
      } as B4SetConfig["dns"],
      fragmentation: {
        strategy: "tcp",
        reverse_order: true,
        middle_sni: true,
        sni_position: 1,
        oob_position: 0,
        oob_char: 120,
        tlsrec_pos: 0,
        seq_overlap: 0,
        seq_overlap_pattern: [],
        combo: {
          extension_split: true,
          first_byte_split: true,
          shuffle_mode: "middle",
          first_delay_ms: 100,
          jitter_max_us: 2000,
          decoy_enabled: false,
          decoy_snis: ["ya.ru", "vk.com", "mail.ru"],
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
        payload_file: "",
        tls_mod: [] as string[],
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

  const handleEditSet = (set: B4SetConfig) => {
    setEditDialog({ open: true, set, isNew: false });
  };

  const handleSaveSet = (set: B4SetConfig) => {
    void (async () => {
      const result = editDialog.isNew
        ? await createSet(set)
        : await updateSet(set);

      if (result.success) {
        showSuccess(editDialog.isNew ? "Set created" : "Set updated");
        setEditDialog({ open: false, set: null, isNew: false });
        onRefresh();
      } else {
        showError(result.error || "Failed");
      }
    })();
  };

  const handleDeleteSet = () => {
    if (!deleteDialog.setId) return;
    void (async () => {
      const result = await deleteSet(deleteDialog.setId!);
      if (result.success) {
        showSuccess("Set deleted");
        setDeleteDialog({ open: false, setId: null });
        onRefresh();
      } else {
        showError(result.error || "Failed to delete");
      }
    })();
  };

  const handleDuplicateSet = (set: B4SetConfig) => {
    void (async () => {
      const result = await duplicateSet(set);
      if (result.success) {
        showSuccess("Set duplicated");
        onRefresh();
      } else {
        showError(result.error || "Failed to duplicate");
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
        showError(result.error || "Failed to update");
      }
    })();
  };

  return (
    <Stack spacing={3}>
      <B4Section
        title="Configuration Sets"
        description="Manage bypass configurations for different domains and scenarios"
        icon={<SetsIcon />}
      >
        {/* Summary Stats Bar */}
        <Paper
          elevation={0}
          sx={{
            p: 2,
            mb: 3,
            bgcolor: colors.background.dark,
            border: `1px solid ${colors.border.default}`,
            borderRadius: radius.md,
          }}
        >
          <Stack
            direction="row"
            spacing={4}
            alignItems="center"
            justifyContent="space-between"
            flexWrap="wrap"
            useFlexGap
          >
            <Stack direction="row" spacing={4}>
              <StatItem
                value={summaryStats.total}
                label="total sets"
                color={colors.text.primary}
              />
              <StatItem
                value={summaryStats.enabled}
                label="enabled"
                color={colors.tertiary}
                icon={<CheckIcon sx={{ fontSize: 16 }} />}
              />
              <StatItem
                value={summaryStats.totalDomains.toLocaleString()}
                label="domains"
                color={colors.secondary}
                icon={<DomainIcon sx={{ fontSize: 16 }} />}
              />
            </Stack>

            {/* Search & Add */}
            <Stack direction="row" spacing={2}>
              <TextField
                size="small"
                placeholder="Search sets..."
                value={filterText}
                onChange={(e) => setFilterText(e.target.value)}
                InputProps={{
                  startAdornment: (
                    <InputAdornment position="start">
                      <SearchOutlinedIcon
                        sx={{ fontSize: 20, color: colors.text.secondary }}
                      />
                    </InputAdornment>
                  ),
                }}
                sx={{
                  width: 200,
                  "& .MuiOutlinedInput-root": {
                    bgcolor: colors.background.paper,
                  },
                }}
              />
              <Button
                startIcon={<AddIcon />}
                onClick={handleAddSet}
                variant="contained"
              >
                Create Set
              </Button>
            </Stack>
          </Stack>
        </Paper>

        {/* Cards Grid */}
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragStart={handleDragStart}
          onDragEnd={handleDragEnd}
        >
          <SortableContext
            items={filteredSets.map((s) => s.id)}
            strategy={rectSortingStrategy}
          >
            <Grid container spacing={3}>
              {filteredSets.map((set) => {
                const index = sets.findIndex((s) => s.id === set.id);
                const stats = setsStats[index] || undefined;

                return (
                  <Grid key={set.id} size={{ xs: 12, sm: 6, lg: 4, xl: 3 }}>
                    <SortableCardWrapper id={set.id}>
                      {(
                        dragHandleProps: React.HTMLAttributes<HTMLDivElement>
                      ) => (
                        <SetCard
                          set={set}
                          stats={stats}
                          index={index}
                          onEdit={() => handleEditSet(set)}
                          onDuplicate={() => handleDuplicateSet(set)}
                          onCompare={() =>
                            setCompareDialog({
                              open: true,
                              setA: set,
                              setB: null,
                            })
                          }
                          onDelete={() =>
                            setDeleteDialog({ open: true, setId: set.id })
                          }
                          onToggleEnabled={(enabled) =>
                            handleToggleEnabled(set, enabled)
                          }
                          dragHandleProps={dragHandleProps}
                        />
                      )}
                    </SortableCardWrapper>
                  </Grid>
                );
              })}
            </Grid>
          </SortableContext>

          <DragOverlay>
            {activeSet ? (
              <Box
                sx={{
                  p: 3,
                  bgcolor: colors.background.paper,
                  border: `2px solid ${colors.secondary}`,
                  borderRadius: radius.md,
                  boxShadow: `0 16px 48px ${colors.accent.primary}60`,
                  minWidth: 280,
                }}
              >
                <Typography variant="h6" fontWeight={600}>
                  {activeSet.name}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  {activeSet.fragmentation.strategy.toUpperCase()}
                </Typography>
              </Box>
            ) : null}
          </DragOverlay>
        </DndContext>

        {/* Empty state */}
        {filteredSets.length === 0 && filterText && (
          <Paper
            elevation={0}
            sx={{
              p: 4,
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
      </B4Section>

      {/* Edit Dialog */}
      <SetEditor
        open={editDialog.open}
        settings={config.system}
        set={editDialog.set!}
        config={config}
        isNew={editDialog.isNew}
        saving={saving}
        stats={
          setsStats[sets.findIndex((s) => s.id === editDialog.set?.id)] ||
          undefined
        }
        onClose={() => setEditDialog({ open: false, set: null, isNew: false })}
        onSave={handleSaveSet}
      />

      {/* Delete Confirmation */}
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
            >
              Cancel
            </Button>
            <Box sx={{ flex: 1 }} />
            <Button onClick={handleDeleteSet} variant="contained" color="error">
              Delete Set
            </Button>
          </>
        }
      >
        <Typography>
          Are you sure you want to delete{" "}
          <strong>{sets.find((s) => s.id === deleteDialog.setId)?.name}</strong>
          ?
        </Typography>
      </B4Dialog>

      {/* Compare Selection Dialog */}
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
    </Stack>
  );
};

interface StatItemProps {
  value: string | number;
  label: string;
  color: string;
  icon?: React.ReactNode;
}

const StatItem = ({ value, label, color, icon }: StatItemProps) => (
  <Stack direction="row" alignItems="center" spacing={1}>
    {icon && <Box sx={{ color, display: "flex" }}>{icon}</Box>}
    <Typography variant="h5" fontWeight={700} sx={{ color }}>
      {value}
    </Typography>
    <Typography variant="body2" color="text.secondary">
      {label}
    </Typography>
  </Stack>
);
