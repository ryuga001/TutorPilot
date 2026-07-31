import { CalendarDays, ShieldCheck, Video } from "lucide-react";

import { Logo } from "@/components/brand/Logo";
import { ThemeToggle } from "@/components/theme-toggle";

const HIGHLIGHTS = [
  { icon: Video, text: "Live classes with built-in recording" },
  { icon: CalendarDays, text: "Batches, tutors and students in one place" },
  { icon: ShieldCheck, text: "Role-aware access for every organization" },
];

export default function AuthLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex min-h-screen">
      {/* Brand panel — hidden on small screens, where a split layout has no
          room to breathe; those get the compact header below instead. */}
      <aside className="relative hidden w-[42%] shrink-0 overflow-hidden bg-gradient-to-br from-primary via-primary to-brand-secondary lg:flex lg:flex-col lg:justify-between lg:p-10 xl:p-14">
        <div className="bg-brand-dots pointer-events-none absolute inset-0" aria-hidden="true" />
        <div className="relative">
          <Logo light />
        </div>
        <div className="relative space-y-8 text-white">
          <h2 className="font-heading text-3xl font-bold leading-tight xl:text-4xl">
            Run your tutoring organization from one dashboard.
          </h2>
          <ul className="space-y-4">
            {HIGHLIGHTS.map(({ icon: Icon, text }) => (
              <li key={text} className="flex items-center gap-3 text-white/90">
                <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-white/15">
                  <Icon className="h-4 w-4" />
                </span>
                <span className="text-sm font-medium">{text}</span>
              </li>
            ))}
          </ul>
        </div>
        <p className="relative text-xs text-white/70">
          &copy; {new Date().getFullYear()} TutorPilot. All rights reserved.
        </p>
      </aside>

      <div className="flex flex-1 flex-col">
        <header className="flex items-center justify-between border-b px-6 py-4 lg:justify-end lg:border-none">
          <Logo className="lg:hidden" />
          <ThemeToggle />
        </header>
        <main className="flex flex-1 items-center justify-center p-6">
          <div className="w-full max-w-md animate-in fade-in slide-in-from-bottom-2 duration-300">
            {children}
          </div>
        </main>
      </div>
    </div>
  );
}
