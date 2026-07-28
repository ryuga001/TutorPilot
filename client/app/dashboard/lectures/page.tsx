"use client";

import { useEffect, useMemo, useState } from "react";
import { CalendarPlus, Loader2, Search, Video } from "lucide-react";
import { toast } from "sonner";

import { RequirePrivilege } from "@/components/auth/RequirePrivilege";
import PageTheme from "@/components/pagetheme/PageTheme";
import { LectureCard } from "@/components/dashboard/lectures/LectureCard";
import { LectureFormDialog } from "@/components/dashboard/lectures/LectureFormDialog";
import { isRecordingPending } from "@/components/dashboard/lectures/LectureStatus";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { apiErrorMessage } from "@/lib/api-error";
import {
  useCancelLectureMutation,
  useDeleteLectureMutation,
  useEndLectureMutation,
  useListLecturesQuery,
  useStartLectureMutation,
} from "@/lib/api/lecturesApi";
import { useCan } from "@/lib/hooks/useCan";
import type { Lecture } from "@/lib/types";

const PAGE_SIZE = 12;

const FILTERS: { value: string; label: string }[] = [
  { value: "", label: "All" },
  { value: "live", label: "Live" },
  { value: "scheduled", label: "Scheduled" },
  { value: "ended", label: "Ended" },
  { value: "cancelled", label: "Cancelled" },
];

function LecturesPageContent() {
  const can = useCan();

  const [status, setStatus] = useState("");
  const [searchInput, setSearchInput] = useState("");
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);

  const [editing, setEditing] = useState<Lecture | null>(null);
  const [formOpen, setFormOpen] = useState(false);
  const [busyId, setBusyId] = useState<number | null>(null);

  // Debounced, so typing does not fire a request per keystroke.
  useEffect(() => {
    const t = setTimeout(() => {
      setSearch(searchInput.trim());
      setPage(1);
    }, 300);
    return () => clearTimeout(t);
  }, [searchInput]);

  const { data, isLoading, isFetching } = useListLecturesQuery(
    { page, page_size: PAGE_SIZE, status, search },
    {
      // A live lecture's state and a recording being processed both change without
      // the user doing anything, so the list refreshes itself while either is true.
      pollingInterval: 15_000,
      skipPollingIfUnfocused: true,
    },
  );

  const [startLecture] = useStartLectureMutation();
  const [endLecture] = useEndLectureMutation();
  const [cancelLecture] = useCancelLectureMutation();
  const [deleteLecture] = useDeleteLectureMutation();

  const items = useMemo(() => data?.items ?? [], [data?.items]);
  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  const liveCount = items.filter((l) => l.status === "live").length;
  const pendingRecordings = items.filter((l) => isRecordingPending(l.recording_status)).length;

  async function run(lecture: Lecture, action: () => Promise<unknown>, success: string) {
    setBusyId(lecture.id);
    try {
      await action();
      toast.success(success);
    } catch (err) {
      toast.error(apiErrorMessage(err as never));
    } finally {
      setBusyId(null);
    }
  }

  const handleStart = (l: Lecture) =>
    run(l, () => startLecture(l.id).unwrap(), "Lecture started — the room is open");
  const handleEnd = (l: Lecture) =>
    run(l, () => endLecture(l.id).unwrap(), "Lecture ended");
  const handleCancel = (l: Lecture) =>
    run(l, () => cancelLecture(l.id).unwrap(), "Lecture cancelled");
  const handleDelete = (l: Lecture) =>
    run(l, () => deleteLecture(l.id).unwrap(), "Lecture deleted");

  function openCreate() {
    setEditing(null);
    setFormOpen(true);
  }

  function openEdit(lecture: Lecture) {
    setEditing(lecture);
    setFormOpen(true);
  }

  return (
    <PageTheme
      title="Lectures"
      subtitle="Schedule sessions, run them live, and find their recordings in the batch drive."
    >
      <div className="mb-5 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
          <span>
            {total} lecture{total === 1 ? "" : "s"}
          </span>
          {liveCount > 0 && (
            <span className="inline-flex items-center gap-1.5 rounded-full border border-red-500/30 bg-red-500/10 px-2 py-0.5 text-xs font-medium text-red-600 dark:text-red-400">
              <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-current" />
              {liveCount} live now
            </span>
          )}
          {pendingRecordings > 0 && (
            <span className="text-xs">{pendingRecordings} recording(s) processing</span>
          )}
        </div>

        {can("lecture.create") && (
          <Button onClick={openCreate}>
            <CalendarPlus className="mr-2 h-4 w-4" /> Schedule lecture
          </Button>
        )}
      </div>

      <div className="mb-5 flex flex-col gap-3 sm:flex-row sm:items-center">
        <Tabs
          value={status}
          onValueChange={(v) => {
            setStatus(v);
            setPage(1);
          }}
          className="w-full sm:w-auto"
        >
          <TabsList>
            {FILTERS.map((f) => (
              <TabsTrigger key={f.value || "all"} value={f.value}>
                {f.label}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>

        <div className="relative sm:ml-auto sm:w-64">
          <Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            placeholder="Search by title"
            className="pl-8"
          />
        </div>
      </div>

      {isLoading ? (
        <div className="flex h-64 items-center justify-center">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : items.length === 0 ? (
        <div className="flex h-64 flex-col items-center justify-center gap-3 rounded-xl border border-dashed text-center">
          <div className="flex h-11 w-11 items-center justify-center rounded-full bg-muted">
            <Video className="h-5 w-5 text-muted-foreground" />
          </div>
          <div>
            <p className="font-medium">
              {search || status ? "No lectures match this view" : "No lectures yet"}
            </p>
            <p className="text-sm text-muted-foreground">
              {search || status
                ? "Try a different filter or search term."
                : "Schedule one to open a live room for a batch."}
            </p>
          </div>
          {!search && !status && can("lecture.create") && (
            <Button variant="outline" onClick={openCreate}>
              <CalendarPlus className="mr-2 h-4 w-4" /> Schedule lecture
            </Button>
          )}
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {items.map((lecture) => (
            <LectureCard
              key={lecture.id}
              lecture={lecture}
              can={can}
              busy={busyId === lecture.id}
              onStart={handleStart}
              onEnd={handleEnd}
              onCancel={handleCancel}
              onEdit={openEdit}
              onDelete={handleDelete}
            />
          ))}
        </div>
      )}

      {totalPages > 1 && (
        <div className="mt-6 flex items-center justify-between text-sm text-muted-foreground">
          <span>{isFetching ? "Refreshing…" : `Page ${page} of ${totalPages}`}</span>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={page <= 1}
              onClick={() => setPage((p) => p - 1)}
            >
              Previous
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={page >= totalPages}
              onClick={() => setPage((p) => p + 1)}
            >
              Next
            </Button>
          </div>
        </div>
      )}

      <LectureFormDialog
        lecture={editing ?? undefined}
        open={formOpen}
        onOpenChange={setFormOpen}
      />
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
