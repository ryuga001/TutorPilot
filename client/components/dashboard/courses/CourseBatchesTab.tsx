"use client";

import { Loader2 } from "lucide-react";

import { BatchCard } from "@/components/dashboard/batches/BatchCard";
import { CreateBatchDialog } from "@/components/dashboard/batches/CreateBatchDialog";
import { useListBatchesQuery } from "@/lib/api/batchesApi";
import { useCan } from "@/lib/hooks/useCan";

export function CourseBatchesTab({ courseId }: { courseId: number }) {
  const can = useCan();
  const { data, isLoading } = useListBatchesQuery({ course_id: courseId, page_size: 50 });
  const items = data?.items ?? [];

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        {can("batch.create") && <CreateBatchDialog courseId={courseId} />}
      </div>

      {isLoading ? (
        <div className="flex h-32 items-center justify-center">
          <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
        </div>
      ) : items.length === 0 ? (
        <div className="flex h-32 items-center justify-center text-sm text-muted-foreground">
          No batches yet for this course.
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {items.map((b) => (
            <BatchCard key={b.id} batch={b} />
          ))}
        </div>
      )}
    </div>
  );
}
