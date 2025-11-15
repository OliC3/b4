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
  Box,
} from "@mui/material";
import { Add as AddIcon } from "@mui/icons-material";
import {
  SortableTableCell,
  SortDirection,
} from "@atoms/common/SortableTableCell";
import { ProtocolChip } from "@atoms/common/ProtocolChip";
import { colors } from "@design";
import { B4Badge } from "@/components/atoms/common/B4Badge";

export type SortColumn =
  | "timestamp"
  | "set"
  | "protocol"
  | "domain"
  | "source"
  | "destination";

export interface ParsedLog {
  timestamp: string;
  protocol: "TCP" | "UDP";
  hostSet: string;
  ipSet: string;
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
  onIpClick: (ip: string) => void;
  tableRef: React.RefObject<HTMLDivElement>;
  onScroll: () => void;
}

export const DomainsTable: React.FC<DomainsTableProps> = ({
  data,
  sortColumn,
  sortDirection,
  onSort,
  onDomainClick,
  onIpClick,
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
            />{" "}
            <SortableTableCell
              label="Set"
              active={sortColumn === "set"}
              direction={sortColumn === "set" ? sortDirection : null}
              onSort={() => onSort("set")}
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
            data.map((log) => (
              <TableRow
                key={log.raw}
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
                    borderBottom: `1px solid ${colors.border.light}`,
                  }}
                >
                  {(log.ipSet || log.hostSet) && (
                    <B4Badge
                      badgeVariant="secondary"
                      label={log.ipSet || log.hostSet}
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
                >
                  <Stack
                    direction="row"
                    spacing={1}
                    alignItems="center"
                    onClick={() =>
                      log.domain && !log.hostSet && onDomainClick(log.domain)
                    }
                  >
                    {log.domain && <Typography>{log.domain}</Typography>}
                    <Box sx={{ flex: 1 }} />
                    {log.domain &&
                      (log.hostSet ? (
                        <B4Badge badgeVariant="secondary" label={log.hostSet} />
                      ) : (
                        <AddIcon
                          sx={{
                            fontSize: 16,
                            bgcolor: `${colors.secondary}88`,
                            color: colors.background.default,
                            borderRadius: "50%",
                          }}
                        />
                      ))}
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
                    color: "text.primary",
                    fontWeight: 500,
                    borderBottom: `1px solid ${colors.border.light}`,
                  }}
                >
                  <Stack direction="row" spacing={1} alignItems="center">
                    <Box
                      sx={{
                        cursor: "pointer",
                        "&:hover": {
                          bgcolor: colors.accent.primary,
                          color: colors.secondary,
                        },
                      }}
                      onClick={() =>
                        log.destination &&
                        !log.ipSet &&
                        onIpClick(log.destination)
                      }
                    >
                      {log.destination}
                    </Box>
                    <Box sx={{ flex: 1 }} />
                    {!log.ipSet && (
                      <AddIcon
                        onClick={() => onIpClick(log.destination)}
                        sx={{
                          fontSize: 16,
                          bgcolor: `${colors.secondary}88`,
                          color: colors.background.default,
                          borderRadius: "50%",
                          "&:hover": {
                            bgcolor: colors.secondary,
                          },
                        }}
                      />
                    )}
                  </Stack>
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </TableContainer>
  );
};
