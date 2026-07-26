"use client";

import { useMemo, useState } from "react";
import { UserPlus } from "lucide-react";
import { toast } from "sonner";
import type { FetchBaseQueryError } from "@reduxjs/toolkit/query";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { DataTable, type Column } from "@/components/table/DataTable";
import {
  useEnrollStudentsMutation,
  useListBatchStudentsQuery,
} from "@/lib/api/batchesApi";
import { useListStudentsQuery } from "@/lib/api/studentsApi";
import { apiErrorMessage } from "@/lib/api-error";
import type { Student } from "@/lib/types";

const PAGE_SIZE = 10;

export function EnrollStudentsDrawer({ batchId }: { batchId: number }) {
  const [open, setOpen] = useState(false);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [selected, setSelected] = useState<Set<number>>(new Set());

  const { data, isLoading, isFetching } = useListStudentsQuery(
    { page, page_size: PAGE_SIZE, search },
    { skip: !open },
  );
  const { data: enrolledData } = useListBatchStudentsQuery(
    { batchId, page: 1, page_size: 200 },
    { skip: !open },
  );
  const enrolledIds = useMemo(
    () => new Set((enrolledData?.items ?? []).map((s) => s.id)),
    [enrolledData],
  );

  const [enrollStudents, { isLoading: enrolling }] = useEnrollStudentsMutation();

  const total = data?.total ?? 0;
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const items = data?.items ?? [];

  function toggle(id: number) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  function reset() {
    setSelected(new Set());
    setSearch("");
    setPage(1);
  }

  async function submit() {
    if (selected.size === 0) return;
    try {
      const result = await enrollStudents({
        batchId,
        studentIds: Array.from(selected),
      }).unwrap();
      toast.success(
        `${result.enrolled} student${result.enrolled === 1 ? "" : "s"} enrolled`,
      );
      setOpen(false);
      reset();
    } catch (err) {
      toast.error(apiErrorMessage(err as FetchBaseQueryError));
    }
  }

  const columns: Column<Student>[] = [
    {
      key: "select",
      header: "",
      headerClassName: "w-0",
      cell: (s) => {
        const already = enrolledIds.has(s.id);
        return (
          <Checkbox
            checked={already || selected.has(s.id)}
            disabled={already}
            onCheckedChange={() => toggle(s.id)}
            aria-label={`Select ${s.first_name} ${s.last_name}`}
          />
        );
      },
    },
    {
      key: "name",
      header: "Name",
      cell: (s) => (
        <span className="flex items-center gap-2">
          <span className="font-medium">
            {s.first_name} {s.last_name}
          </span>
          {enrolledIds.has(s.id) && (
            <Badge variant="secondary" className="text-[10px]">
              Enrolled
            </Badge>
          )}
        </span>
      ),
    },
    { key: "email", header: "Email", cell: (s) => s.email },
    { key: "phone", header: "Phone", cell: (s) => s.phone_no || "—" },
  ];

  return (
    <Sheet
      open={open}
      onOpenChange={(o) => {
        setOpen(o);
        if (!o) reset();
      }}
    >
      <SheetTrigger asChild>
        <Button variant="secondary" size="sm">
          <UserPlus className="mr-2 h-4 w-4" />
          Enroll students
        </Button>
      </SheetTrigger>
      <SheetContent className="flex w-full flex-col sm:max-w-2xl">
        <SheetHeader className="shrink-0">
          <SheetTitle>Enroll students</SheetTitle>
          <SheetDescription>
            Search your organization&apos;s students and select who to enroll
            in this batch.
          </SheetDescription>
        </SheetHeader>

        <div className="mt-4 flex-1 overflow-y-auto">
          <DataTable
            columns={columns}
            data={items}
            getRowId={(s) => s.id}
            isLoading={isLoading}
            isFetching={isFetching}
            emptyMessage="No students found."
            searchPlaceholder="Search by name or email…"
            manualPagination
            searchValue={search}
            onSearchChange={(v) => {
              setSearch(v);
              setPage(1);
            }}
            pageIndex={page}
            onPageChange={setPage}
            pageCount={pageCount}
            totalItems={total}
          />
        </div>

        <SheetFooter className="shrink-0 items-center border-t pt-4 sm:justify-between">
          <span className="text-sm text-muted-foreground">
            {selected.size} selected
          </span>
          <div className="flex gap-2">
            <Button variant="ghost" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button onClick={submit} disabled={selected.size === 0 || enrolling}>
              Enroll selected
            </Button>
          </div>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
