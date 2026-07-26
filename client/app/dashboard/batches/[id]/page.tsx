"use client";

import Link from "next/link";
import { notFound, useParams, useRouter } from "next/navigation";
import { ArrowLeft, Loader2, Trash2 } from "lucide-react";
import { toast } from "sonner";
import type { FetchBaseQueryError } from "@reduxjs/toolkit/query";

import { RequirePrivilege } from "@/components/auth/RequirePrivilege";
import { DriveTab } from "@/components/dashboard/batches/DriveTab";
import { ModulesTutorsTab } from "@/components/dashboard/batches/ModulesTutorsTab";
import { RenameBatchDialog } from "@/components/dashboard/batches/RenameBatchDialog";
import { StudentsTab } from "@/components/dashboard/batches/StudentsTab";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  useDeleteBatchMutation,
  useGetBatchQuery,
  useSetBatchPublishedMutation,
} from "@/lib/api/batchesApi";
import { useGetCourseQuery } from "@/lib/api/coursesApi";
import { apiErrorMessage } from "@/lib/api-error";
import { useCan } from "@/lib/hooks/useCan";

function BatchDetail({ id }: { id: number }) {
  const router = useRouter();
  const can = useCan();
  const canEdit = can("batch.edit");
  const canDelete = can("batch.delete");

  const { data: batch, isLoading, error } = useGetBatchQuery(id);
  const { data: course } = useGetCourseQuery(batch?.course_id ?? 0, { skip: !batch });
  const [setPublished, { isLoading: publishing }] = useSetBatchPublishedMutation();
  const [deleteBatch, { isLoading: deleting }] = useDeleteBatchMutation();

  if (isLoading) {
    return (
      <div className="flex h-40 items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (error) {
    const status = (error as FetchBaseQueryError).status;
    if (status === 404) notFound();
    return <p className="text-sm text-destructive">Could not load this batch.</p>;
  }
  if (!batch) notFound();

  const published = batch.status === "published";

  async function togglePublish() {
    try {
      await setPublished({ id, published: !published }).unwrap();
      toast.success(
        published
          ? "Unpublished"
          : "Published — tutors and students have been emailed their details.",
      );
    } catch (err) {
      toast.error(apiErrorMessage(err as FetchBaseQueryError));
    }
  }

  async function remove() {
    if (!window.confirm("Delete this batch? This cannot be undone.")) return;
    try {
      await deleteBatch(id).unwrap();
      toast.success("Batch deleted");
      router.push(course ? `/dashboard/courses/${course.id}` : "/dashboard/batches");
    } catch (err) {
      toast.error(apiErrorMessage(err as FetchBaseQueryError));
    }
  }

  return (
    <div className="mx-auto w-full">
      <div className="mb-6 flex items-start justify-between gap-4">
        <div>
          <Link
            href={course ? `/dashboard/courses/${course.id}` : "/dashboard/batches"}
            className="mb-2 inline-flex items-center text-sm text-muted-foreground hover:text-foreground"
          >
            <ArrowLeft className="mr-1 h-4 w-4" />
            {course ? course.title : "Batches"}
          </Link>
          <div className="flex items-center gap-2">
            <h1 className="text-2xl font-semibold tracking-tight">{batch.name}</h1>
            {canEdit && <RenameBatchDialog id={batch.id} currentName={batch.name} />}
            <Badge variant={published ? "default" : "secondary"}>{batch.status}</Badge>
          </div>
        </div>
        <div className="flex shrink-0 gap-2">
          {canEdit && (
            <Button variant="outline" size="sm" onClick={togglePublish} disabled={publishing}>
              {publishing && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {published ? "Unpublish" : "Publish"}
            </Button>
          )}
          {canDelete && (
            <Button variant="destructive" size="sm" onClick={remove} disabled={deleting}>
              <Trash2 className="mr-2 h-4 w-4" />
              Delete
            </Button>
          )}
        </div>
      </div>

      <Tabs defaultValue="modules">
        <TabsList>
          <TabsTrigger value="modules">Modules &amp; Tutors</TabsTrigger>
          <TabsTrigger value="students">Students</TabsTrigger>
          <TabsTrigger value="drive">Drive</TabsTrigger>
        </TabsList>
        <TabsContent value="modules">
          <ModulesTutorsTab batch={batch} canEdit={canEdit} />
        </TabsContent>
        <TabsContent value="students">
          <StudentsTab batchId={batch.id} canEdit={canEdit} />
        </TabsContent>
        <TabsContent value="drive">
          <DriveTab batchId={batch.id} canEdit={canEdit} />
        </TabsContent>
      </Tabs>
    </div>
  );
}

export default function BatchPage() {
  const params = useParams<{ id: string }>();
  const id = Number(params.id);
  if (!Number.isFinite(id) || id <= 0) notFound();

  return (
    <RequirePrivilege need="batch.view">
      <BatchDetail id={id} />
    </RequirePrivilege>
  );
}
