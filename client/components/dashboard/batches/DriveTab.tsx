"use client";

import { useState, type ChangeEvent } from "react";
import {
  ChevronRight,
  Folder,
  FolderPlus,
  Home,
  Loader2,
  MoreVertical,
  Pencil,
  Trash2,
  Upload,
} from "lucide-react";
import { toast } from "sonner";
import type { FetchBaseQueryError } from "@reduxjs/toolkit/query";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import {
  useCreateFolderMutation,
  useDeleteNodeMutation,
  useListDriveQuery,
  useRenameNodeMutation,
  useUploadDriveFileMutation,
} from "@/lib/api/batchesApi";
import { apiErrorMessage } from "@/lib/api-error";
import { formatBytes, getFileIcon } from "@/lib/file-icons";
import type { DriveNode } from "@/lib/types";
import { cn } from "@/lib/utils";

function nodeIcon(node: DriveNode) {
  return node.type === "folder" ? Folder : getFileIcon(node.content_type);
}

interface Crumb {
  id: number;
  name: string;
}

function toastErr(err: unknown) {
  toast.error(apiErrorMessage(err as FetchBaseQueryError));
}

export function DriveTab({ batchId, canEdit }: { batchId: number; canEdit: boolean }) {
  const [path, setPath] = useState<Crumb[]>([]);
  const parentId = path.length > 0 ? path[path.length - 1].id : undefined;

  const { data: nodes, isLoading } = useListDriveQuery({ batchId, parentId });
  const [createFolder, { isLoading: creatingFolder }] = useCreateFolderMutation();
  const [uploadFile, { isLoading: uploading }] = useUploadDriveFileMutation();
  const [renameNode] = useRenameNodeMutation();
  const [deleteNode] = useDeleteNodeMutation();

  const [newFolderOpen, setNewFolderOpen] = useState(false);
  const [newFolderName, setNewFolderName] = useState("");
  const [renaming, setRenaming] = useState<DriveNode | null>(null);
  const [renameValue, setRenameValue] = useState("");

  function openFolder(node: DriveNode) {
    setPath((p) => [...p, { id: node.id, name: node.name }]);
  }

  function goToCrumb(index: number) {
    // index -1 = root
    setPath((p) => p.slice(0, index + 1));
  }

  async function handleCreateFolder() {
    if (!newFolderName.trim()) return;
    try {
      await createFolder({ batchId, name: newFolderName.trim(), parentId }).unwrap();
      setNewFolderOpen(false);
      setNewFolderName("");
    } catch (err) {
      toastErr(err);
    }
  }

  async function onUpload(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    try {
      await uploadFile({ batchId, file, parentId }).unwrap();
      toast.success("File uploaded");
    } catch (err) {
      toastErr(err);
    } finally {
      e.target.value = "";
    }
  }

  async function handleRename() {
    if (!renaming || !renameValue.trim()) return;
    try {
      await renameNode({ batchId, nodeId: renaming.id, name: renameValue.trim() }).unwrap();
      setRenaming(null);
    } catch (err) {
      toastErr(err);
    }
  }

  async function handleDelete(node: DriveNode) {
    const label = node.type === "folder" ? "this folder and everything inside it" : node.name;
    if (!window.confirm(`Delete ${label}? This cannot be undone.`)) return;
    try {
      await deleteNode({ batchId, nodeId: node.id }).unwrap();
      toast.success("Deleted");
    } catch (err) {
      toastErr(err);
    }
  }

  function handleOpen(node: DriveNode) {
    if (node.type === "folder") {
      openFolder(node);
    } else if (node.url) {
      window.open(node.url, "_blank", "noopener,noreferrer");
    }
  }

  const items = nodes ?? [];

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <nav className="flex flex-wrap items-center gap-1 text-sm">
          <button
            type="button"
            onClick={() => goToCrumb(-1)}
            className={cn(
              "flex items-center gap-1 px-1.5 py-1 hover:bg-accent",
              path.length === 0 ? "font-medium text-foreground" : "text-muted-foreground",
            )}
          >
            <Home className="h-3.5 w-3.5" />
            Drive
          </button>
          {path.map((crumb, i) => (
            <span key={crumb.id} className="flex items-center gap-1">
              <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
              <button
                type="button"
                onClick={() => goToCrumb(i)}
                className={cn(
                  "px-1.5 py-1 hover:bg-accent",
                  i === path.length - 1 ? "font-medium text-foreground" : "text-muted-foreground",
                )}
              >
                {crumb.name}
              </button>
            </span>
          ))}
        </nav>

        {canEdit && (
          <div className="flex gap-2">
            <Button variant="secondary" size="sm" onClick={() => setNewFolderOpen(true)}>
              <FolderPlus className="mr-2 h-4 w-4" />
              New folder
            </Button>
            <input id="drive-upload" type="file" className="hidden" onChange={onUpload} />
            <Button asChild size="sm">
              <label htmlFor="drive-upload" className="cursor-pointer">
                {uploading ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <Upload className="mr-2 h-4 w-4" />
                )}
                Upload
              </label>
            </Button>
          </div>
        )}
      </div>

      <div className="min-h-[16rem] border">
        {isLoading ? (
          <div className="flex h-64 items-center justify-center">
            <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          </div>
        ) : items.length === 0 ? (
          <div className="flex h-64 flex-col items-center justify-center gap-2 text-sm text-muted-foreground">
            <Folder className="h-8 w-8" />
            This folder is empty.
          </div>
        ) : (
          <div className="grid grid-cols-2 gap-1 p-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5">
            {items.map((node) => {
              const Icon = nodeIcon(node);
              return (
                <div
                  key={node.id}
                  role="button"
                  tabIndex={0}
                  onClick={() => handleOpen(node)}
                  onKeyDown={(e) => e.key === "Enter" && handleOpen(node)}
                  className="group relative flex cursor-pointer flex-col items-center gap-1.5 p-3 text-center hover:bg-accent"
                >
                  {canEdit && (
                    <div
                      className="absolute right-1 top-1 opacity-0 transition-opacity group-hover:opacity-100 group-focus-within:opacity-100"
                      onClick={(e) => e.stopPropagation()}
                    >
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="icon" className="h-7 w-7" aria-label="Node actions">
                            <MoreVertical className="h-4 w-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem
                            onClick={() => {
                              setRenaming(node);
                              setRenameValue(node.name);
                            }}
                          >
                            <Pencil className="mr-2 h-4 w-4" />
                            Rename
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            onClick={() => handleDelete(node)}
                            className="text-destructive focus:text-destructive"
                          >
                            <Trash2 className="mr-2 h-4 w-4" />
                            Delete
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </div>
                  )}
                  <Icon
                    className={cn(
                      "h-10 w-10",
                      node.type === "folder" ? "text-primary" : "text-muted-foreground",
                    )}
                  />
                  <span className="w-full truncate text-xs font-medium" title={node.name}>
                    {node.name}
                  </span>
                  {node.type === "file" && (
                    <span className="text-[11px] text-muted-foreground">
                      {formatBytes(node.size_bytes)}
                    </span>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>

      <Dialog open={newFolderOpen} onOpenChange={setNewFolderOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New folder</DialogTitle>
          </DialogHeader>
          <div className="py-4">
            <Input
              autoFocus
              value={newFolderName}
              onChange={(e) => setNewFolderName(e.target.value)}
              placeholder="Folder name"
              onKeyDown={(e) => e.key === "Enter" && handleCreateFolder()}
            />
          </div>
          <DialogFooter>
            <Button onClick={handleCreateFolder} disabled={creatingFolder || !newFolderName.trim()}>
              {creatingFolder && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Create
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={renaming !== null} onOpenChange={(o) => !o && setRenaming(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Rename</DialogTitle>
          </DialogHeader>
          <div className="py-4">
            <Input
              autoFocus
              value={renameValue}
              onChange={(e) => setRenameValue(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleRename()}
            />
          </div>
          <DialogFooter>
            <Button onClick={handleRename} disabled={!renameValue.trim()}>
              Save
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
