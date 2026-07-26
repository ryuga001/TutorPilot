"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";

import { RequirePrivilege } from "@/components/auth/RequirePrivilege";
import { CreateBatchDialog } from "@/components/dashboard/batches/CreateBatchDialog";
import PageTheme from "@/components/pagetheme/PageTheme";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { DataTable, type Column } from "@/components/table/DataTable";
import { useListBatchesQuery } from "@/lib/api/batchesApi";
import { useListCoursesQuery } from "@/lib/api/coursesApi";
import { useCan } from "@/lib/hooks/useCan";
import type { Batch } from "@/lib/types";

const PAGE_SIZE = 20;

function BatchesList() {
  const can = useCan();
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState<string>("all");
  const [courseFilter, setCourseFilter] = useState<string>("all");

  const { data: courseOptions } = useListCoursesQuery({ page: 1, page_size: 100 });
  const courseTitleById = useMemo(() => {
    const map = new Map<number, string>();
    for (const c of courseOptions?.items ?? []) map.set(c.id, c.title);
    return map;
  }, [courseOptions]);

  const { data, isLoading, isFetching } = useListBatchesQuery({
    page,
    page_size: PAGE_SIZE,
    search,
    status: status === "all" ? undefined : status,
    course_id: courseFilter === "all" ? undefined : Number(courseFilter),
  });

  const total = data?.total ?? 0;
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const items = data?.items ?? [];
  const router = useRouter();

  const columns: Column<Batch>[] = [
    {
      key: "name",
      header: "Batch",
      cell: (b) => <span className="font-medium">{b.name}</span>,
    },
    {
      key: "course",
      header: "Course",
      cell: (b) => courseTitleById.get(b.course_id) ?? `#${b.course_id}`,
    },
    {
      key: "status",
      header: "Status",
      cell: (b) => (
        <Badge variant={b.status === "published" ? "default" : "secondary"}>
          {b.status}
        </Badge>
      ),
    },
    { key: "tutors", header: "Tutors", cell: (b) => b.tutor_count },
    { key: "students", header: "Students", cell: (b) => b.student_count },
    {
      key: "created",
      header: "Created",
      cell: (b) => new Date(b.created_at).toLocaleDateString(),
    },
  ];

  return (
    <PageTheme title="Batches" subtitle="Every scheduled offering, across all courses.">
      <DataTable
        columns={columns}
        data={items}
        getRowId={(b) => b.id}
        isLoading={isLoading}
        isFetching={isFetching}
        emptyMessage="No batches found."
        searchPlaceholder="Search batches…"
        onRowClick={(b) => router.push(`/dashboard/batches/${b.id}`)}
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
        toolbar={
          <>
            <Select
              value={courseFilter}
              onValueChange={(v) => {
                setCourseFilter(v);
                setPage(1);
              }}
            >
              <SelectTrigger className="w-44">
                <SelectValue placeholder="Course" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All courses</SelectItem>
                {(courseOptions?.items ?? []).map((c) => (
                  <SelectItem key={c.id} value={String(c.id)}>
                    {c.title}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Select
              value={status}
              onValueChange={(v) => {
                setStatus(v);
                setPage(1);
              }}
            >
              <SelectTrigger className="w-36">
                <SelectValue placeholder="Status" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All statuses</SelectItem>
                <SelectItem value="draft">Draft</SelectItem>
                <SelectItem value="published">Published</SelectItem>
              </SelectContent>
            </Select>
            {can("batch.create") && <CreateBatchDialog />}
          </>
        }
      />
    </PageTheme>
  );
}

export default function BatchesPage() {
  return (
    <RequirePrivilege need="batch.view">
      <BatchesList />
    </RequirePrivilege>
  );
}
