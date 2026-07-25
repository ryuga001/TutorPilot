"use client";

import { createContext, useCallback, useContext, type ReactNode } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";

import { authApi, useGetMeQuery, useLogoutMutation } from "@/lib/api/authApi";
import { logout as logoutAction } from "@/lib/features/authSlice";
import { useAppDispatch, useAppSelector } from "@/lib/hooks";
import type { UserView } from "@/lib/types";

interface DashboardContextValue {
  me?: UserView;
  isLoading: boolean;
  isError: boolean;
  loggingOut: boolean;
  handleLogout: () => Promise<void>;
}

const DashboardContext = createContext<DashboardContextValue | null>(null);


export function DashboardProvider({ children }: { children: ReactNode }) {
  const { data: me, isLoading, isError } = useGetMeQuery();
  const [logout, { isLoading: loggingOut }] = useLogoutMutation();
  const dispatch = useAppDispatch();
  const router = useRouter();
  const refreshToken = useAppSelector((s) => s.auth.refreshToken);

  const handleLogout = useCallback(async () => {
    try {
      if (refreshToken) {
        await logout({ refresh_token: refreshToken }).unwrap();
      }
    } catch {
      /* ignore: we clear the session locally regardless */
    } finally {
      dispatch(logoutAction());
      dispatch(authApi.util.resetApiState());
      toast.success("Signed out.");
      router.replace("/login");
    }
  }, [refreshToken, logout, dispatch, router]);

  return (
    <DashboardContext.Provider
      value={{ me, isLoading, isError, loggingOut, handleLogout }}
    >
      {children}
    </DashboardContext.Provider>
  );
}

export function useDashboard() {
  const ctx = useContext(DashboardContext);
  if (!ctx) {
    throw new Error("useDashboard must be used within a DashboardProvider");
  }
  return ctx;
}
