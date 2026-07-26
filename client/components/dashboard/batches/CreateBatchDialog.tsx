"use client";

import { useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { Loader2, Plus } from "lucide-react";
import { toast } from "sonner";
import type { FetchBaseQueryError } from "@reduxjs/toolkit/query";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
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
import { useCreateBatchMutation } from "@/lib/api/batchesApi";
import { useListCoursesQuery } from "@/lib/api/coursesApi";
import { apiErrorMessage } from "@/lib/api-error";

export function CreateBatchDialog({ courseId }: { courseId?: number }) {
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [selectedCourse, setSelectedCourse] = useState<string>(
    courseId ? String(courseId) : "",
  );
  const [createBatch, { isLoading }] = useCreateBatchMutation();
  const { data: courseOptions } = useListCoursesQuery(
    { page: 1, page_size: 100 },
    { skip: courseId !== undefined || !open },
  );
  const router = useRouter();

  async function submit(e: FormEvent) {
    e.preventDefault();
    const finalCourseId = courseId ?? Number(selectedCourse);
    if (!finalCourseId) return;
    try {
      const batch = await createBatch({ course_id: finalCourseId, name }).unwrap();
      toast.success("Batch created");
      setOpen(false);
      setName("");
      router.push(`/dashboard/batches/${batch.id}`);
    } catch (err) {
      toast.error(apiErrorMessage(err as FetchBaseQueryError));
    }
  }

  const canSubmit = name.trim().length > 0 && (courseId !== undefined || Boolean(selectedCourse));

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button size="sm">
          <Plus className="mr-2 h-4 w-4" />
          New batch
        </Button>
      </DialogTrigger>
      <DialogContent>
        <form onSubmit={submit}>
          <DialogHeader>
            <DialogTitle>Create batch</DialogTitle>
            <DialogDescription>
              A batch is one scheduled offering of a course — assign tutors to
              its modules and enroll students next.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            {courseId === undefined && (
              <div className="space-y-2">
                <Label htmlFor="course">Course</Label>
                <Select value={selectedCourse} onValueChange={setSelectedCourse}>
                  <SelectTrigger id="course">
                    <SelectValue placeholder="Select a course" />
                  </SelectTrigger>
                  <SelectContent>
                    {(courseOptions?.items ?? []).map((c) => (
                      <SelectItem key={c.id} value={String(c.id)}>
                        {c.title}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}
            <div className="space-y-2">
              <Label htmlFor="batch-name">Batch name</Label>
              <Input
                id="batch-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. Spring 2026 Cohort"
                required
              />
            </div>
          </div>
          <DialogFooter>
            <Button type="submit" disabled={isLoading || !canSubmit}>
              {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Create
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
