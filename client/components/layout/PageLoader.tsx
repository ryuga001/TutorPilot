import { Loader2 } from "lucide-react";

import { cn } from "@/lib/utils";

export interface PageLoaderProps {
  label?: string;
  className?: string;
}

/**
 * The one loading indicator reused everywhere a route segment, a lazy-loaded
 * component (next/dynamic), or a data fetch needs a placeholder — so a
 * "loading" moment always looks the same instead of every screen inventing
 * its own spinner.
 */
export function PageLoader({ label, className }: PageLoaderProps) {
  return (
    <div
      role="status"
      aria-live="polite"
      className={cn(
        "flex min-h-[16rem] w-full flex-col items-center justify-center gap-3 text-muted-foreground",
        className,
      )}
    >
      <Loader2 className="h-6 w-6 animate-spin text-primary" />
      {label && <p className="text-sm">{label}</p>}
    </div>
  );
}

export default PageLoader;
