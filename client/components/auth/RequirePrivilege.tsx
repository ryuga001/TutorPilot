"use client";

import type { ReactNode } from "react";
import { Loader2 } from "lucide-react";

import { NotAuthorized } from "@/components/not-authorized";
import { useCan, usePrivilegesReady } from "@/lib/hooks/useCan";

export function RequirePrivilege({
  need,
  children,
}: {
  need: string;
  children: ReactNode;
}) {
  const ready = usePrivilegesReady();
  const can = useCan();

  if (!ready) {
    return (
      <div className="flex min-h-[40vh] items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }
  if (!can(need)) {
    return <NotAuthorized />;
  }
  return <>{children}</>;
}
