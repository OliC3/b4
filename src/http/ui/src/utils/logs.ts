import { ParsedLog } from "@organisms/domains/Table";

export const SORT_STORAGE_KEY = "b4_domains_sort";

export interface DomainSortState {
  column: string | null;
  direction: "asc" | "desc" | null;
}

export function loadSortState(): DomainSortState {
  try {
    const stored = localStorage.getItem(SORT_STORAGE_KEY);
    if (stored) {
      return JSON.parse(stored) as DomainSortState;
    }
  } catch (e) {
    console.error("Failed to load sort state:", e);
  }
  return { column: null, direction: null };
}

export function saveSortState(
  column: string | null,
  direction: "asc" | "desc" | null
): void {
  try {
    localStorage.setItem(
      SORT_STORAGE_KEY,
      JSON.stringify({ column, direction })
    );
  } catch (e) {
    console.error("Failed to save sort state:", e);
  }
}

export function parseSniLogLine(line: string): ParsedLog | null {
  // Example: 2025/10/13 22:41:12.466126 [INFO] SNI TCP: assets.alicdn.com 192.168.1.100:38894 -> 92.123.206.67:443
  const regex =
    /^(\d{4}\/\d{2}\/\d{2} \d{2}:\d{2}:\d{2}\.\d+)\s+\[INFO\]\s+(TCP|UDP):\s+(\S+)?\|(\S+)\s+(\S+)\s+->\s+(\S+)\|(\S+)$/;
  const match = regex.exec(line);

  if (!match) return null;

  const [, timestamp, protocol, hostSet, domain, source, ipSet, destination] =
    match;

  return {
    timestamp,
    protocol: protocol as "TCP" | "UDP",
    hostSet,
    domain,
    source,
    ipSet,
    destination,
    raw: line,
  };
}

// Generate domain variants from most specific to least specific
export function generateDomainVariants(domain: string): string[] {
  const parts = domain.split(".");
  const variants: string[] = [];

  // Generate from full domain to TLD+1 (e.g., example.com)
  for (let i = 0; i < parts.length - 1; i++) {
    variants.push(parts.slice(i).join("."));
  }

  return variants;
}

// Local storage utilities
export const STORAGE_KEY = "b4_domains_lines";
export const MAX_STORED_LINES = 1000;

export function loadPersistedLines(): string[] {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      const parsed = JSON.parse(stored) as unknown;
      return Array.isArray(parsed) ? (parsed as string[]) : [];
    }
  } catch (e) {
    console.error("Failed to load persisted domains:", e);
  }
  return [];
}

export function persistLogLines(lines: string[]): void {
  try {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify(lines.slice(-MAX_STORED_LINES))
    );
  } catch (e) {
    console.error("Failed to persist domains:", e);
  }
}

export function clearLogPersistedLines(): void {
  localStorage.removeItem(STORAGE_KEY);
}
