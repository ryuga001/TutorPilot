"use client";

import Link from "next/link";
import {
  CalendarDays,
  GraduationCap,
  Layers,
  MoreHorizontal,
  Pencil,
  PlayCircle,
  Square,
  Trash2,
  UserRound,
  XCircle,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  LectureStatusBadge,
  RecordingStatusBadge,
} from "@/components/dashboard/lectures/LectureStatus";
import type { Lecture } from "@/lib/types";

interface Props {
  lecture: Lecture;
  can: (privilege: string) => boolean;
  busy?: boolean;
  onStart: (lecture: Lecture) => void;
  onEnd: (lecture: Lecture) => void;
  onCancel: (lecture: Lecture) => void;
  onEdit: (lecture: Lecture) => void;
  onDelete: (lecture: Lecture) => void;
}

export function LectureCard({
  lecture,
  can,
  busy,
  onStart,
  onEnd,
  onCancel,
  onEdit,
  onDelete,
}: Props) {
  // The state machine decides what is offered, so the UI never presents an action
  // the API will refuse: a lecture can only start from scheduled and only end from
  // live, and neither is possible once it has ended or been cancelled.
  const canStart = lecture.status === "scheduled";
  const canEnd = lecture.status === "live";
  const canCancel = lecture.status === "scheduled";
  const canEdit = lecture.status === "scheduled" || lecture.status === "live";
  const canDelete = lecture.status !== "live";

  const control = can("lecture.control");

  return (
    <div className="group flex flex-col rounded-xl border bg-card p-5 shadow-sm transition-shadow hover:shadow-md">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 space-y-2">
          <div className="flex flex-wrap items-center gap-2">
            <LectureStatusBadge status={lecture.status} />
            <RecordingStatusBadge status={lecture.recording_status} />
          </div>
          <Link
            href={`/dashboard/lectures/${lecture.id}`}
            className="block truncate text-base font-semibold hover:text-primary hover:underline"
          >
            {lecture.title}
          </Link>
          {lecture.description ? (
            <p className="line-clamp-2 text-sm text-muted-foreground">{lecture.description}</p>
          ) : null}
        </div>

        {(control || can("lecture.edit") || can("lecture.delete")) && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" className="h-8 w-8 shrink-0">
                <MoreHorizontal className="h-4 w-4" />
                <span className="sr-only">Lecture actions</span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {can("lecture.edit") && (
                <DropdownMenuItem disabled={!canEdit} onClick={() => onEdit(lecture)}>
                  <Pencil className="mr-2 h-4 w-4" /> Edit
                </DropdownMenuItem>
              )}
              {control && canCancel && (
                <DropdownMenuItem onClick={() => onCancel(lecture)}>
                  <XCircle className="mr-2 h-4 w-4" /> Cancel lecture
                </DropdownMenuItem>
              )}
              {can("lecture.delete") && (
                <>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    disabled={!canDelete}
                    onClick={() => onDelete(lecture)}
                    className="text-destructive focus:text-destructive"
                  >
                    <Trash2 className="mr-2 h-4 w-4" /> Delete
                  </DropdownMenuItem>
                </>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>

      <dl className="mt-4 grid gap-1.5 text-sm text-muted-foreground">
        <div className="flex items-center gap-2">
          <CalendarDays className="h-3.5 w-3.5 shrink-0" />
          <span>{new Date(lecture.start_time).toLocaleString()}</span>
        </div>
        {lecture.batch_name ? (
          <div className="flex items-center gap-2">
            <Layers className="h-3.5 w-3.5 shrink-0" />
            <span className="truncate">
              {lecture.batch_name}
              {lecture.course_title ? ` · ${lecture.course_title}` : ""}
            </span>
          </div>
        ) : null}
        {lecture.module_title ? (
          <div className="flex items-center gap-2">
            <GraduationCap className="h-3.5 w-3.5 shrink-0" />
            <span className="truncate">{lecture.module_title}</span>
          </div>
        ) : null}
        {lecture.tutor_name ? (
          <div className="flex items-center gap-2">
            <UserRound className="h-3.5 w-3.5 shrink-0" />
            <span className="truncate">{lecture.tutor_name}</span>
          </div>
        ) : null}
      </dl>

      <div className="mt-5 flex flex-wrap gap-2 border-t pt-4">
        {control && canStart && (
          <Button size="sm" disabled={busy} onClick={() => onStart(lecture)}>
            <PlayCircle className="mr-2 h-4 w-4" /> Start
          </Button>
        )}
        {control && canEnd && (
          <Button size="sm" variant="destructive" disabled={busy} onClick={() => onEnd(lecture)}>
            <Square className="mr-2 h-4 w-4" /> End
          </Button>
        )}
        <Button size="sm" variant={canEnd ? "default" : "secondary"} asChild>
          <Link href={`/dashboard/lectures/${lecture.id}`}>
            {canEnd ? "Join now" : "Open"}
          </Link>
        </Button>
      </div>
    </div>
  );
}
