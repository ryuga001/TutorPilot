"use client";

import { useMemo, useState, type FormEvent } from "react";
import Link from "next/link";
import { CalendarDays, Loader2, PlayCircle, Square, Trash2, Video, Pencil } from "lucide-react";
import { toast } from "sonner";
import type { FetchBaseQueryError } from "@reduxjs/toolkit/query";

import { RequirePrivilege } from "@/components/auth/RequirePrivilege";
import PageTheme from "@/components/pagetheme/PageTheme";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { apiErrorMessage } from "@/lib/api-error";
import { useDeleteLectureMutation, useEndLectureMutation, useListLecturesQuery, useStartLectureMutation, useUpdateLectureMutation } from "@/lib/api/lecturesApi";
import { useCan } from "@/lib/hooks/useCan";

function LecturesPageContent() {
  const can = useCan();
  const [page, setPage] = useState(1);
  const { data, isLoading, isFetching } = useListLecturesQuery({ page, page_size: 10 });
  const [startLecture] = useStartLectureMutation();
  const [endLecture] = useEndLectureMutation();
  const [deleteLecture] = useDeleteLectureMutation();
  const [updateLecture] = useUpdateLectureMutation();
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editTitle, setEditTitle] = useState("");
  const [editDescription, setEditDescription] = useState("");
  const [editRecordingEnabled, setEditRecordingEnabled] = useState(true);

  const items = useMemo(() => data?.items ?? [], [data?.items]);

  const handleStart = async (id: number) => {
    try {
      await startLecture(id).unwrap();
      toast.success("Lecture started");
    } catch (err) {
      toast.error(apiErrorMessage(err as FetchBaseQueryError));
    }
  };

  const handleEnd = async (id: number) => {
    try {
      await endLecture(id).unwrap();
      toast.success("Lecture ended");
    } catch (err) {
      toast.error(apiErrorMessage(err as FetchBaseQueryError));
    }
  };

  const startEdit = (lecture: { title: string; description: string; recordingEnabled: boolean; id: number }) => {
    setEditingId(lecture.id);
    setEditTitle(lecture.title);
    setEditDescription(lecture.description ?? "");
    setEditRecordingEnabled(lecture.recordingEnabled);
  };

  const submitEdit = async (e: FormEvent) => {
    e.preventDefault();
    if (!editingId) return;
    try {
      await updateLecture({
        id: editingId,
        body: {
          title: editTitle,
          description: editDescription,
          recordingEnabled: editRecordingEnabled,
        },
      }).unwrap();
      toast.success("Lecture updated");
      setEditingId(null);
    } catch (err) {
      toast.error(apiErrorMessage(err as FetchBaseQueryError));
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await deleteLecture(id).unwrap();
      toast.success("Lecture deleted");
    } catch (err) {
      toast.error(apiErrorMessage(err as FetchBaseQueryError));
    }
  };

  return (
    <PageTheme title="Lectures" subtitle="Run live sessions, monitor recordings, and manage lecture rooms.">
      <div className="mb-4 flex items-center justify-between">
        <div className="text-sm text-muted-foreground">{data?.total ?? 0} lecture(s)</div>
        {can("lecture.create") && (
          <Button asChild variant="default">
            <Link href="/dashboard/lectures/new">Create lecture</Link>
          </Button>
        )}
      </div>

      {isLoading ? (
        <div className="flex h-40 items-center justify-center">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : items.length === 0 ? (
        <div className="flex h-40 items-center justify-center rounded-lg border border-dashed text-sm text-muted-foreground">
          No lectures yet.
        </div>
      ) : (
        <div className="grid gap-4 lg:grid-cols-2">
          {items.map((lecture) => (
            <div key={lecture.id} className="rounded-lg border bg-card p-4 shadow-sm">
              <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
                <div className="space-y-2">
                  <div className="flex items-center gap-2">
                    <Video className="h-4 w-4 text-primary" />
                    <h3 className="font-semibold">{lecture.title}</h3>
                    <span className="rounded-full bg-muted px-2 py-0.5 text-xs uppercase text-muted-foreground">
                      {lecture.status}
                    </span>
                  </div>
                  <p className="text-sm text-muted-foreground">{lecture.description || "No description provided."}</p>
                  <div className="flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
                    <span className="inline-flex items-center gap-1"><CalendarDays className="h-4 w-4" />{new Date(lecture.startTime).toLocaleString()}</span>
                    {lecture.roomName && <span className="font-mono text-xs">Room: {lecture.roomName}</span>}
                  </div>
                </div>

                <div className="flex flex-wrap gap-2">
                  {lecture.status !== "live" ? (
                    <Button size="sm" variant="outline" onClick={() => void handleStart(lecture.id)}>
                      <PlayCircle className="mr-2 h-4 w-4" /> Start
                    </Button>
                  ) : (
                    <Button size="sm" variant="outline" onClick={() => void handleEnd(lecture.id)}>
                      <Square className="mr-2 h-4 w-4" /> End
                    </Button>
                  )}
                  <Dialog open={editingId === lecture.id} onOpenChange={(open) => !open && setEditingId(null)}>
                    <DialogTrigger asChild>
                      <Button size="sm" variant="secondary" onClick={() => startEdit(lecture)}>
                        <Pencil className="mr-2 h-4 w-4" /> Edit
                      </Button>
                    </DialogTrigger>
                    <DialogContent>
                      <form onSubmit={submitEdit}>
                        <DialogHeader>
                          <DialogTitle>Edit lecture</DialogTitle>
                          <DialogDescription>Update the lecture title, description, and recording preference.</DialogDescription>
                        </DialogHeader>
                        <div className="space-y-4 py-4">
                          <div className="space-y-2">
                            <Label htmlFor="edit-title">Title</Label>
                            <Input id="edit-title" value={editTitle} onChange={(e) => setEditTitle(e.target.value)} required />
                          </div>
                          <div className="space-y-2">
                            <Label htmlFor="edit-description">Description</Label>
                            <Textarea id="edit-description" value={editDescription} onChange={(e) => setEditDescription(e.target.value)} />
                          </div>
                          <label className="flex items-center gap-2 text-sm">
                            <input type="checkbox" checked={editRecordingEnabled} onChange={(e) => setEditRecordingEnabled(e.target.checked)} />
                            Enable recording
                          </label>
                        </div>
                        <DialogFooter>
                          <Button type="submit">Save</Button>
                        </DialogFooter>
                      </form>
                    </DialogContent>
                  </Dialog>
                  <Button size="sm" variant="destructive" onClick={() => handleDelete(lecture.id)}>
                    <Trash2 className="mr-2 h-4 w-4" /> Delete
                  </Button>
                  <Button size="sm" variant="secondary" asChild>
                    <Link href={`/dashboard/lectures/${lecture.id}`}>Open</Link>
                  </Button>
                </div>
              </div>

              {lecture.recordingUrl && (
                <div className="mt-4 rounded-md border border-dashed bg-muted/40 p-3 text-sm">
                  <div className="font-medium">Recording</div>
                  <a href={lecture.recordingUrl} target="_blank" rel="noreferrer" className="text-primary hover:underline">
                    Open recording
                  </a>
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      <div className="mt-6 flex items-center justify-between text-sm text-muted-foreground">
        <span>{isFetching ? "Refreshing…" : ""}</span>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
            Previous
          </Button>
          <span>Page {page}</span>
          <Button variant="outline" size="sm" onClick={() => setPage((p) => p + 1)}>
            Next
          </Button>
        </div>
      </div>
    </PageTheme>
  );
}

export default function LecturesPage() {
  return (
    <RequirePrivilege need="lecture.view">
      <LecturesPageContent />
    </RequirePrivilege>
  );
}
