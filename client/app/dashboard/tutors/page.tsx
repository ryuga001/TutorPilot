"use client";

import { useState } from "react";
import { Pencil, PlusCircle, Trash2, Users } from "lucide-react";
import { toast } from "sonner";
import type { FetchBaseQueryError } from "@reduxjs/toolkit/query";

import { RequirePrivilege } from "@/components/auth/RequirePrivilege";
import {
  PersonFormDrawer,
  type PersonFormValues,
} from "@/components/dashboard/people/PersonFormDrawer";
import { CredentialsDialog } from "@/components/dashboard/people/CredentialsDialog";
import PageTheme from "@/components/pagetheme/PageTheme";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { DataTable, type Column } from "@/components/table/DataTable";
import {
  useCreateTutorMutation,
  useDeleteTutorMutation,
  useListTutorsQuery,
  useUpdateTutorMutation,
  useUploadTutorProfileImageMutation,
} from "@/lib/api/tutorsApi";
import { apiErrorMessage } from "@/lib/api-error";
import { useConfirm } from "@/components/providers/confirm-provider";
import { useCan } from "@/lib/hooks/useCan";
import type { Tutor } from "@/lib/types";

const PAGE_SIZE = 20;

function toastErr(err: unknown) {
  toast.error(apiErrorMessage(err as FetchBaseQueryError));
}

function TutorsList() {
  const can = useCan();
  const confirm = useConfirm();
  const canCreate = can("tutor.create");
  const canEdit = can("tutor.edit");
  const canDelete = can("tutor.delete");

  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const { data, isLoading, isFetching } = useListTutorsQuery({
    page,
    page_size: PAGE_SIZE,
    search,
  });

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [editing, setEditing] = useState<Tutor | null>(null);

  // Shown once, right after a tutor (and their login) is created.
  const [credentials, setCredentials] = useState<{
    email: string;
    tempPassword: string;
    name: string;
  } | null>(null);

  const [createTutor, { isLoading: creating }] = useCreateTutorMutation();
  const [updateTutor, { isLoading: updating }] = useUpdateTutorMutation();
  const [deleteTutor] = useDeleteTutorMutation();
  const [uploadImage, { isLoading: uploadingImage }] =
    useUploadTutorProfileImageMutation();

  const total = data?.total ?? 0;
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const items = data?.items ?? [];

  function openCreate() {
    setEditing(null);
    setDrawerOpen(true);
  }

  function openEdit(t: Tutor) {
    setEditing(t);
    setDrawerOpen(true);
  }

  async function handleSubmit(values: PersonFormValues) {
    try {
      if (editing) {
        await updateTutor({ id: editing.id, body: values }).unwrap();
        toast.success("Tutor updated");
      } else {
        // Creating a tutor also creates the login they sign in with, and the
        // response carries the temporary password once.
        const created = await createTutor(values).unwrap();
        if (created.temp_password) {
          setCredentials({
            email: created.email,
            tempPassword: created.temp_password,
            name: `${created.first_name} ${created.last_name}`,
          });
        }
        toast.success("Tutor created");
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

  async function handleDelete(t: Tutor) {
    const ok = await confirm({
      title: `Delete ${t.first_name} ${t.last_name}?`,
      description: "This cannot be undone.",
    });
    if (!ok) return;
    try {
      await deleteTutor(t.id).unwrap();
      toast.success("Tutor deleted");
    } catch (err) {
      toastErr(err);
    }
  }

  const columns: Column<Tutor>[] = [
    {
      key: "name",
      header: "Name",
      cell: (t) => (
        <div className="flex items-center gap-3">
          <Avatar className="h-8 w-8">
            {t.profile_image_url && <AvatarImage src={t.profile_image_url} alt="" />}
            <AvatarFallback>
              {(t.first_name?.[0] ?? t.email[0] ?? "?").toUpperCase()}
            </AvatarFallback>
          </Avatar>
          <span className="font-medium">
            {t.first_name} {t.last_name}
          </span>
        </div>
      ),
    },
    { key: "email", header: "Email", cell: (t) => t.email },
    { key: "phone", header: "Phone", cell: (t) => t.phone_no || "—" },
    {
      key: "designation",
      header: "Designation",
      cell: (t) => t.designation || "—",
    },
    {
      key: "actions",
      header: "",
      headerClassName: "w-0",
      className: "text-right",
      cell: (t) => (
        <div className="flex justify-end gap-1">
          {canEdit && (
            <Button
              variant="secondary"
              size="icon"
              className="h-8 w-8"
              onClick={() => openEdit(t)}
              aria-label="Edit tutor"
            >
              <Pencil className="h-4 w-4" />
            </Button>
          )}
          {canDelete && (
            <Button
              variant="destructive"
              size="icon"
              className="h-8 w-8"
              onClick={() => handleDelete(t)}
              aria-label="Delete tutor"
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          )}
        </div>
      ),
    },
  ];

  return (
    <PageTheme icon={Users} title="Tutors" subtitle="Manage your organization's tutors.">
      <DataTable
        columns={columns}
        data={items}
        getRowId={(t) => t.id}
        isLoading={isLoading}
        isFetching={isFetching}
        emptyMessage="No tutors found"
        emptyDescription="Add your first tutor to get started."
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
              Add tutor
            </Button>
          ) : undefined
        }
      />

      <PersonFormDrawer
        open={drawerOpen}
        onOpenChange={setDrawerOpen}
        title={editing ? "Edit tutor" : "Add tutor"}
        description={
          editing
            ? "Update this tutor's details."
            : "Create a new tutor profile."
        }
        showDesignation
        initialValues={
          editing
            ? {
                first_name: editing.first_name,
                last_name: editing.last_name,
                email: editing.email,
                phone_no: editing.phone_no,
                designation: editing.designation,
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

      <CredentialsDialog
        tempPassword={credentials?.tempPassword ?? null}
        email={credentials?.email}
        personName={credentials?.name}
        onClose={() => setCredentials(null)}
      />
    </PageTheme>
  );
}

export default function TutorsPage() {
  return (
    <RequirePrivilege need="tutor.view">
      <TutorsList />
    </RequirePrivilege>
  );
}
