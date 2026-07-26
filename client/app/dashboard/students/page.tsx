"use client";

import { useState } from "react";
import { Pencil, PlusCircle, Trash2, User } from "lucide-react";
import { toast } from "sonner";
import type { FetchBaseQueryError } from "@reduxjs/toolkit/query";

import { RequirePrivilege } from "@/components/auth/RequirePrivilege";
import {
  PersonFormDrawer,
  type PersonFormValues,
} from "@/components/dashboard/people/PersonFormDrawer";
import PageTheme from "@/components/pagetheme/PageTheme";
import { Button } from "@/components/ui/button";
import { DataTable, type Column } from "@/components/table/DataTable";
import {
  useCreateStudentMutation,
  useDeleteStudentMutation,
  useListStudentsQuery,
  useUpdateStudentMutation,
  useUploadStudentProfileImageMutation,
} from "@/lib/api/studentsApi";
import { apiErrorMessage } from "@/lib/api-error";
import { useCan } from "@/lib/hooks/useCan";
import type { Student } from "@/lib/types";

const PAGE_SIZE = 20;

function toastErr(err: unknown) {
  toast.error(apiErrorMessage(err as FetchBaseQueryError));
}

function StudentsList() {
  const can = useCan();
  const canCreate = can("student.create");
  const canEdit = can("student.edit");
  const canDelete = can("student.delete");

  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const { data, isLoading, isFetching } = useListStudentsQuery({
    page,
    page_size: PAGE_SIZE,
    search,
  });

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [editing, setEditing] = useState<Student | null>(null);

  const [createStudent, { isLoading: creating }] = useCreateStudentMutation();
  const [updateStudent, { isLoading: updating }] = useUpdateStudentMutation();
  const [deleteStudent] = useDeleteStudentMutation();
  const [uploadImage, { isLoading: uploadingImage }] =
    useUploadStudentProfileImageMutation();

  const total = data?.total ?? 0;
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const items = data?.items ?? [];

  function openCreate() {
    setEditing(null);
    setDrawerOpen(true);
  }

  function openEdit(s: Student) {
    setEditing(s);
    setDrawerOpen(true);
  }

  async function handleSubmit(values: PersonFormValues) {
    try {
      if (editing) {
        await updateStudent({ id: editing.id, body: values }).unwrap();
        toast.success("Student updated");
      } else {
        await createStudent(values).unwrap();
        toast.success("Student created");
      }
      setDrawerOpen(false);
    } catch (err) {
      toastErr(err);
    }
  }

  async function handleUploadImage(file: File) {
    if (!editing) return;
    try {
      const updated = await uploadImage({ id: editing.id, file }).unwrap();
      setEditing(updated);
      toast.success("Photo updated");
    } catch (err) {
      toastErr(err);
    }
  }

  async function handleDelete(s: Student) {
    if (!window.confirm(`Delete ${s.first_name} ${s.last_name}?`)) return;
    try {
      await deleteStudent(s.id).unwrap();
      toast.success("Student deleted");
    } catch (err) {
      toastErr(err);
    }
  }

  const columns: Column<Student>[] = [
    {
      key: "avatar",
      header: "",
      headerClassName: "w-0",
      cell: (s) => (
        <div className="flex h-8 w-8 items-center justify-center overflow-hidden border bg-muted">
          {s.profile_image_url ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={s.profile_image_url}
              alt=""
              className="h-full w-full object-cover"
            />
          ) : (
            <User className="h-4 w-4 text-muted-foreground" />
          )}
        </div>
      ),
    },
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
      cell: (s) => (
        <div className="flex justify-end gap-1">
          {canEdit && (
            <Button
              variant="secondary"
              size="icon"
              className="h-8 w-8"
              onClick={() => openEdit(s)}
              aria-label="Edit student"
            >
              <Pencil className="h-4 w-4" />
            </Button>
          )}
          {canDelete && (
            <Button
              variant="destructive"
              size="icon"
              className="h-8 w-8"
              onClick={() => handleDelete(s)}
              aria-label="Delete student"
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          )}
        </div>
      ),
    },
  ];

  return (
    <PageTheme title="Students" subtitle="Manage your organization's students.">
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
        toolbar={
          canCreate ? (
            <Button size="sm" onClick={openCreate}>
              <PlusCircle className="mr-2 h-4 w-4" />
              Add student
            </Button>
          ) : undefined
        }
      />

      <PersonFormDrawer
        open={drawerOpen}
        onOpenChange={setDrawerOpen}
        title={editing ? "Edit student" : "Add student"}
        description={
          editing
            ? "Update this student's details."
            : "Create a new student profile."
        }
        initialValues={
          editing
            ? {
                first_name: editing.first_name,
                last_name: editing.last_name,
                email: editing.email,
                phone_no: editing.phone_no,
                address: {
                  local_address: editing.address?.local_address ?? "",
                  city: editing.address?.city ?? "",
                  state: editing.address?.state ?? "",
                  country: editing.address?.country ?? "",
                },
              }
            : undefined
        }
        onSubmit={handleSubmit}
        isSubmitting={creating || updating}
        profileImageUrl={editing?.profile_image_url}
        onUploadImage={editing ? handleUploadImage : undefined}
        uploadingImage={uploadingImage}
      />
    </PageTheme>
  );
}

export default function StudentsPage() {
  return (
    <RequirePrivilege need="student.view">
      <StudentsList />
    </RequirePrivilege>
  );
}
