import React, { useState, useRef, useEffect, useCallback } from "react";
import { Container, Paper, Snackbar, Alert } from "@mui/material";
import { DomainsControlBar } from "../../components/molecules/domains/ControlBar";
import { DomainAddModal } from "../organisms/domains/AddModal";
import {
  DomainsTable,
  SortColumn,
} from "../../components/organisms/domains/Table";
import { SortDirection } from "../../components/atoms/common/SortableTableCell";
import { useDomainActions } from "../../hooks/useDomainActions";
import { useDomainsWebSocket } from "../../hooks/useDomainsWebSocket";
import {
  useParsedLogs,
  useFilteredLogs,
  useSortedLogs,
  generateDomainVariants,
  loadPersistedLines,
  persistLines,
  clearPersistedLines,
} from "../../utils/domains";
import { colors } from "../../Theme";

export default function Domains() {
  // State
  const [lines, setLines] = useState<string[]>(loadPersistedLines);
  const [paused, setPaused] = useState(false);
  const [filter, setFilter] = useState("");
  const [autoScroll, setAutoScroll] = useState(true);
  const [sortColumn, setSortColumn] = useState<SortColumn | null>(null);
  const [sortDirection, setSortDirection] = useState<SortDirection>(null);

  // Refs
  const tableRef = useRef<HTMLDivElement | null>(null);

  // Custom hooks
  const {
    modalState,
    snackbar,
    openModal,
    closeModal,
    selectVariant,
    addDomain,
    closeSnackbar,
  } = useDomainActions();

  // Persist lines to localStorage
  useEffect(() => {
    persistLines(lines);
  }, [lines]);

  // WebSocket connection
  const handleWebSocketMessage = useCallback((line: string) => {
    setLines((prev) => [...prev.slice(-999), line]);
  }, []);

  const handleWebSocketError = useCallback(() => {
    setLines((prev) => [...prev, "[WS ERROR]"]);
  }, []);

  useDomainsWebSocket({
    paused,
    onMessage: handleWebSocketMessage,
    onError: handleWebSocketError,
  });

  // Auto-scroll effect for new data
  useEffect(() => {
    const el = tableRef.current;
    if (el && autoScroll) {
      el.scrollTop = el.scrollHeight;
    }
  }, [lines, autoScroll]);

  // // Auto-scroll when sorting changes
  // useEffect(() => {
  //   const el = tableRef.current;
  //   if (el && sortColumn && sortDirection) {
  //     setTimeout(() => {
  //       if (sortDirection === "asc") {
  //         el.scrollTo({ top: 0, behavior: "smooth" });
  //       } else if (sortDirection === "desc") {
  //         el.scrollTo({ top: el.scrollHeight, behavior: "smooth" });
  //       }
  //     }, 50);
  //   }
  // }, [sortColumn, sortDirection]);

  // Data processing
  const parsedLogs = useParsedLogs(lines);
  const filteredLogs = useFilteredLogs(parsedLogs, filter);
  const sortedData = useSortedLogs(filteredLogs, sortColumn, sortDirection);

  // Handlers
  const handleScroll = () => {
    const el = tableRef.current;
    if (el) {
      const isAtBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 50;
      setAutoScroll(isAtBottom);
    }
  };

  const handleSort = (column: SortColumn) => {
    // Disable auto-scroll when user manually sorts
    setAutoScroll(false);

    if (sortColumn === column) {
      // Cycle through: asc -> desc -> null
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

  const handleDomainClick = (domain: string) => {
    const variants = generateDomainVariants(domain);
    openModal(domain, variants);
  };

  const handleReset = () => {
    setLines([]);
    clearPersistedLines();
  };

  const handleHotkeysDown = (e: KeyboardEvent) => {
    if ((e.ctrlKey && e.key === "x") || e.key === "Delete") {
      // Ctrl+X to clear logs
      e.preventDefault();
      handleReset();
    } else if (e.key === "p" || e.key === "Pause") {
      // P or Pause to toggle pause
      e.preventDefault();
      setPaused((prev) => !prev);
    }
  };

  useEffect(() => {
    window.addEventListener("keydown", handleHotkeysDown);
    return () => {
      window.removeEventListener("keydown", handleHotkeysDown);
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
          borderColor: paused ? colors.border.strong : colors.border.default,
          transition: "border-color 0.3s",
        }}
      >
        {/* Control Bar */}
        <DomainsControlBar
          filter={filter}
          onFilterChange={setFilter}
          totalCount={parsedLogs.length}
          filteredCount={filteredLogs.length}
          sortColumn={sortColumn}
          paused={paused}
          onPauseChange={setPaused}
          onClearSort={handleClearSort}
          onReset={handleReset}
        />

        {/* Domains Table */}
        <DomainsTable
          data={sortedData}
          sortColumn={sortColumn}
          sortDirection={sortDirection}
          onSort={handleSort}
          onDomainClick={handleDomainClick}
          tableRef={tableRef}
          onScroll={handleScroll}
        />
      </Paper>

      {/* Domain Add Modal */}
      <DomainAddModal
        open={modalState.open}
        domain={modalState.domain}
        variants={modalState.variants}
        selected={modalState.selected}
        onClose={closeModal}
        onSelectVariant={selectVariant}
        onAdd={addDomain}
      />

      {/* Snackbar for notifications */}
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
