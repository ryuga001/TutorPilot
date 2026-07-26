"use client";

import { useEffect, useState, type FormEvent } from "react";
import { Loader2 } from "lucide-react";
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
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useAssignTutorMutation } from "@/lib/api/batchesApi";
import { useListTutorsQuery } from "@/lib/api/tutorsApi";
import { apiErrorMessage } from "@/lib/api-error";
import type { ModuleAssignment } from "@/lib/types";

export function AssignTutorDialog({
  batchId,
  moduleAssignment,
  open,
  onOpenChange,
}: {
  batchId: number;
  moduleAssignment: ModuleAssignment;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { data: tutorOptions } = useListTutorsQuery(
    { page: 1, page_size: 100 },
    { skip: !open },
  );
  const [tutorId, setTutorId] = useState(
    moduleAssignment.tutor ? String(moduleAssignment.tutor.id) : "",
  );
  const [startDate, setStartDate] = useState(moduleAssignment.start_date ?? "");
  const [endDate, setEndDate] = useState(moduleAssignment.expected_end_date ?? "");
  const [assignTutor, { isLoading }] = useAssignTutorMutation();

  useEffect(() => {
    if (open) {
      setTutorId(moduleAssignment.tutor ? String(moduleAssignment.tutor.id) : "");
      setStartDate(moduleAssignment.start_date ?? "");
      setEndDate(moduleAssignment.expected_end_date ?? "");
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  async function submit(e: FormEvent) {
    e.preventDefault();
    try {
      await assignTutor({
        batchId,
        moduleId: moduleAssignment.course_module_id,
        body: { tutor_id: Number(tutorId), start_date: startDate, expected_end_date: endDate },
      }).unwrap();
      toast.success("Tutor assigned");
      onOpenChange(false);
    } catch (err) {
      toast.error(apiErrorMessage(err as FetchBaseQueryError));
    }
  }

  const canSubmit = Boolean(tutorId) && Boolean(startDate) && Boolean(endDate);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <form onSubmit={submit}>
          <DialogHeader>
            <DialogTitle>Assign tutor — {moduleAssignment.module_title}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="tutor">Tutor</Label>
              <Select value={tutorId} onValueChange={setTutorId}>
                <SelectTrigger id="tutor">
                  <SelectValue placeholder="Select a tutor" />
                </SelectTrigger>
                <SelectContent>
                  {(tutorOptions?.items ?? []).map((t) => (
                    <SelectItem key={t.id} value={String(t.id)}>
                      {t.first_name} {t.last_name} · {t.email}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="start">Start date</Label>
                <input
                  id="start"
                  type="date"
                  value={startDate}
                  onChange={(e) => setStartDate(e.target.value)}
                  className="flex h-10 w-full border border-input bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="end">Expected end date</Label>
                <input
                  id="end"
                  type="date"
                  value={endDate}
                  onChange={(e) => setEndDate(e.target.value)}
                  min={startDate || undefined}
                  className="flex h-10 w-full border border-input bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                  required
                />
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button type="submit" disabled={isLoading || !canSubmit}>
              {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Save assignment
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
