"use client";

import Link from "next/link";
import { notFound, useParams, useRouter } from "next/navigation";
import { ArrowLeft, Loader2, Trash2 } from "lucide-react";
import { toast } from "sonner";
import type { FetchBaseQueryError } from "@reduxjs/toolkit/query";

import { RequirePrivilege } from "@/components/auth/RequirePrivilege";
import { CourseCurriculumTab } from "@/components/dashboard/courses/CourseCurriculumTab";
import { CourseDetailsTab } from "@/components/dashboard/courses/CourseDetailsTab";
import { CourseResourcesTab } from "@/components/dashboard/courses/CourseResourcesTab";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  useDeleteCourseMutation,
  useGetCourseQuery,
  useSetCoursePublishedMutation,
} from "@/lib/api/coursesApi";
import { apiErrorMessage } from "@/lib/api-error";
import { useCan } from "@/lib/hooks/useCan";

function CourseDetail({ id }: { id: number }) {
  const router = useRouter();
  const can = useCan();
  const canEdit = can("course.edit");
  const canDelete = can("course.delete");
  const { data: course, isLoading, error } = useGetCourseQuery(id);
  const [setPublished, { isLoading: publishing }] = useSetCoursePublishedMutation();
  const [deleteCourse, { isLoading: deleting }] = useDeleteCourseMutation();

  if (isLoading) {
    return (
      <div className="flex h-40 items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (error) {
    const status = (error as FetchBaseQueryError).status;
    if (status === 404) notFound();
    return (
      <p className="text-sm text-destructive">Could not load this course.</p>
    );
  }
  if (!course) notFound();

  const published = course.status === "published";

  async function togglePublish() {
    try {
      await setPublished({ id, published: !published }).unwrap();
      toast.success(published ? "Unpublished" : "Published");
    } catch (err) {
      toast.error(apiErrorMessage(err as FetchBaseQueryError));
    }
  }

  async function remove() {
    if (!window.confirm("Delete this course? This cannot be undone.")) return;
    try {
      await deleteCourse(id).unwrap();
      toast.success("Course deleted");
      router.push("/dashboard/courses");
    } catch (err) {
      toast.error(apiErrorMessage(err as FetchBaseQueryError));
    }
  }

  return (
    <div className="mx-auto w-full max-w-4xl">
      <div className="mb-6 flex items-start justify-between gap-4">
        <div>
          <Link
            href="/dashboard/courses"
            className="mb-2 inline-flex items-center text-sm text-muted-foreground hover:text-foreground"
          >
            <ArrowLeft className="mr-1 h-4 w-4" />
            Courses
          </Link>
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-semibold tracking-tight">
              {course.title}
            </h1>
            <Badge variant={published ? "default" : "secondary"}>
              {course.status}
            </Badge>
          </div>
        </div>
        <div className="flex shrink-0 gap-2">
          {canEdit && (
            <Button
              variant="outline"
              size="sm"
              onClick={togglePublish}
              disabled={publishing}
            >
              {publishing && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {published ? "Unpublish" : "Publish"}
            </Button>
          )}
          {canDelete && (
            <Button
              variant="destructive"
              size="sm"
              onClick={remove}
              disabled={deleting}
            >
              <Trash2 className="mr-2 h-4 w-4" />
              Delete
            </Button>
          )}
        </div>
      </div>

      <Tabs defaultValue="details">
        <TabsList>
          <TabsTrigger value="details">Details</TabsTrigger>
          <TabsTrigger value="curriculum">Curriculum</TabsTrigger>
          <TabsTrigger value="resources">Resources</TabsTrigger>
        </TabsList>
        <TabsContent value="details">
          <CourseDetailsTab course={course} canEdit={canEdit} />
        </TabsContent>
        <TabsContent value="curriculum">
          <CourseCurriculumTab course={course} canEdit={canEdit} />
        </TabsContent>
        <TabsContent value="resources">
          <CourseResourcesTab courseId={course.id} canEdit={canEdit} />
        </TabsContent>
      </Tabs>
    </div>
  );
}

export default function CoursePage() {
  const params = useParams<{ id: string }>();
  const id = Number(params.id);
  if (!Number.isFinite(id) || id <= 0) notFound();

  return (
    <RequirePrivilege need="course.view">
      <CourseDetail id={id} />
    </RequirePrivilege>
  );
}
