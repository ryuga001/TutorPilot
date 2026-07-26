import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import type { ReactNode } from "react";

import { Badge } from "@/components/ui/badge";

export interface DetailPageHeaderProps {
  backHref: string;
  backLabel: string;
  title: string;
  titleAdornment?: ReactNode;
  badge?: { label: string; variant?: "default" | "secondary" | "destructive" | "outline" };
  actions?: ReactNode;
}

export function DetailPageHeader({
  backHref,
  backLabel,
  title,
  titleAdornment,
  badge,
  actions,
}: DetailPageHeaderProps) {
  return (
    <div className="mb-6 flex items-start justify-between gap-4">
      <div>
        <Link
          href={backHref}
          className="mb-2 inline-flex items-center text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="mr-1 h-4 w-4" />
          {backLabel}
        </Link>
        <div className="flex items-center gap-2">
          <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
          {titleAdornment}
          {badge && <Badge variant={badge.variant ?? "secondary"}>{badge.label}</Badge>}
        </div>
      </div>
      {actions && <div className="flex shrink-0 gap-2">{actions}</div>}
    </div>
  );
}
