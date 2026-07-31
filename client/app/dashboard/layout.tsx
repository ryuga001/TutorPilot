import type { ReactNode } from "react";

import { AuthGuard } from "@/components/auth-guard";
import { AppShell } from "@/components/layout/AppShell";
import { DashboardProvider } from "@/lib/dashboard-context";

export default function DashboardLayout({
  children,
}: Readonly<{ children: ReactNode }>) {
  return (
    <AuthGuard>
      <DashboardProvider>
        <AppShell>{children}</AppShell>
      </DashboardProvider>
    </AuthGuard>
  );
}
