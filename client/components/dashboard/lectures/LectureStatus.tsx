"use client";

import {
  AlertTriangle,
  CalendarClock,
  CheckCircle2,
  CircleSlash,
  Disc,
  Loader2,
  Radio,
  Video,
} from "lucide-react";

import { cn } from "@/lib/utils";
import type { LectureStatus, RecordingStatus } from "@/lib/types";

/** Colour and wording for each lecture state, kept in one place so the list, the
 *  detail page and the cards cannot drift apart. */
const STATUS_STYLES: Record<
  LectureStatus,
  { label: string; className: string; Icon: typeof Radio }
> = {
  scheduled: {
    label: "Scheduled",
    className: "border-sky-500/30 bg-sky-500/10 text-sky-600 dark:text-sky-400",
    Icon: CalendarClock,
  },
  live: {
    label: "Live",
    className: "border-red-500/30 bg-red-500/10 text-red-600 dark:text-red-400",
    Icon: Radio,
  },
  ended: {
    label: "Ended",
    className: "border-border bg-muted text-muted-foreground",
    Icon: CheckCircle2,
  },
  cancelled: {
    label: "Cancelled",
    className: "border-border bg-muted/60 text-muted-foreground line-through",
    Icon: CircleSlash,
  },
};

export function LectureStatusBadge({
  status,
  className,
}: {
  status: LectureStatus;
  className?: string;
}) {
  const { label, className: tone, Icon } = STATUS_STYLES[status] ?? STATUS_STYLES.scheduled;
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs font-medium",
        tone,
        className,
      )}
    >
      {/* A live lecture pulses, so it is findable at a glance in a long list. */}
      <Icon className={cn("h-3 w-3", status === "live" && "animate-pulse")} />
      {label}
    </span>
  );
}

/** Recording states the user needs to understand. "processing" is the one that
 *  matters most: the lecture has ended but the file does not exist yet, and without
 *  saying so the UI looks broken. */
const RECORDING_STYLES: Record<
  RecordingStatus,
  { label: string; className: string; Icon: typeof Disc; spin?: boolean } | null
> = {
  none: null,
  starting: {
    label: "Preparing recording",
    className: "border-border bg-muted text-muted-foreground",
    Icon: Loader2,
    spin: true,
  },
  recording: {
    label: "Recording",
    className: "border-red-500/30 bg-red-500/10 text-red-600 dark:text-red-400",
    Icon: Disc,
  },
  processing: {
    label: "Processing recording",
    className: "border-amber-500/30 bg-amber-500/10 text-amber-600 dark:text-amber-400",
    Icon: Loader2,
    spin: true,
  },
  ready: {
    label: "Recording ready",
    className: "border-emerald-500/30 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
    Icon: Video,
  },
  failed: {
    label: "Recording failed",
    className: "border-destructive/30 bg-destructive/10 text-destructive",
    Icon: AlertTriangle,
  },
};

export function RecordingStatusBadge({
  status,
  className,
}: {
  status: RecordingStatus;
  className?: string;
}) {
  const style = RECORDING_STYLES[status];
  if (!style) return null;

  const { label, className: tone, Icon, spin } = style;
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs font-medium",
        tone,
        className,
      )}
    >
      <Icon className={cn("h-3 w-3", spin && "animate-spin")} />
      {label}
    </span>
  );
}

/** True while the recording pipeline is still working, so the caller knows to poll. */
export function isRecordingPending(status: RecordingStatus): boolean {
  return status === "starting" || status === "recording" || status === "processing";
}

export function formatDuration(seconds?: number): string {
  if (!seconds || seconds <= 0) return "—";
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.floor(seconds % 60);
  if (h > 0) return `${h}h ${String(m).padStart(2, "0")}m`;
  if (m > 0) return `${m}m ${String(s).padStart(2, "0")}s`;
  return `${s}s`;
}

export function formatBytes(bytes?: number): string {
  if (!bytes || bytes <= 0) return "—";
  const units = ["B", "KB", "MB", "GB"];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  return `${value.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`;
}
