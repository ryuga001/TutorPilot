import Link from "next/link";

import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { Course } from "@/lib/types";

export function CourseCard({ course }: { course: Course }) {
  return (
    <Link href={`/dashboard/courses/${course.id}`} className="group block">
      <Card className="flex h-full flex-col overflow-hidden transition-colors group-hover:border-primary">
        <div className="aspect-video w-full bg-muted">
          {course.thumbnail_url ? (
            <img
              src={course.thumbnail_url}
              alt={course.title}
              loading="lazy"
              decoding="async"
              className="h-full w-full object-cover"
            />
          ) : (
            <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
              No cover image
            </div>
          )}
        </div>
        <CardHeader className="p-4 pb-2">
          <div className="flex items-start justify-between gap-2">
            <CardTitle className="text-base leading-tight">
              {course.title}
            </CardTitle>
            <Badge variant={course.status === "published" ? "default" : "secondary"}>
              {course.status}
            </Badge>
          </div>
        </CardHeader>
        <CardContent className="p-4 pt-0">
          <p className="line-clamp-2 text-sm text-muted-foreground">
            {course.summary || "No description."}
          </p>
        </CardContent>
      </Card>
    </Link>
  );
}
