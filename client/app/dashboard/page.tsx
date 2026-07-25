"use client";

import { Loader2 } from "lucide-react";

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { useDashboard } from "@/lib/dashboard-context";

export default function DashboardPage() {
  const { me, isLoading, isError } = useDashboard();

  const fields: { label: string; value: string | number | undefined }[] = [
    { label: "User ID", value: me?.id },
    { label: "Email", value: me?.email },
    { label: "Role", value: me?.role },
    { label: "Customer ID", value: me?.customer_id },
    {
      label: "Member since",
      value: me?.created_at
        ? new Date(me.created_at).toLocaleDateString()
        : undefined,
    },
  ];

  return (
    <div className="mx-auto w-full max-w-3xl">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold tracking-tight">
          Welcome{me ? `, ${me.email}` : ""}
        </h1>
        <p className="text-sm text-muted-foreground">
          You are signed in to the TutorPilot dashboard.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Account</CardTitle>
          <CardDescription>Details from GET /auth/me.</CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" /> Loading…
            </div>
          )}
          {isError && (
            <p className="text-sm text-destructive">
              Could not load your profile.
            </p>
          )}
          {me && (
            <dl className="divide-y">
              {fields.map((f) => (
                <div
                  key={f.label}
                  className="flex items-center justify-between py-3"
                >
                  <dt className="text-sm text-muted-foreground">{f.label}</dt>
                  <dd className="text-sm font-medium">{f.value ?? "—"}</dd>
                </div>
              ))}
            </dl>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
