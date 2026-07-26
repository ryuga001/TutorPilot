"use client";

import { useState } from "react";
import { CalendarRange, Loader2, UserX } from "lucide-react";
import { toast } from "sonner";
import type { FetchBaseQueryError } from "@reduxjs/toolkit/query";

import { AssignTutorDialog } from "@/components/dashboard/batches/AssignTutorDialog";
import { Button } from "@/components/ui/button";
import { useUnassignTutorMutation } from "@/lib/api/batchesApi";
import { apiErrorMessage } from "@/lib/api-error";
import type { Batch } from "@/lib/types";

export function ModulesTutorsTab({ batch, canEdit }: { batch: Batch; canEdit: boolean }) {
  const modules = batch.modules ?? [];
  const [assigningModuleId, setAssigningModuleId] = useState<number | null>(null);
  const [unassignTutor, { isLoading: unassigning }] = useUnassignTutorMutation();

  const assigning = modules.find((m) => m.course_module_id === assigningModuleId);

  async function handleUnassign(moduleId: number) {
    if (!window.confirm("Unassign this module's tutor?")) return;
    try {
      await unassignTutor({ batchId: batch.id, moduleId }).unwrap();
      toast.success("Tutor unassigned");
    } catch (err) {
      toast.error(apiErrorMessage(err as FetchBaseQueryError));
    }
  }

  if (modules.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        This course has no modules yet — add some in the course&apos;s Curriculum tab first.
      </p>
    );
  }

  return (
    <div className="space-y-3">
      {modules.map((m) => (
        <div
          key={m.course_module_id}
          className="flex flex-col gap-3 border p-4 sm:flex-row sm:items-center sm:justify-between"
        >
          <div>
            <p className="font-medium">{m.module_title}</p>
            {m.tutor ? (
              <div className="mt-1 space-y-0.5 text-sm text-muted-foreground">
                <p>
                  {m.tutor.first_name} {m.tutor.last_name} · {m.tutor.email}
                </p>
                {m.start_date && m.expected_end_date && (
                  <p className="flex items-center gap-1.5">
                    <CalendarRange className="h-3.5 w-3.5" />
                    {m.start_date} → {m.expected_end_date}
                  </p>
                )}
              </div>
            ) : (
              <p className="mt-1 text-sm text-muted-foreground">Unassigned</p>
            )}
          </div>
          {canEdit && (
            <div className="flex shrink-0 gap-2">
              <Button
                variant="secondary"
                size="sm"
                onClick={() => setAssigningModuleId(m.course_module_id)}
              >
                {m.tutor ? "Reassign" : "Assign tutor"}
              </Button>
              {m.tutor && (
                <Button
                  variant="destructive"
                  size="icon"
                  className="h-9 w-9"
                  onClick={() => handleUnassign(m.course_module_id)}
                  disabled={unassigning}
                  aria-label="Unassign tutor"
                >
                  {unassigning ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <UserX className="h-4 w-4" />
                  )}
                </Button>
              )}
            </div>
          )}
        </div>
      ))}

      {assigning && (
        <AssignTutorDialog
          batchId={batch.id}
          moduleAssignment={assigning}
          open={assigningModuleId !== null}
          onOpenChange={(open) => !open && setAssigningModuleId(null)}
        />
      )}
    </div>
  );
}
