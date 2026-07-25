import type { ReactNode } from "react";

import { AuthGuard } from "@/components/auth-guard";
import Navbar from "@/components/dashboard/navbar/Navbar";
import Sidebar from "@/components/dashboard/sidebard/Sidebar";
import { DashboardProvider } from "@/lib/dashboard-context";

export default function DashboardLayout({
  children,
}: Readonly<{ children: ReactNode }>) {
  return (
    <AuthGuard>
      <DashboardProvider>
        <div className="flex min-h-screen flex-col bg-background">
          <header>
            <Navbar />
          </header>
          <section className="flex flex-1">
            <aside>
              <Sidebar />
            </aside>
            <main className="flex-1 overflow-y-auto p-4 sm:p-6">{children}</main>
          </section>
        </div>
      </DashboardProvider>
    </AuthGuard>
  );
}
