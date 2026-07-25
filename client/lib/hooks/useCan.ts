"use client";

import { useAppSelector } from "@/lib/hooks";

export function useCan() {
  const privileges = useAppSelector((s) => s.auth.privileges);
  return (privilege: string) => (privileges ?? []).includes(privilege);
}

export function usePrivilegesReady() {
  return useAppSelector((s) => s.auth.privileges !== null);
}
