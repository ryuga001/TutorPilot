"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";
import {
  BookOpen,
  CalendarDays,
  ChevronsLeft,
  GraduationCap,
  LayoutDashboard,
  Loader2,
  LogOut,
  Users,
  Video,
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
  { href: "/dashboard/lectures", label: "Lectures", icon: Video, privilege: "lecture.view" },
  { href: "/dashboard/tutors", label: "Tutors", icon: Users, privilege: "tutor.view" },
  { href: "/dashboard/students", label: "Students", icon: GraduationCap, privilege: "student.view" },
];

interface SidebarNavProps {
  collapsed?: boolean;
  onNavigate?: () => void;
}

/** The nav-link list, shared between the desktop rail and the mobile drawer. */
export function SidebarNav({ collapsed, onNavigate }: SidebarNavProps) {
  const pathname = usePathname();
  const can = useCan();
  const items = NAV_ITEMS.filter((i) => !i.privilege || can(i.privilege));

  return (
    <nav aria-label="Primary" className="flex flex-1 flex-col gap-1 p-3">
      {items.map(({ href, label, icon: Icon }) => {
        const active = href === "/dashboard" ? pathname === href : pathname.startsWith(href);
        return (
          <Link
            key={href}
            href={href}
            onClick={onNavigate}
            aria-current={active ? "page" : undefined}
            title={collapsed ? label : undefined}
            className={cn(
              "group flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors duration-150",
              "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
              active
                ? "bg-primary text-primary-foreground shadow-soft"
                : "text-muted-foreground hover:bg-accent hover:text-accent-foreground",
              collapsed && "justify-center px-2",
            )}
          >
            <Icon className="h-4 w-4 shrink-0" />
            {!collapsed && <span className="truncate">{label}</span>}
          </Link>
        );
      })}
    </nav>
  );
}

const COLLAPSE_STORAGE_KEY = "tutorpilot:sidebar-collapsed";

const Sidebar = () => {
  const { handleLogout, loggingOut } = useDashboard();
  const [collapsed, setCollapsed] = useState(false);

  useEffect(() => {
    setCollapsed(window.localStorage.getItem(COLLAPSE_STORAGE_KEY) === "1");
  }, []);

  function toggleCollapsed() {
    setCollapsed((prev) => {
      const next = !prev;
      window.localStorage.setItem(COLLAPSE_STORAGE_KEY, next ? "1" : "0");
      return next;
    });
  }

  return (
    <div
      className={cn(
        "hidden h-full shrink-0 flex-col border-r bg-background transition-[width] duration-200 md:flex",
        collapsed ? "w-16" : "w-56",
      )}
    >
      <SidebarNav collapsed={collapsed} />

      <div className="mt-auto space-y-1 border-t p-3">
        <Button
          variant="ghost"
          onClick={toggleCollapsed}
          className={cn("w-full text-muted-foreground", collapsed ? "justify-center px-2" : "justify-start gap-3")}
          aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
          title={collapsed ? "Expand sidebar" : undefined}
        >
          <ChevronsLeft className={cn("h-4 w-4 shrink-0 transition-transform", collapsed && "rotate-180")} />
          {!collapsed && "Collapse"}
        </Button>
        <Button
          variant="destructive"
          onClick={handleLogout}
          disabled={loggingOut}
          className={cn("w-full", collapsed ? "justify-center px-2" : "justify-start gap-3")}
          title={collapsed ? "Sign out" : undefined}
        >
          {loggingOut ? (
            <Loader2 className="h-4 w-4 shrink-0 animate-spin" />
          ) : (
            <LogOut className="h-4 w-4 shrink-0" />
          )}
          {!collapsed && "Sign out"}
        </Button>
      </div>
    </div>
  );
};

export default Sidebar;
