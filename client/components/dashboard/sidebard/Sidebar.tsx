"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  Loader2,
  LogOut,
  Mail,
  Shield,
  Users,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { useDashboard } from "@/lib/dashboard-context";
import { cn } from "@/lib/utils";

const NAV_ITEMS = [
  { href: "/dashboard", label: "Overview", icon: LayoutDashboard },
  { href: "/dashboard/tutors", label: "Tutors", icon: Users },
  { href: "/dashboard/courses", label: "Courses", icon: Shield },
  { href: "/dashboard/students", label: "Students", icon: Users },
];

const Sidebar = () => {
  const pathname = usePathname();
  const { handleLogout, loggingOut } = useDashboard();

  return (
    <div className="hidden h-full w-56 shrink-0 flex-col border-r bg-background md:flex">
      <nav className="flex flex-1 flex-col gap-1 p-3">
        {NAV_ITEMS.map(({ href, label, icon: Icon }) => {
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
          variant="ghost"
          onClick={handleLogout}
          disabled={loggingOut}
          className="w-full justify-start gap-3 text-muted-foreground hover:text-foreground"
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
