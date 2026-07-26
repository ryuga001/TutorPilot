"use client";

import { useState } from "react";
import {
  ChevronDown,
  ChevronRight,
  Loader2,
  Plus,
  Save,
  Trash2,
} from "lucide-react";
import { toast } from "sonner";
import type { FetchBaseQueryError } from "@reduxjs/toolkit/query";

import { MarkdownEditor } from "@/components/markdown/MarkdownEditor";
import { MarkdownView } from "@/components/markdown/MarkdownView";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  useCreateLessonMutation,
  useCreateModuleMutation,
  useDeleteLessonMutation,
  useDeleteModuleMutation,
  useUpdateLessonMutation,
} from "@/lib/api/coursesApi";
import { apiErrorMessage } from "@/lib/api-error";
import type { Course, CourseLesson, CourseModule } from "@/lib/types";

function toastErr(err: unknown) {
  toast.error(apiErrorMessage(err as FetchBaseQueryError));
}

export function CourseCurriculumTab({
  course,
  canEdit,
}: {
  course: Course;
  canEdit: boolean;
}) {
  const modules = course.modules ?? [];
  const [newModule, setNewModule] = useState("");
  const [createModule, { isLoading }] = useCreateModuleMutation();

  async function addModule() {
    if (!newModule.trim()) return;
    try {
      await createModule({
        courseId: course.id,
        body: { title: newModule, position: modules.length },
      }).unwrap();
      setNewModule("");
    } catch (err) {
      toastErr(err);
    }
  }

  return (
    <div className="space-y-4">
      {modules.length === 0 && (
        <p className="text-sm text-muted-foreground">No modules yet.</p>
      )}
      {modules.map((m) => (
        <ModuleItem key={m.id} courseId={course.id} module={m} canEdit={canEdit} />
      ))}
      {canEdit && (
        <div className="flex gap-2">
          <Input
            value={newModule}
            onChange={(e) => setNewModule(e.target.value)}
            placeholder="New module title"
          />
          <Button onClick={addModule} disabled={isLoading || !newModule.trim()}>
            {isLoading ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Plus className="h-4 w-4" />
            )}
            Add module
          </Button>
        </div>
      )}
    </div>
  );
}

function ModuleItem({
  courseId,
  module,
  canEdit,
}: {
  courseId: number;
  module: CourseModule;
  canEdit: boolean;
}) {
  const [deleteModule, { isLoading: deleting }] = useDeleteModuleMutation();
  const [createLesson, { isLoading: creatingLesson }] = useCreateLessonMutation();
  const [newLesson, setNewLesson] = useState("");

  async function addLesson() {
    if (!newLesson.trim()) return;
    try {
      await createLesson({
        courseId,
        moduleId: module.id,
        body: { title: newLesson, content_md: "", position: module.lessons.length },
      }).unwrap();
      setNewLesson("");
    } catch (err) {
      toastErr(err);
    }
  }

  async function removeModule() {
    try {
      await deleteModule({ courseId, moduleId: module.id }).unwrap();
    } catch (err) {
      toastErr(err);
    }
  }

  return (
    <div className="border">
      <div className="flex items-center justify-between border-b bg-muted/40 px-4 py-2">
        <h3 className="font-medium">{module.title}</h3>
        {canEdit && (
          <Button
            variant="destructive"
            size="icon"
            className="h-8 w-8"
            onClick={removeModule}
            disabled={deleting}
            aria-label="Delete module"
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        )}
      </div>
      <div className="divide-y">
        {module.lessons.length === 0 && (
          <p className="px-4 py-3 text-sm text-muted-foreground">No lessons.</p>
        )}
        {module.lessons.map((l) => (
          <LessonItem
            key={l.id}
            courseId={courseId}
            moduleId={module.id}
            lesson={l}
            canEdit={canEdit}
          />
        ))}
      </div>
      {canEdit && (
        <div className="flex gap-2 p-3">
          <Input
            value={newLesson}
            onChange={(e) => setNewLesson(e.target.value)}
            placeholder="New lesson title"
          />
          <Button
            variant="outline"
            onClick={addLesson}
            disabled={creatingLesson || !newLesson.trim()}
          >
            {creatingLesson ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Plus className="h-4 w-4" />
            )}
            Add lesson
          </Button>
        </div>
      )}
    </div>
  );
}

function LessonItem({
  courseId,
  moduleId,
  lesson,
  canEdit,
}: {
  courseId: number;
  moduleId: number;
  lesson: CourseLesson;
  canEdit: boolean;
}) {
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState(false);
  const [title, setTitle] = useState(lesson.title);
  const [content, setContent] = useState(lesson.content_md);
  const [updateLesson, { isLoading: saving }] = useUpdateLessonMutation();
  const [deleteLesson, { isLoading: deleting }] = useDeleteLessonMutation();

  async function save() {
    try {
      await updateLesson({
        courseId,
        moduleId,
        lessonId: lesson.id,
        body: { title, content_md: content, position: lesson.position },
      }).unwrap();
      toast.success("Lesson saved");
      setEditing(false);
    } catch (err) {
      toastErr(err);
    }
  }

  async function remove() {
    try {
      await deleteLesson({ courseId, moduleId, lessonId: lesson.id }).unwrap();
    } catch (err) {
      toastErr(err);
    }
  }

  return (
    <div className="px-4 py-2">
      <div className="flex items-center justify-between">
        <button
          type="button"
          onClick={() => setOpen((o) => !o)}
          className="flex items-center gap-2 text-sm font-medium"
        >
          {open ? (
            <ChevronDown className="h-4 w-4" />
          ) : (
            <ChevronRight className="h-4 w-4" />
          )}
          {lesson.title}
        </button>
        {canEdit && (
          <div className="flex gap-1">
            <Button
              variant="secondary"
              size="sm"
              onClick={() => {
                setOpen(true);
                setEditing(true);
              }}
            >
              Edit
            </Button>
            <Button
              variant="destructive"
              size="icon"
              className="h-8 w-8"
              onClick={remove}
              disabled={deleting}
              aria-label="Delete lesson"
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
        )}
      </div>
      {open && (
        <div className="mt-3">
          {editing ? (
            <div className="space-y-3">
              <Input
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="Lesson title"
              />
              <MarkdownEditor value={content} onChange={setContent} height={300} />
              <div className="flex gap-2">
                <Button size="sm" onClick={save} disabled={saving}>
                  {saving ? (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  ) : (
                    <Save className="mr-2 h-4 w-4" />
                  )}
                  Save
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={() => {
                    setTitle(lesson.title);
                    setContent(lesson.content_md);
                    setEditing(false);
                  }}
                >
                  Cancel
                </Button>
              </div>
            </div>
          ) : (
            <MarkdownView content={lesson.content_md} />
          )}
        </div>
      )}
    </div>
  );
}
