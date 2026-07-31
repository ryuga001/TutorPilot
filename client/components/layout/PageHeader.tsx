import Link from "next/link";
import { ChevronRight, type LucideIcon } from "lucide-react";

import { cn } from "@/lib/utils";

export interface Breadcrumb {
  label: string;
  href?: string;
}

export interface PageHeaderProps {
  title?: string;
  subtitle?: string;
  /** Small tile shown to the left of the title — ties every page back to the
   *  brand hue instead of a bare heading. */
  icon?: LucideIcon;
  breadcrumbs?: Breadcrumb[];
  /** Buttons/toolbar rendered on the right, aligned with the title. */
  actions?: React.ReactNode;
  children?: React.ReactNode;
  className?: string;
}

/**
 * The one page-chrome primitive every dashboard route renders through, so
 * spacing, title treatment and breadcrumbs stay identical everywhere instead
 * of each page hand-rolling its own header block.
 */
export function PageHeader({
  title,
  subtitle,
  icon: Icon,
  breadcrumbs,
  actions,
  children,
  className,
}: PageHeaderProps) {
  return (
    <div className={cn("space-y-6 pb-2", className)}>
      <section className="space-y-3">
        {breadcrumbs && breadcrumbs.length > 0 && (
          <nav aria-label="Breadcrumb">
            <ol className="flex flex-wrap items-center gap-1.5 text-sm text-muted-foreground">
              {breadcrumbs.map((b, i) => {
                const isLast = i === breadcrumbs.length - 1;
                return (
                  <li key={`${b.label}-${i}`} className="flex items-center gap-1.5">
                    {b.href && !isLast ? (
                      <Link href={b.href} className="transition-colors hover:text-foreground">
                        {b.label}
                      </Link>
                    ) : (
                      <span className={cn(isLast && "font-medium text-foreground")}>
                        {b.label}
                      </span>
                    )}
                    {!isLast && <ChevronRight className="h-3.5 w-3.5" aria-hidden="true" />}
                  </li>
                );
              })}
            </ol>
          </nav>
        )}

        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-3">
            {Icon && (
              <div
                className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-primary to-primary/70 text-primary-foreground shadow-soft"
                aria-hidden="true"
              >
                <Icon className="h-5 w-5" />
              </div>
            )}
            <div className="space-y-1">
              {title && (
                <h1 className="font-heading text-2xl font-bold tracking-tight sm:text-3xl">
                  {title}
                </h1>
              )}
              {subtitle && (
                <p className="max-w-2xl text-sm text-muted-foreground">{subtitle}</p>
              )}
            </div>
          </div>
          {actions && (
            <div className="flex shrink-0 items-center gap-2">{actions}</div>
          )}
        </div>
      </section>

      {children && (
        <section className="animate-in fade-in slide-in-from-bottom-1 space-y-6 duration-300">
          {children}
        </section>
      )}
    </div>
  );
}

export default PageHeader;
