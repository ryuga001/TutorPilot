import { GraduationCap } from "lucide-react";

import { cn } from "@/lib/utils";

export interface LogoProps {
  /** Hide the wordmark — used in tight spaces like a collapsed sidebar. */
  iconOnly?: boolean;
  /** All-white wordmark for placement on a colored brand panel, where the
   *  default indigo "Pilot" accent would disappear against the gradient. */
  light?: boolean;
  className?: string;
}

/**
 * The one brand mark used everywhere (navbar, auth panel, collapsed
 * sidebar) so it never drifts into an inconsistent one-off per screen.
 */
export function Logo({ iconOnly, light, className }: LogoProps) {
  return (
    <div className={cn("flex items-center gap-2", className)}>
      <div
        className={cn(
          "flex h-8 w-8 shrink-0 items-center justify-center rounded-lg shadow-soft",
          light
            ? "bg-white/15 text-white"
            : "bg-gradient-to-br from-primary to-brand-secondary text-primary-foreground",
        )}
      >
        <GraduationCap className="h-4 w-4" strokeWidth={2.25} />
      </div>
      {!iconOnly && (
        <span className={cn("font-heading text-lg font-bold tracking-tight", light && "text-white")}>
          Tutor<span className={cn(!light && "text-primary")}>Pilot</span>
        </span>
      )}
    </div>
  );
}

export default Logo;
