"use client";

import { type ChangeEvent } from "react";
import { Copy, Loader2, Trash2, Upload } from "lucide-react";
import { toast } from "sonner";
import type { FetchBaseQueryError } from "@reduxjs/toolkit/query";

import { Button } from "@/components/ui/button";
import {
  useDeleteResourceMutation,
  useListResourcesQuery,
  useUploadResourceMutation,
} from "@/lib/api/coursesApi";
import { apiErrorMessage } from "@/lib/api-error";

function toastErr(err: unknown) {
  toast.error(apiErrorMessage(err as FetchBaseQueryError));
}

export function CourseResourcesTab({
  courseId,
  canEdit,
}: {
  courseId: number;
  canEdit: boolean;
}) {
  const { data, isLoading } = useListResourcesQuery({ courseId });
  const [upload, { isLoading: uploading }] = useUploadResourceMutation();
  const [remove] = useDeleteResourceMutation();
  const items = data?.items ?? [];

  async function onFile(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    try {
      await upload({ courseId, file }).unwrap();
      toast.success("Uploaded");
    } catch (err) {
      toastErr(err);
    } finally {
      e.target.value = "";
    }
  }

  async function del(id: number) {
    try {
      await remove({ courseId, resourceId: id }).unwrap();
    } catch (err) {
      toastErr(err);
    }
  }

  function copy(url: string) {
    navigator.clipboard.writeText(url);
    toast.success("Link copied");
  }

  return (
    <div className="space-y-4">
      {canEdit && (
        <div>
          <input id="res" type="file" className="hidden" onChange={onFile} />
          <Button asChild>
            <label htmlFor="res" className="cursor-pointer">
              {uploading ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Upload className="mr-2 h-4 w-4" />
              )}
              Upload file
            </label>
          </Button>
        </div>
      )}

      {isLoading ? (
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      ) : items.length === 0 ? (
        <p className="text-sm text-muted-foreground">No resources yet.</p>
      ) : (
        <ul className="divide-y border">
          {items.map((r) => (
            <li
              key={r.id}
              className="flex items-center justify-between gap-3 px-4 py-2"
            >
              <a
                href={r.url}
                target="_blank"
                rel="noreferrer"
                className="truncate text-sm text-primary hover:underline"
              >
                {r.name}
              </a>
              <div className="flex shrink-0 items-center gap-1">
                <Button
                  variant="secondary"
                  size="icon"
                  className="h-8 w-8"
                  onClick={() => copy(r.url)}
                  aria-label="Copy link"
                >
                  <Copy className="h-4 w-4" />
                </Button>
                {canEdit && (
                  <Button
                    variant="destructive"
                    size="icon"
                    className="h-8 w-8"
                    onClick={() => del(r.id)}
                    aria-label="Delete resource"
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                )}
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
