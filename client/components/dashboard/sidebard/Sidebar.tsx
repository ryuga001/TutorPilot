"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  BookOpen,
  CalendarDays,
  GraduationCap,
  LayoutDashboard,
  Loader2,
  LogOut,
  Users,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { useDashboard } from "@/lib/dashboard-context";
import { useCan } from "@/lib/hooks/useCan";
import { cn } from "@/lib/utils";

const NAV_ITEMS: {
  href: string;
  label: string;
  icon: typeof LayoutDashboard;
  privilege?: string;
}[] = [
  { href: "/dashboard", label: "Overview", icon: LayoutDashboard },
  { href: "/dashboard/courses", label: "Courses", icon: BookOpen, privilege: "course.view" },
  { href: "/dashboard/batches", label: "Batches", icon: CalendarDays, privilege: "batch.view" },
  { href: "/dashboard/tutors", label: "Tutors", icon: Users, privilege: "tutor.view" },
  { href: "/dashboard/students", label: "Students", icon: GraduationCap, privilege: "student.view" },
];

const Sidebar = () => {
  const pathname = usePathname();
  const can = useCan();
  const { handleLogout, loggingOut } = useDashboard();

  const items = NAV_ITEMS.filter((i) => !i.privilege || can(i.privilege));

  return (
    <div className="hidden h-full w-56 shrink-0 flex-col border-r bg-background md:flex">
      <nav className="flex flex-1 flex-col gap-1 p-3">
        {items.map(({ href, label, icon: Icon }) => {
          const active =
            href === "/dashboard"
              ? pathname === href
              : pathname.startsWith(href);
          return (
            <Link
              key={href}
              href={href}
              className={cn(
                "flex items-center gap-3 px-3 py-2 text-sm font-medium transition-colors",
                active
                  ? "bg-primary text-primary-foreground"
                  : "text-muted-foreground hover:bg-accent hover:text-accent-foreground",
              )}
            >
              <Icon className="h-4 w-4" />
              {label}
            </Link>
          );
        })}
      </nav>

      <div className="mt-auto border-t p-3">
        <Button
          variant="destructive"
          onClick={handleLogout}
          disabled={loggingOut}
          className="w-full justify-start gap-3"
        >
          {loggingOut ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <LogOut className="h-4 w-4" />
          )}
          Sign out
        </Button>
      </div>
    </div>
  );
};

export default Sidebar;
