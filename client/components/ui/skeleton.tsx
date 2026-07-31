import { cn } from "@/lib/utils";

/**
 * A shimmering placeholder block. Prefer this over a spinner for anything
 * that reserves layout space (table rows, cards) — it previews the shape of
 * the content that's coming, which reads as faster and feels less jarring
 * when the real content pops in.
 */
function Skeleton({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={cn("relative overflow-hidden rounded-md bg-muted", className)} {...props}>
      <div className="absolute inset-0 -translate-x-full animate-shimmer bg-gradient-to-r from-transparent via-foreground/10 to-transparent" />
    </div>
  );
}

export { Skeleton };
