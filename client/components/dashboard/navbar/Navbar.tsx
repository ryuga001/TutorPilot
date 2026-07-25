"use client";

import { ThemeToggle } from "@/components/theme-toggle";
import { useDashboard } from "@/lib/dashboard-context";

const Navbar = () => {
  const { me } = useDashboard();
  const initial = (me?.first_name || me?.email || "?").charAt(0).toUpperCase();

  return (
    <nav className="sticky top-0 z-30 flex h-14 items-center justify-between border-b bg-background px-4 sm:px-6">
      <span className="text-lg font-bold tracking-tight">
        Tutor<span className="text-primary">Pilot</span>
      </span>
      <div className="flex items-center gap-3">
        <ThemeToggle />
        <div className="flex items-center gap-2">
          <div className="flex h-8 w-8 items-center justify-center bg-primary text-sm font-semibold text-primary-foreground">
            {initial}
          </div>
        </div>
      </div>
    </nav>
  );
};

export default Navbar;
