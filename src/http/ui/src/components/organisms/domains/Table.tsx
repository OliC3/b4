import React from "react";
import {
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
  Stack,
} from "@mui/material";
import CheckCircleIcon from "@mui/icons-material/CheckCircle";
import AddIcon from "@mui/icons-material/Add";
import {
  SortableTableCell,
  SortDirection,
} from "@atoms/common/SortableTableCell";
import { ProtocolChip } from "@atoms/common/ProtocolChip";
import { colors } from "@design";

export type SortColumn =
  | "timestamp"
  | "protocol"
  | "isTarget"
  | "domain"
  | "source"
  | "destination";

export interface ParsedLog {
  timestamp: string;
  protocol: "TCP" | "UDP";
  isTarget: boolean;
  domain: string;
  source: string;
  destination: string;
  raw: string;
}

interface DomainsTableProps {
  data: ParsedLog[];
  sortColumn: SortColumn | null;
  sortDirection: SortDirection;
  onSort: (column: SortColumn) => void;
  onDomainClick: (domain: string) => void;
  tableRef: React.RefObject<HTMLDivElement>;
  onScroll: () => void;
}

export const DomainsTable: React.FC<DomainsTableProps> = ({
  data,
  sortColumn,
  sortDirection,
  onSort,
  onDomainClick,
  tableRef,
  onScroll,
}) => {
  return (
    <TableContainer
      ref={tableRef}
      onScroll={onScroll}
      sx={{
        flex: 1,
        backgroundColor: colors.background.dark,
      }}
    >
      <Table stickyHeader size="small">
        <TableHead>
          <TableRow>
            <SortableTableCell
              label="Time"
              active={sortColumn === "timestamp"}
              direction={sortColumn === "timestamp" ? sortDirection : null}
              onSort={() => onSort("timestamp")}
            />
            <SortableTableCell
              label="Protocol"
              active={sortColumn === "protocol"}
              direction={sortColumn === "protocol" ? sortDirection : null}
              onSort={() => onSort("protocol")}
            />
            <SortableTableCell
              label="Target"
              active={sortColumn === "isTarget"}
              direction={sortColumn === "isTarget" ? sortDirection : null}
              onSort={() => onSort("isTarget")}
              align="center"
            />
            <SortableTableCell
              label="Domain"
              active={sortColumn === "domain"}
              direction={sortColumn === "domain" ? sortDirection : null}
              onSort={() => onSort("domain")}
            />
            <SortableTableCell
              label="Source"
              active={sortColumn === "source"}
              direction={sortColumn === "source" ? sortDirection : null}
              onSort={() => onSort("source")}
            />
            <SortableTableCell
              label="Destination"
              active={sortColumn === "destination"}
              direction={sortColumn === "destination" ? sortDirection : null}
              onSort={() => onSort("destination")}
            />
          </TableRow>
        </TableHead>
        <TableBody>
          {data.length === 0 ? (
            <TableRow>
              <TableCell
                colSpan={6}
                sx={{
                  textAlign: "center",
                  py: 4,
                  color: "text.secondary",
                  fontStyle: "italic",
                  bgcolor: colors.background.dark,
                  borderBottom: "none",
                }}
              >
                Waiting for connections...
              </TableCell>
            </TableRow>
          ) : (
            data.map((log, i) => (
              <TableRow
                key={i}
                sx={{
                  "&:hover": {
                    bgcolor: colors.accent.primaryStrong,
                  },
                }}
              >
                <TableCell
                  sx={{
                    color: "text.secondary",
                    fontFamily: "monospace",
                    fontSize: 12,
                    borderBottom: `1px solid ${colors.border.light}`,
                  }}
                >
                  {log.timestamp.split(" ")[1]}
                </TableCell>
                <TableCell
                  sx={{
                    borderBottom: `1px solid ${colors.border.light}`,
                  }}
                >
                  <ProtocolChip protocol={log.protocol} />
                </TableCell>
                <TableCell
                  sx={{
                    textAlign: "center",
                    borderBottom: `1px solid ${colors.border.light}`,
                  }}
                >
                  {log.isTarget && (
                    <CheckCircleIcon
                      sx={{ color: colors.secondary, fontSize: 18 }}
                    />
                  )}
                </TableCell>
                <TableCell
                  sx={{
                    color: "text.primary",
                    fontWeight: 500,
                    borderBottom: `1px solid ${colors.border.light}`,
                    cursor: "pointer",
                    "&:hover": {
                      bgcolor: colors.accent.primary,
                      color: colors.secondary,
                    },
                  }}
                  onClick={() => onDomainClick(log.domain)}
                >
                  <Stack direction="row" spacing={1} alignItems="center">
                    <Typography>{log.domain}</Typography>
                    <AddIcon sx={{ fontSize: 16, opacity: 0.7 }} />
                  </Stack>
                </TableCell>
                <TableCell
                  sx={{
                    color: "text.secondary",
                    fontFamily: "monospace",
                    fontSize: 12,
                    borderBottom: `1px solid ${colors.border.light}`,
                  }}
                >
                  {log.source}
                </TableCell>
                <TableCell
                  sx={{
                    color: "text.secondary",
                    fontFamily: "monospace",
                    fontSize: 12,
                    borderBottom: `1px solid ${colors.border.light}`,
                  }}
                >
                  {log.destination}
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </TableContainer>
  );
};
