"use client";

import { useState } from "react";
import { UserMinus } from "lucide-react";
import { toast } from "sonner";
import type { FetchBaseQueryError } from "@reduxjs/toolkit/query";

import { ImportStudentsDialog } from "@/components/dashboard/batches/ImportStudentsDialog";
import { Button } from "@/components/ui/button";
import { DataTable, type Column } from "@/components/table/DataTable";
import {
  useListBatchStudentsQuery,
  useRemoveBatchStudentMutation,
} from "@/lib/api/batchesApi";
import { apiErrorMessage } from "@/lib/api-error";
import type { StudentSummary } from "@/lib/types";

const PAGE_SIZE = 20;

export function StudentsTab({ batchId, canEdit }: { batchId: number; canEdit: boolean }) {
  const [page, setPage] = useState(1);
  const { data, isLoading, isFetching } = useListBatchStudentsQuery({
    batchId,
    page,
    page_size: PAGE_SIZE,
  });
  const [removeStudent] = useRemoveBatchStudentMutation();

  const total = data?.total ?? 0;
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const items = data?.items ?? [];

  async function handleRemove(s: StudentSummary) {
    if (!window.confirm(`Remove ${s.first_name} ${s.last_name} from this batch?`)) return;
    try {
      await removeStudent({ batchId, studentId: s.id }).unwrap();
      toast.success("Student removed");
    } catch (err) {
      toast.error(apiErrorMessage(err as FetchBaseQueryError));
    }
  }

  const columns: Column<StudentSummary>[] = [
    {
      key: "name",
      header: "Name",
      cell: (s) => (
        <span className="font-medium">
          {s.first_name} {s.last_name}
        </span>
      ),
    },
    { key: "email", header: "Email", cell: (s) => s.email },
    { key: "phone", header: "Phone", cell: (s) => s.phone_no || "—" },
    {
      key: "actions",
      header: "",
      headerClassName: "w-0",
      className: "text-right",
      cell: (s) =>
        canEdit && (
          <Button
            variant="destructive"
            size="icon"
            className="h-8 w-8"
            onClick={() => handleRemove(s)}
            aria-label="Remove student"
          >
            <UserMinus className="h-4 w-4" />
          </Button>
        ),
    },
  ];

  return (
    <DataTable
      columns={columns}
      data={items}
      getRowId={(s) => s.id}
      isLoading={isLoading}
      isFetching={isFetching}
      emptyMessage="No students enrolled yet."
      searchable={false}
      manualPagination
      pageIndex={page}
      onPageChange={setPage}
      pageCount={pageCount}
      totalItems={total}
      toolbar={canEdit ? <ImportStudentsDialog batchId={batchId} /> : undefined}
    />
  );
}
