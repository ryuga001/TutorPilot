"use client";

import { useMemo, useState, type ReactNode } from "react";
import { ChevronLeft, ChevronRight, Loader2, Search } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { cn } from "@/lib/utils";

export interface Column<T> {
  key: string;
  header: ReactNode;
  cell: (row: T) => ReactNode;
  className?: string;
  headerClassName?: string;
}

export interface DataTableProps<T> {
  columns: Column<T>[];
  data: T[];

  searchable?: boolean;
  searchPlaceholder?: string;
  /** Client-side filtering only (ignored in manual mode). */
  searchKeys?: (keyof T)[];

  toolbar?: ReactNode;

  /**
   * Server-driven ("manual") mode: `data` is already the current page from
   * the API — internal filtering/slicing is skipped entirely. The caller
   * owns search + page state and passes it in via the props below.
   */
  manualPagination?: boolean;
  /** Controlled search value; required in manual mode. */
  searchValue?: string;
  onSearchChange?: (value: string) => void;
  /** 1-based current page (manual mode). */
  pageIndex?: number;
  onPageChange?: (page: number) => void;
  /** Total pages / total row count across all pages (manual mode). */
  pageCount?: number;
  totalItems?: number;
  /** Shows a small inline spinner next to the pager while a new page loads. */
  isFetching?: boolean;

  /** Client-side page size (ignored in manual mode). */
  pageSize?: number;

  isLoading?: boolean;
  emptyMessage?: string;

  getRowId?: (row: T, index: number) => string | number;
  onRowClick?: (row: T) => void;
}

export function DataTable<T>({
  columns,
  data,
  searchable = true,
  searchPlaceholder = "Search…",
  searchKeys,
  toolbar,
  manualPagination = false,
  searchValue,
  onSearchChange,
  pageIndex,
  onPageChange,
  pageCount: manualPageCount,
  totalItems,
  isFetching = false,
  pageSize = 10,
  isLoading = false,
  emptyMessage = "No results.",
  getRowId,
  onRowClick,
}: DataTableProps<T>) {
  const [internalQuery, setInternalQuery] = useState("");
  const [internalPage, setInternalPage] = useState(0);

  const query = manualPagination ? searchValue ?? "" : internalQuery;

  const filtered = useMemo(() => {
    if (manualPagination) return data;
    if (!searchable || !query.trim()) return data;
    const q = query.trim().toLowerCase();
    return data.filter((row) => {
      const values = searchKeys
        ? searchKeys.map((k) => row[k])
        : Object.values(row as Record<string, unknown>);
      return values.some((v) =>
        String(v ?? "")
          .toLowerCase()
          .includes(q),
      );
    });
  }, [data, query, searchable, searchKeys, manualPagination]);

  const pageCount = manualPagination
    ? Math.max(1, manualPageCount ?? 1)
    : Math.max(1, Math.ceil(filtered.length / pageSize));
  const safePage = manualPagination
    ? Math.max(0, (pageIndex ?? 1) - 1)
    : Math.min(internalPage, pageCount - 1);
  const start = manualPagination ? 0 : safePage * pageSize;
  const rows = manualPagination ? data : filtered.slice(start, start + pageSize);
  const total = manualPagination ? totalItems ?? data.length : filtered.length;

  function handleSearchChange(value: string) {
    if (manualPagination) {
      onSearchChange?.(value);
    } else {
      setInternalQuery(value);
      setInternalPage(0);
    }
  }

  function goToPage(zeroBasedPage: number) {
    if (manualPagination) {
      onPageChange?.(zeroBasedPage + 1);
    } else {
      setInternalPage(zeroBasedPage);
    }
  }

  return (
    <div className="space-y-3">
      {(searchable || toolbar) && (
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          {searchable ? (
            <div className="relative w-full sm:max-w-xs">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                value={query}
                onChange={(e) => handleSearchChange(e.target.value)}
                placeholder={searchPlaceholder}
                className="pl-8"
              />
            </div>
          ) : (
            <div />
          )}
          {toolbar && <div className="flex items-center gap-2">{toolbar}</div>}
        </div>
      )}

      <div className="border">
        <Table>
          <TableHeader>
            <TableRow>
              {columns.map((c) => (
                <TableHead key={c.key} className={c.headerClassName}>
                  {c.header}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow>
                <TableCell
                  colSpan={columns.length}
                  className="h-24 text-center text-muted-foreground"
                >
                  <Loader2 className="mx-auto h-5 w-5 animate-spin" />
                </TableCell>
              </TableRow>
            ) : rows.length === 0 ? (
              <TableRow>
                <TableCell
                  colSpan={columns.length}
                  className="h-24 text-center text-muted-foreground"
                >
                  {emptyMessage}
                </TableCell>
              </TableRow>
            ) : (
              rows.map((row, i) => (
                <TableRow
                  key={getRowId ? getRowId(row, start + i) : start + i}
                  onClick={onRowClick ? () => onRowClick(row) : undefined}
                  className={cn(onRowClick && "cursor-pointer")}
                >
                  {columns.map((c) => (
                    <TableCell key={c.key} className={c.className}>
                      {c.cell(row)}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      <div className="flex items-center justify-between text-sm text-muted-foreground">
        <span>
          {total === 0
            ? "0"
            : manualPagination
              ? total
              : `${start + 1}–${Math.min(start + pageSize, total)} of ${total}`}
          {manualPagination && total > 0 ? " total" : ""}
        </span>
        <div className="flex items-center gap-2">
          {isFetching && <Loader2 className="h-4 w-4 animate-spin" />}
          <span>
            Page {safePage + 1} of {pageCount}
          </span>
          <Button
            variant="outline"
            size="icon"
            className="h-8 w-8"
            disabled={safePage <= 0}
            onClick={() => goToPage(safePage - 1)}
            aria-label="Previous page"
          >
            <ChevronLeft className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            size="icon"
            className="h-8 w-8"
            disabled={safePage >= pageCount - 1}
            onClick={() => goToPage(safePage + 1)}
            aria-label="Next page"
          >
            <ChevronRight className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}
