import { useState, useCallback, useMemo } from "react";
import { ParsedLog, SortColumn } from "@organisms/domains/Table";
import { SortDirection } from "@atoms/common/SortableTableCell";
import { parseSniLogLine } from "@utils";

interface DomainModalState {
  open: boolean;
  domain: string;
  variants: string[];
  selected: string;
}

interface SnackbarState {
  open: boolean;
  message: string;
  severity: "success" | "error";
}

export function useDomainActions() {
  const [modalState, setModalState] = useState<DomainModalState>({
    open: false,
    domain: "",
    variants: [],
    selected: "",
  });

  const [snackbar, setSnackbar] = useState<SnackbarState>({
    open: false,
    message: "",
    severity: "success",
  });

  const openModal = useCallback((domain: string, variants: string[]) => {
    setModalState({
      open: true,
      domain,
      variants,
      selected: variants[0] || domain,
    });
  }, []);

  const closeModal = useCallback(() => {
    setModalState({
      open: false,
      domain: "",
      variants: [],
      selected: "",
    });
  }, []);

  const selectVariant = useCallback((variant: string) => {
    setModalState((prev) => ({ ...prev, selected: variant }));
  }, []);

  const addDomain = useCallback(
    async (setId: string) => {
      if (!modalState.selected) return;

      try {
        const response = await fetch("/api/geosite/domain", {
          method: "PUT",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            domain: modalState.selected,
            set_id: setId,
          }),
        });

        if (response.ok) {
          setSnackbar({
            open: true,
            message: `Successfully added "${modalState.selected}" to manual domains`,
            severity: "success",
          });
          closeModal();
        } else {
          const error = (await response.json()) as { message: string };
          setSnackbar({
            open: true,
            message: `Failed to add domain: ${error.message}`,
            severity: "error",
          });
        }
      } catch (error) {
        setSnackbar({
          open: true,
          message: `Error adding domain: ${String(error)}`,
          severity: "error",
        });
      }
    },
    [modalState.selected, closeModal]
  );

  const closeSnackbar = useCallback(() => {
    setSnackbar((prev) => ({ ...prev, open: false }));
  }, []);

  return {
    modalState,
    snackbar,
    openModal,
    closeModal,
    selectVariant,
    addDomain,
    closeSnackbar,
  };
}

const parseCache = new WeakMap<string[], ParsedLog[]>();

// Hook to parse logs
export function useParsedLogs(lines: string[], showAll: boolean): ParsedLog[] {
  return useMemo(() => {
    // ADD cache check
    if (parseCache.has(lines)) {
      const cached = parseCache.get(lines)!;
      return showAll ? cached : cached.filter((log) => log.domain !== "");
    }

    const parsed = lines
      .map(parseSniLogLine)
      .filter((log): log is ParsedLog => log !== null);

    parseCache.set(lines, parsed);
    return showAll ? parsed : parsed.filter((log) => log.domain !== "");
  }, [lines, showAll]);
}

export function useFilteredLogs(
  parsedLogs: ParsedLog[],
  filter: string
): ParsedLog[] {
  return useMemo(() => {
    const f = filter.trim().toLowerCase();
    if (!f) return parsedLogs;

    const filters = f
      .split("+")
      .map((s) => s.trim())
      .filter((s) => s.length > 0);
    if (filters.length === 0) return parsedLogs;

    const fieldFilters: Record<string, string[]> = {};
    const globalFilters: string[] = [];

    for (const filterTerm of filters) {
      const colonIndex = filterTerm.indexOf(":");
      if (colonIndex > 0) {
        const field = filterTerm.substring(0, colonIndex);
        const value = filterTerm.substring(colonIndex + 1);
        if (!fieldFilters[field]) fieldFilters[field] = [];
        fieldFilters[field].push(value);
      } else {
        globalFilters.push(filterTerm);
      }
    }

    return parsedLogs.filter((log: ParsedLog) => {
      for (const [field, values] of Object.entries(fieldFilters)) {
        const fieldValue =
          log[field as keyof typeof log]?.toString().toLowerCase() || "";
        if (!values.some((value) => fieldValue.includes(value))) return false;
      }

      for (const filterTerm of globalFilters) {
        const matches = [
          log.hostSet,
          log.ipSet,
          log.domain,
          log.source,
          log.protocol,
          log.destination,
        ].some((value) => value?.toLowerCase().includes(filterTerm));
        if (!matches) return false;
      }

      return true;
    });
  }, [parsedLogs, filter]);
}

// Hook to sort logs
export function useSortedLogs(
  filteredLogs: ParsedLog[],
  sortColumn: SortColumn | null,
  sortDirection: SortDirection
): ParsedLog[] {
  return useMemo(() => {
    if (!sortColumn || !sortDirection) {
      return filteredLogs;
    }

    function normalizeSortValue(
      value: string | number | boolean | undefined,
      column: SortColumn
    ): number | string {
      if (column === "timestamp") {
        const str = typeof value === "string" ? value : "";
        return new Date(str.replaceAll(/\/+/g, "-")).getTime();
      }
      if (typeof value === "string") {
        return value.toLowerCase();
      }
      if (typeof value === "boolean") {
        return value ? 1 : 0;
      }
      return value ?? "";
    }

    const sorted = [...filteredLogs].sort((a, b) => {
      const aValue = normalizeSortValue(
        a[sortColumn as keyof ParsedLog],
        sortColumn
      );
      const bValue = normalizeSortValue(
        b[sortColumn as keyof ParsedLog],
        sortColumn
      );

      if (aValue < bValue) {
        return sortDirection === "asc" ? -1 : 1;
      }
      if (aValue > bValue) {
        return sortDirection === "asc" ? 1 : -1;
      }
      return 0;
    });

    return sorted;
  }, [filteredLogs, sortColumn, sortDirection]);
}
