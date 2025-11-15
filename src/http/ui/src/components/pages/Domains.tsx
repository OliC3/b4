import { useState, useRef, useEffect, useCallback } from "react";
import { Container, Paper, Snackbar, Alert } from "@mui/material";
import { DomainsControlBar } from "@/components/organisms/domains/ControlBar";
import { AddSniModal } from "@/components/organisms/domains/AddSniModal";
import { DomainsTable, SortColumn } from "@organisms/domains/Table";
import { SortDirection } from "@atoms/common/SortableTableCell";
import {
  useDomainActions,
  useParsedLogs,
  useFilteredLogs,
  useSortedLogs,
} from "@hooks/useDomainActions";

import { useIpActions } from "@hooks/useIpActions";
import {
  generateDomainVariants,
  loadSortState,
  saveSortState,
  generateIpVariants,
  MAX_VISIBLE_ROWS,
} from "@utils";
import { colors } from "@design";
import { useWebSocket } from "@ctx/B4WsProvider";
import { AddIpModal } from "../organisms/domains/AddIpModal";
import { B4Config, B4SetConfig } from "@/models/Config";

export default function Domains() {
  const {
    domains,
    pauseDomains,
    showAll,
    setShowAll,
    setPauseDomains,
    clearDomains,
    resetDomainsBadge,
  } = useWebSocket();

  const [filter, setFilter] = useState("");
  const [autoScroll, setAutoScroll] = useState(true);
  const tableRef = useRef<HTMLDivElement | null>(null);
  const [sortColumn, setSortColumn] = useState<SortColumn | null>(() => {
    const saved = loadSortState();
    return saved.column as SortColumn | null;
  });
  const [sortDirection, setSortDirection] = useState<SortDirection>(() => {
    const saved = loadSortState();
    return saved.direction;
  });

  const {
    modalState,
    snackbar,
    openModal,
    closeModal,
    selectVariant,
    addDomain,
    closeSnackbar,
  } = useDomainActions();

  const {
    modalState: modalIpState,
    openModal: openIpModal,
    closeModal: closeIpModal,
    selectVariant: selectIpVariant,
    addIp,
  } = useIpActions();

  useEffect(() => {
    saveSortState(sortColumn, sortDirection);
  }, [sortColumn, sortDirection]);

  useEffect(() => {
    const el = tableRef.current;
    if (el && autoScroll) {
      el.scrollTop = el.scrollHeight;
    }
  }, [domains, autoScroll]);

  const parsedLogs = useParsedLogs(domains, showAll);
  const filteredLogs = useFilteredLogs(parsedLogs, filter);
  const sortedData = useSortedLogs(filteredLogs, sortColumn, sortDirection);
  const [availableSets, setAvailableSets] = useState<B4SetConfig[]>([]);

  const [ipInfoToken, setIpInfoToken] = useState<string>("");

  const fetchSets = useCallback(async () => {
    try {
      const response = await fetch("/api/config");
      if (response.ok) {
        const data = (await response.json()) as B4Config;
        if (data.sets && Array.isArray(data.sets)) {
          setAvailableSets(data.sets);
        }
        if (data.system?.api?.ipinfo_token) {
          setIpInfoToken(data.system.api.ipinfo_token);
        }
      }
    } catch (error) {
      console.error("Failed to fetch sets:", error);
    }
  }, []);

  useEffect(() => {
    void fetchSets();
  }, [fetchSets]);

  const handleScroll = () => {
    const el = tableRef.current;
    if (el) {
      const isAtBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 50;
      setAutoScroll(isAtBottom);
    }
  };

  const handleSort = (column: SortColumn) => {
    setAutoScroll(false);

    if (sortColumn === column) {
      if (sortDirection === "asc") {
        setSortDirection("desc");
      } else if (sortDirection === "desc") {
        setSortDirection(null);
        setSortColumn(null);
      }
    } else {
      setSortColumn(column);
      setSortDirection("asc");
    }
  };

  const handleClearSort = () => {
    setSortColumn(null);
    setSortDirection(null);
    setAutoScroll(true);
  };

  const handleIpClick = (ip: string) => {
    const variants = generateIpVariants(ip);
    openIpModal(ip, variants);
  };

  const handleDomainClick = (domain: string) => {
    const variants = generateDomainVariants(domain);
    openModal(domain, variants);
  };

  const handleHotkeysDown = useCallback(
    (e: KeyboardEvent) => {
      const target = e.target as HTMLElement;
      if (
        target.tagName === "INPUT" ||
        target.tagName === "TEXTAREA" ||
        target.isContentEditable
      ) {
        return;
      }

      if ((e.ctrlKey && e.key === "x") || e.key === "Delete") {
        e.preventDefault();
        clearDomains();
        resetDomainsBadge();
      } else if (e.key === "p" || e.key === "Pause") {
        e.preventDefault();
        setPauseDomains(!pauseDomains);
      }
    },
    [clearDomains, pauseDomains, setPauseDomains, resetDomainsBadge]
  );

  useEffect(() => {
    globalThis.window.addEventListener("keydown", handleHotkeysDown);
    return () => {
      globalThis.window.removeEventListener("keydown", handleHotkeysDown);
    };
  }, [handleHotkeysDown]);

  return (
    <Container
      maxWidth={false}
      sx={{
        flex: 1,
        py: 3,
        px: 3,
        display: "flex",
        flexDirection: "column",
        overflow: "hidden",
      }}
    >
      <Paper
        elevation={0}
        variant="outlined"
        sx={{
          flex: 1,
          display: "flex",
          flexDirection: "column",
          overflow: "hidden",
          border: "1px solid",
          borderColor: pauseDomains
            ? colors.border.strong
            : colors.border.default,
          transition: "border-color 0.3s",
        }}
      >
        <DomainsControlBar
          filter={filter}
          onFilterChange={setFilter}
          totalCount={parsedLogs.length}
          filteredCount={filteredLogs.length}
          sortColumn={sortColumn}
          paused={pauseDomains}
          showAll={showAll}
          onShowAllChange={setShowAll}
          onPauseChange={setPauseDomains}
          onClearSort={handleClearSort}
          onReset={clearDomains}
        />

        <DomainsTable
          data={sortedData.slice(-MAX_VISIBLE_ROWS)}
          sortColumn={sortColumn}
          sortDirection={sortDirection}
          onSort={handleSort}
          onDomainClick={handleDomainClick}
          onIpClick={handleIpClick}
          tableRef={tableRef}
          onScroll={handleScroll}
        />
      </Paper>

      <AddSniModal
        open={modalState.open}
        domain={modalState.domain}
        variants={modalState.variants}
        selected={modalState.selected}
        onClose={closeModal}
        onSelectVariant={selectVariant}
        sets={availableSets}
        onAdd={(...args) => {
          void (async () => {
            await addDomain(...args);
            await fetchSets();
          })();
        }}
      />

      <AddIpModal
        open={modalIpState.open}
        ip={modalIpState.ip}
        variants={modalIpState.variants}
        selected={modalIpState.selected as string}
        sets={availableSets}
        ipInfoToken={ipInfoToken}
        onClose={closeIpModal}
        onSelectVariant={selectIpVariant}
        onAdd={(...args) => {
          void (async () => {
            await addIp(...args);
            await fetchSets();
          })();
        }}
        onAddHostname={(hostname) => {
          const variants = generateDomainVariants(hostname);
          openModal(hostname, variants);
        }}
      />

      <Snackbar
        open={snackbar.open}
        autoHideDuration={6000}
        onClose={closeSnackbar}
        anchorOrigin={{ vertical: "bottom", horizontal: "right" }}
      >
        <Alert
          onClose={closeSnackbar}
          severity={snackbar.severity}
          sx={{ width: "100%" }}
        >
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Container>
  );
}
