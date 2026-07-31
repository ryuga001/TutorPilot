import type { LucideIcon } from "lucide-react";

import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

export interface StatCardProps {
  label: string;
  value: React.ReactNode;
  icon: LucideIcon;
  hint?: string;
  /** Rotates the icon chip through the two brand hues so a row of tiles
   *  doesn't read as one flat block of color. */
  tone?: "primary" | "secondary" | "success";
  isLoading?: boolean;
  className?: string;
}

const TONE_CLASSES: Record<NonNullable<StatCardProps["tone"]>, string> = {
  primary: "bg-primary/10 text-primary",
  secondary: "bg-brand-secondary/15 text-brand-secondary",
  success: "bg-success/10 text-success",
};

export function StatCard({
  label,
  value,
  icon: Icon,
  hint,
  tone = "primary",
  isLoading,
  className,
}: StatCardProps) {
  return (
    <Card
      className={cn(
        "group flex items-center gap-4 p-5 shadow-soft transition-all duration-200 hover:-translate-y-0.5 hover:shadow-elevated",
        className,
      )}
    >
      <div
        className={cn(
          "flex h-11 w-11 shrink-0 items-center justify-center rounded-xl transition-transform duration-200 group-hover:scale-105",
          TONE_CLASSES[tone],
        )}
        aria-hidden="true"
      >
        <Icon className="h-5 w-5" />
      </div>
      <div className="min-w-0 flex-1 space-y-1">
        <p className="text-sm text-muted-foreground">{label}</p>
        {isLoading ? (
          <Skeleton className="h-7 w-16" />
        ) : (
          <p className="font-heading text-2xl font-bold tracking-tight">{value}</p>
        )}
        {hint && <p className="truncate text-xs text-muted-foreground">{hint}</p>}
      </div>
    </Card>
  );
}

export default StatCard;
