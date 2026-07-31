"use client";

import { useState } from "react";
import { Menu } from "lucide-react";

import { Logo } from "@/components/brand/Logo";
import { SidebarNav } from "@/components/dashboard/sidebard/Sidebar";
import { ThemeToggle } from "@/components/theme-toggle";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useDashboard } from "@/lib/dashboard-context";

const Navbar = () => {
  const { me } = useDashboard();
  const [mobileOpen, setMobileOpen] = useState(false);
  const initial = (me?.first_name || me?.email || "?").charAt(0).toUpperCase();

  return (
    <nav className="sticky top-0 z-30 flex h-14 items-center justify-between border-b bg-background/80 px-4 backdrop-blur-md sm:px-6">
      <div className="flex items-center gap-2">
        <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
          <SheetContent
            side="left"
            className="flex w-72 max-w-[85vw] flex-col gap-0 p-0 sm:max-w-xs"
          >
            <SheetHeader className="border-b p-4">
              <SheetTitle asChild>
                <span>
                  <Logo />
                </span>
              </SheetTitle>
            </SheetHeader>
            <SidebarNav onNavigate={() => setMobileOpen(false)} />
          </SheetContent>
          <Button
            variant="ghost"
            size="icon"
            className="md:hidden"
            aria-label="Open navigation menu"
            onClick={() => setMobileOpen(true)}
          >
            <Menu className="h-5 w-5" />
          </Button>
        </Sheet>
        <Logo className="hidden sm:flex" />
        <Logo iconOnly className="sm:hidden" />
      </div>
      <div className="flex items-center gap-3">
        <ThemeToggle />
        <Avatar className="h-8 w-8">
          <AvatarFallback>{initial}</AvatarFallback>
        </Avatar>
      </div>
    </nav>
  );
};

export default Navbar;
