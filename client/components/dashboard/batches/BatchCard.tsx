import Link from "next/link";
import { GraduationCap, Users2 } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { Batch } from "@/lib/types";

export function BatchCard({ batch }: { batch: Batch }) {
  return (
    <Link href={`/dashboard/batches/${batch.id}`} className="group block">
      <Card className="flex h-full flex-col transition-colors group-hover:border-primary">
        <CardHeader className="p-4 pb-2">
          <div className="flex items-start justify-between gap-2">
            <CardTitle className="text-base leading-tight">{batch.name}</CardTitle>
            <Badge variant={batch.status === "published" ? "default" : "secondary"}>
              {batch.status}
            </Badge>
          </div>
        </CardHeader>
        <CardContent className="flex items-center gap-4 p-4 pt-0 text-sm text-muted-foreground">
          <span className="flex items-center gap-1.5">
            <Users2 className="h-4 w-4" />
            {batch.tutor_count} tutor{batch.tutor_count === 1 ? "" : "s"}
          </span>
          <span className="flex items-center gap-1.5">
            <GraduationCap className="h-4 w-4" />
            {batch.student_count} student{batch.student_count === 1 ? "" : "s"}
          </span>
        </CardContent>
      </Card>
    </Link>
  );
}
