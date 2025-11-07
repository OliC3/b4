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

  const addDomain = useCallback(async () => {
    if (!modalState.selected) return;

    try {
      const response = await fetch("/api/geosite/domain", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          domain: modalState.selected,
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
        const error = await response.json();
        setSnackbar({
          open: true,
          message: `Failed to add domain: ${error.message}`,
          severity: "error",
        });
      }
    } catch (error) {
      setSnackbar({
        open: true,
        message: `Error adding domain: ${error}`,
        severity: "error",
      });
    }
  }, [modalState.selected, closeModal]);

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

// Hook to parse logs
export function useParsedLogs(lines: string[]): ParsedLog[] {
  return useMemo(() => {
    return lines
      .map(parseSniLogLine)
      .filter((log): log is ParsedLog => log !== null);
  }, [lines]);
}

// Hook to filter logs
export function useFilteredLogs(
  parsedLogs: ParsedLog[],
  filter: string
): ParsedLog[] {
  return useMemo(() => {
    const f = filter.trim().toLowerCase();
    const filters = f
      .split("+")
      .map((s) => s.trim())
      .filter((s) => s.length > 0);

    if (filters.length === 0) {
      return parsedLogs;
    }

    // Group filters by field
    const fieldFilters: Record<string, string[]> = {};
    const globalFilters: string[] = [];

    filters.forEach((filterTerm) => {
      const colonIndex = filterTerm.indexOf(":");

      if (colonIndex > 0) {
        const field = filterTerm.substring(0, colonIndex);
        const value = filterTerm.substring(colonIndex + 1);

        if (!fieldFilters[field]) {
          fieldFilters[field] = [];
        }
        fieldFilters[field].push(value);
      } else {
        globalFilters.push(filterTerm);
      }
    });

    return parsedLogs.filter((log) => {
      // Check field-specific filters (OR within field, AND across fields)
      for (const [field, values] of Object.entries(fieldFilters)) {
        const fieldValue =
          log[field as keyof typeof log]?.toString().toLowerCase() || "";
        const matches = values.some((value) => fieldValue.includes(value));
        if (!matches) return false;
      }

      // Check global filters (must match at least one field)
      for (const filterTerm of globalFilters) {
        const matches =
          log.domain.toLowerCase().includes(filterTerm) ||
          log.source.toLowerCase().includes(filterTerm) ||
          log.protocol.toLowerCase().includes(filterTerm) ||
          log.destination.toLowerCase().includes(filterTerm);
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

    const sorted = [...filteredLogs].sort((a, b) => {
      let aValue: any = a[sortColumn];
      let bValue: any = b[sortColumn];

      // Handle different data types
      if (sortColumn === "timestamp") {
        aValue = new Date(aValue.replace(/\//g, "-")).getTime();
        bValue = new Date(bValue.replace(/\//g, "-")).getTime();
      } else if (sortColumn === "isTarget") {
        aValue = aValue ? 1 : 0;
        bValue = bValue ? 1 : 0;
      } else if (typeof aValue === "string") {
        aValue = aValue.toLowerCase();
        bValue = bValue.toLowerCase();
      }

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
