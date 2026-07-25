"use client";

import { useEffect, useState, type ReactNode } from "react";
import { useRouter } from "next/navigation";
import { Loader2 } from "lucide-react";

import { useAppSelector } from "@/lib/hooks";

export function AuthGuard({ children }: { children: ReactNode }) {
  const router = useRouter();
  const accessToken = useAppSelector((s) => s.auth.accessToken);
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  useEffect(() => {
    if (mounted && !accessToken) {
      router.replace("/login");
    }
  }, [mounted, accessToken, router]);

  if (!mounted) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!accessToken) return null;

  return <>{children}</>;
}
