"use client";

import { useState, type ChangeEvent } from "react";
import { Loader2, Pencil, Upload, X } from "lucide-react";
import { toast } from "sonner";
import type { FetchBaseQueryError } from "@reduxjs/toolkit/query";

import { MarkdownEditor } from "@/components/markdown/MarkdownEditor";
import { MarkdownView } from "@/components/markdown/MarkdownView";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  useUpdateCourseMutation,
  useUploadThumbnailMutation,
} from "@/lib/api/coursesApi";
import { apiErrorMessage } from "@/lib/api-error";
import type { Course } from "@/lib/types";

export function CourseDetailsTab({
  course,
  canEdit,
}: {
  course: Course;
  canEdit: boolean;
}) {
  const [editing, setEditing] = useState(false);
  const [title, setTitle] = useState(course.title);
  const [summary, setSummary] = useState(course.summary);
  const [descriptionMD, setDescriptionMD] = useState(course.description_md);
  const [update, { isLoading }] = useUpdateCourseMutation();
  const [uploadThumb, { isLoading: uploadingThumb }] = useUploadThumbnailMutation();

  function reset() {
    setTitle(course.title);
    setSummary(course.summary);
    setDescriptionMD(course.description_md);
  }

  async function save() {
    try {
      await update({
        id: course.id,
        body: { title, summary, description_md: descriptionMD },
      }).unwrap();
      toast.success("Saved");
      setEditing(false);
    } catch (err) {
      toast.error(apiErrorMessage(err as FetchBaseQueryError));
    }
  }

  async function onThumb(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    try {
      await uploadThumb({ id: course.id, file }).unwrap();
      toast.success("Cover updated");
    } catch (err) {
      toast.error(apiErrorMessage(err as FetchBaseQueryError));
    } finally {
      e.target.value = "";
    }
  }

  if (!editing) {
    return (
      <div className="space-y-6">
        {canEdit && (
          <div className="flex justify-end">
            <Button variant="outline" size="sm" onClick={() => setEditing(true)}>
              <Pencil className="mr-2 h-4 w-4" />
              Edit details
            </Button>
          </div>
        )}
        {course.thumbnail_url && (
          <img
            src={course.thumbnail_url}
            alt=""
            loading="lazy"
            decoding="async"
            className="max-h-64 w-full border object-cover"
          />
        )}
        {course.summary && (
          <p className="text-sm text-muted-foreground">{course.summary}</p>
        )}
        <MarkdownView content={course.description_md} />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="c-title">Title</Label>
        <Input id="c-title" value={title} onChange={(e) => setTitle(e.target.value)} />
      </div>
      <div className="space-y-2">
        <Label htmlFor="c-summary">Summary</Label>
        <Textarea
          id="c-summary"
          value={summary}
          onChange={(e) => setSummary(e.target.value)}
        />
      </div>
      <div className="space-y-2">
        <Label>Cover image</Label>
        <div>
          <input
            id="thumb"
            type="file"
            accept="image/*"
            className="hidden"
            onChange={onThumb}
          />
          <Button asChild variant="outline" size="sm">
            <label htmlFor="thumb" className="cursor-pointer">
              {uploadingThumb ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Upload className="mr-2 h-4 w-4" />
              )}
              Upload cover
            </label>
          </Button>
        </div>
      </div>
      <div className="space-y-2">
        <Label>Description</Label>
        <MarkdownEditor value={descriptionMD} onChange={setDescriptionMD} />
      </div>
      <div className="flex gap-2">
        <Button onClick={save} disabled={isLoading}>
          {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          Save
        </Button>
        <Button
          variant="ghost"
          onClick={() => {
            reset();
            setEditing(false);
          }}
        >
          <X className="mr-2 h-4 w-4" />
          Cancel
        </Button>
      </div>
    </div>
  );
}
