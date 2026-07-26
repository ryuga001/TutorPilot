"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { Loader2 } from "lucide-react";

import { RequirePrivilege } from "@/components/auth/RequirePrivilege";
import PageTheme from "@/components/pagetheme/PageTheme";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useCreateLectureMutation } from "@/lib/api/lecturesApi";

function NewLecturePageContent() {
  const router = useRouter();
  const [createLecture, { isLoading }] = useCreateLectureMutation();
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [startTime, setStartTime] = useState(new Date().toISOString().slice(0, 16));
  const [recordingEnabled, setRecordingEnabled] = useState(true);

  const canSubmit = useMemo(() => title.trim().length > 0, [title]);

  const handleSubmit = async () => {
    if (!canSubmit) return;
    await createLecture({
      batchId: 1,
      title,
      description,
      recordingEnabled,
      startTime: new Date(startTime).toISOString(),
    });
    router.push("/dashboard/lectures");
  };

  return (
    <PageTheme title="Create lecture" subtitle="Create a lecture room and prepare it for live teaching.">
      <div className="max-w-2xl space-y-4 rounded-lg border bg-card p-6 shadow-sm">
        <div>
          <label className="mb-1 block text-sm font-medium">Title</label>
          <Input value={title} onChange={(e) => setTitle(e.target.value)} placeholder="Lecture title" />
        </div>
        <div>
          <label className="mb-1 block text-sm font-medium">Description</label>
          <Input value={description} onChange={(e) => setDescription(e.target.value)} placeholder="Short description" />
        </div>
        <div>
          <label className="mb-1 block text-sm font-medium">Start time</label>
          <Input type="datetime-local" value={startTime} onChange={(e) => setStartTime(e.target.value)} />
        </div>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={recordingEnabled} onChange={(e) => setRecordingEnabled(e.target.checked)} />
          Enable recording
        </label>
        <div className="flex gap-2">
          <Button onClick={handleSubmit} disabled={!canSubmit || isLoading}>
            {isLoading ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
            Create lecture
          </Button>
          <Button variant="outline" onClick={() => router.push("/dashboard/lectures")}>Cancel</Button>
        </div>
      </div>
    </PageTheme>
  );
}

export default function NewLecturePage() {
  return (
    <RequirePrivilege need="lecture.create">
      <NewLecturePageContent />
    </RequirePrivilege>
  );
}
