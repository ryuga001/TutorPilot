"use client";

import { useEffect, useMemo, useState, type ReactNode } from "react";
import { toast } from "sonner";
import { Loader2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { apiErrorMessage } from "@/lib/api-error";
import { useListBatchesQuery } from "@/lib/api/batchesApi";
import { useListTutorsQuery } from "@/lib/api/tutorsApi";
import {
  useCreateLectureMutation,
  useUpdateLectureMutation,
} from "@/lib/api/lecturesApi";
import type { Lecture } from "@/lib/types";

/** datetime-local wants "YYYY-MM-DDTHH:mm" in the viewer's own timezone, while the
 *  API speaks RFC 3339 in UTC. */
function toLocalInputValue(iso: string): string {
  const d = new Date(iso);
  const offsetMs = d.getTimezoneOffset() * 60_000;
  return new Date(d.getTime() - offsetMs).toISOString().slice(0, 16);
}

function defaultStart(): string {
  const d = new Date();
  // Next whole hour, which is nearly always what someone scheduling a class wants.
  d.setMinutes(0, 0, 0);
  d.setHours(d.getHours() + 1);
  return toLocalInputValue(d.toISOString());
}

const NONE = "none";

interface Props {
  /** Omit to create; pass a lecture to edit it. */
  lecture?: Lecture;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  trigger?: ReactNode;
  onSaved?: (lecture: Lecture) => void;
}

export function LectureFormDialog({ lecture, open, onOpenChange, onSaved }: Props) {
  const isEdit = Boolean(lecture);

  const [createLecture, { isLoading: isCreating }] = useCreateLectureMutation();
  const [updateLecture, { isLoading: isUpdating }] = useUpdateLectureMutation();
  const isSaving = isCreating || isUpdating;

  // Only loaded while the dialog is open: a lecture form is not a reason to fetch
  // every batch and tutor on page load.
  const { data: batches } = useListBatchesQuery({ page: 1, page_size: 100 }, { skip: !open });
  const { data: tutors } = useListTutorsQuery({ page: 1, page_size: 100 }, { skip: !open });

  const [batchId, setBatchId] = useState<string>("");
  const [tutorId, setTutorId] = useState<string>(NONE);
  const [moduleId, setModuleId] = useState<string>(NONE);
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [startTime, setStartTime] = useState(defaultStart);
  const [endTime, setEndTime] = useState("");
  const [recordingEnabled, setRecordingEnabled] = useState(true);

  // Reset whenever the dialog opens, so a cancelled edit does not bleed into the next.
  useEffect(() => {
    if (!open) return;
    setBatchId(lecture ? String(lecture.batch_id) : "");
    setTutorId(lecture?.tutor_id ? String(lecture.tutor_id) : NONE);
    setModuleId(lecture?.module_id ? String(lecture.module_id) : NONE);
    setTitle(lecture?.title ?? "");
    setDescription(lecture?.description ?? "");
    setStartTime(lecture ? toLocalInputValue(lecture.start_time) : defaultStart());
    setEndTime(lecture?.end_time ? toLocalInputValue(lecture.end_time) : "");
    setRecordingEnabled(lecture?.recording_enabled ?? true);
  }, [open, lecture]);

  // Modules belong to the selected batch's course, so the options follow the batch.
  const selectedBatch = useMemo(
    () => batches?.items.find((b) => String(b.id) === batchId),
    [batches?.items, batchId],
  );
  const modules = selectedBatch?.modules ?? [];

  const canSubmit = title.trim().length > 0 && (isEdit || batchId !== "") && startTime !== "";

  async function onSubmit() {
    if (!canSubmit) return;

    const body = {
      title: title.trim(),
      description: description.trim(),
      recording_enabled: recordingEnabled,
      start_time: new Date(startTime).toISOString(),
      end_time: endTime ? new Date(endTime).toISOString() : undefined,
      tutor_id: tutorId === NONE ? undefined : Number(tutorId),
      module_id: moduleId === NONE ? undefined : Number(moduleId),
    };

    try {
      const saved = lecture
        ? await updateLecture({ id: lecture.id, body }).unwrap()
        : await createLecture({ ...body, batch_id: Number(batchId) }).unwrap();
      toast.success(isEdit ? "Lecture updated" : "Lecture scheduled");
      onOpenChange(false);
      onSaved?.(saved);
    } catch (err) {
      toast.error(apiErrorMessage(err as never));
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit lecture" : "Schedule a lecture"}</DialogTitle>
          <DialogDescription>
            {isEdit
              ? "Change the details of this lecture. The batch cannot be moved once it is created."
              : "Pick the batch it belongs to and when it runs. The room opens when you start it."}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          {!isEdit && (
            <div className="space-y-2">
              <Label htmlFor="lecture-batch">Batch</Label>
              <Select value={batchId} onValueChange={setBatchId}>
                <SelectTrigger id="lecture-batch">
                  <SelectValue placeholder="Select a batch" />
                </SelectTrigger>
                <SelectContent>
                  {batches?.items.map((b) => (
                    <SelectItem key={b.id} value={String(b.id)}>
                      {b.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}

          <div className="space-y-2">
            <Label htmlFor="lecture-title">Title</Label>
            <Input
              id="lecture-title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Introduction to algebra"
              maxLength={255}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="lecture-description">Description</Label>
            <Textarea
              id="lecture-description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="What this session covers"
              rows={3}
            />
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="lecture-start">Starts</Label>
              <Input
                id="lecture-start"
                type="datetime-local"
                value={startTime}
                onChange={(e) => setStartTime(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="lecture-end">Ends (optional)</Label>
              <Input
                id="lecture-end"
                type="datetime-local"
                value={endTime}
                min={startTime}
                onChange={(e) => setEndTime(e.target.value)}
              />
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="lecture-tutor">Tutor</Label>
              <Select value={tutorId} onValueChange={setTutorId}>
                <SelectTrigger id="lecture-tutor">
                  <SelectValue placeholder="Unassigned" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={NONE}>Unassigned</SelectItem>
                  {tutors?.items.map((t) => (
                    <SelectItem key={t.id} value={String(t.id)}>
                      {t.first_name} {t.last_name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="lecture-module">Module</Label>
              <Select
                value={moduleId}
                onValueChange={setModuleId}
                disabled={!isEdit && modules.length === 0}
              >
                <SelectTrigger id="lecture-module">
                  <SelectValue placeholder="None" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={NONE}>None</SelectItem>
                  {modules.map((m) => (
                    <SelectItem key={m.course_module_id} value={String(m.course_module_id)}>
                      {m.module_title}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <label className="flex items-start gap-3 rounded-md border bg-muted/30 p-3 text-sm">
            <Checkbox
              checked={recordingEnabled}
              onCheckedChange={(v) => setRecordingEnabled(v === true)}
              className="mt-0.5"
            />
            <span>
              <span className="font-medium">Record this lecture</span>
              <span className="block text-muted-foreground">
                The finished recording is filed into the batch drive, under Lecture
                Recordings.
              </span>
            </span>
          </label>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={onSubmit} disabled={!canSubmit || isSaving}>
            {isSaving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            {isEdit ? "Save changes" : "Schedule lecture"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
