import type { FetchBaseQueryError } from "@reduxjs/toolkit/query";
import type { SerializedError } from "@reduxjs/toolkit";

/** Pulls a human-readable message out of an RTK Query error. */
export function apiErrorMessage(
  err: FetchBaseQueryError | SerializedError | undefined,
  fallback = "Something went wrong. Please try again.",
): string {
  if (!err) return fallback;

  if ("status" in err) {
    const data = err.data as { error?: string; message?: string } | undefined;
    if (data?.error) return data.error;
    if (data?.message) return data.message;
    if (err.status === "FETCH_ERROR") return "Cannot reach the server.";
    return fallback;
  }

  return err.message ?? fallback;
}
