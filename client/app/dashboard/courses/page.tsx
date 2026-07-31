"use client";

import { useState } from "react";
import { BookOpen, ChevronLeft, ChevronRight, Loader2, Search } from "lucide-react";

import { RequirePrivilege } from "@/components/auth/RequirePrivilege";
import { CourseCard } from "@/components/dashboard/courses/CourseCard";
import { CreateCourseDialog } from "@/components/dashboard/courses/CreateCourseDialog";
import PageTheme from "@/components/pagetheme/PageTheme";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useListCoursesQuery } from "@/lib/api/coursesApi";
import { useCan } from "@/lib/hooks/useCan";

const PAGE_SIZE = 12;

function CoursesList() {
  const can = useCan();
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const { data, isLoading, isFetching } = useListCoursesQuery({
    page,
    page_size: PAGE_SIZE,
    search,
  });

  const total = data?.total ?? 0;
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const items = data?.items ?? [];

  return (
    <PageTheme
      icon={BookOpen}
      title="Courses"
      subtitle="Create and manage your organization's courses."
    >
      <div className="mb-4 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div className="relative w-full sm:max-w-xs">
          <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            value={search}
            onChange={(e) => {
              setSearch(e.target.value);
              setPage(1);
            }}
            placeholder="Search courses…"
            className="pl-8"
          />
        </div>
        {can("course.create") && <CreateCourseDialog />}
      </div>

      {isLoading ? (
        <div className="flex h-40 items-center justify-center">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : items.length === 0 ? (
        <div className="flex h-40 items-center justify-center text-sm text-muted-foreground">
          No courses yet.
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {items.map((c) => (
            <CourseCard key={c.id} course={c} />
          ))}
        </div>
      )}

      <div className="mt-6 flex items-center justify-between text-sm text-muted-foreground">
        <span>
          {total} course{total === 1 ? "" : "s"}
        </span>
        <div className="flex items-center gap-2">
          {isFetching && <Loader2 className="h-4 w-4 animate-spin" />}
          <span>
            Page {page} of {pageCount}
          </span>
          <Button
            variant="outline"
            size="icon"
            className="h-8 w-8"
            disabled={page <= 1}
            onClick={() => setPage((p) => p - 1)}
            aria-label="Previous page"
          >
            <ChevronLeft className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            size="icon"
            className="h-8 w-8"
            disabled={page >= pageCount}
            onClick={() => setPage((p) => p + 1)}
            aria-label="Next page"
          >
            <ChevronRight className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </PageTheme>
  );
}

export default function CoursesPage() {
  return (
    <RequirePrivilege need="course.view">
      <CoursesList />
    </RequirePrivilege>
  );
}
