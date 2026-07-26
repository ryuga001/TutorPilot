"use client";

import { type ChangeEvent } from "react";
import { Copy, Loader2, MoreVertical, Trash2, Upload } from "lucide-react";
import { toast } from "sonner";
import type { FetchBaseQueryError } from "@reduxjs/toolkit/query";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  useDeleteResourceMutation,
  useListResourcesQuery,
  useUploadResourceMutation,
} from "@/lib/api/coursesApi";
import { apiErrorMessage } from "@/lib/api-error";
import { formatBytes, getFileIcon } from "@/lib/file-icons";

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
      <div className="flex justify-end">
        {canEdit && (
          <>
            <input id="res" type="file" className="hidden" onChange={onFile} />
            <Button asChild size="sm">
              <label htmlFor="res" className="cursor-pointer">
                {uploading ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <Upload className="mr-2 h-4 w-4" />
                )}
                Upload file
              </label>
            </Button>
          </>
        )}
      </div>

      <div className="min-h-[12rem] border">
        {isLoading ? (
          <div className="flex h-48 items-center justify-center">
            <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          </div>
        ) : items.length === 0 ? (
          <div className="flex h-48 flex-col items-center justify-center gap-2 text-sm text-muted-foreground">
            <Upload className="h-8 w-8" />
            No resources yet.
          </div>
        ) : (
          <div className="grid grid-cols-2 gap-1 p-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5">
            {items.map((r) => {
              const Icon = getFileIcon(r.content_type);
              return (
                <div
                  key={r.id}
                  role="button"
                  tabIndex={0}
                  onClick={() => window.open(r.url, "_blank", "noopener,noreferrer")}
                  onKeyDown={(e) =>
                    e.key === "Enter" &&
                    window.open(r.url, "_blank", "noopener,noreferrer")
                  }
                  className="group relative flex cursor-pointer flex-col items-center gap-1.5 p-3 text-center hover:bg-accent"
                >
                  <div
                    className="absolute right-1 top-1 opacity-0 transition-opacity group-hover:opacity-100 group-focus-within:opacity-100"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-7 w-7"
                          aria-label="Resource actions"
                        >
                          <MoreVertical className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem onClick={() => copy(r.url)}>
                          <Copy className="mr-2 h-4 w-4" />
                          Copy link
                        </DropdownMenuItem>
                        {canEdit && (
                          <DropdownMenuItem
                            onClick={() => del(r.id)}
                            className="text-destructive focus:text-destructive"
                          >
                            <Trash2 className="mr-2 h-4 w-4" />
                            Delete
                          </DropdownMenuItem>
                        )}
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                  <Icon className="h-10 w-10 text-muted-foreground" />
                  <span className="w-full truncate text-xs font-medium" title={r.name}>
                    {r.name}
                  </span>
                  <span className="text-[11px] text-muted-foreground">
                    {formatBytes(r.size_bytes)}
                  </span>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
