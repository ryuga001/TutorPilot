"use client";

import { Loader2, Users } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { formatDuration } from "@/components/dashboard/lectures/LectureStatus";
import { useLectureAttendanceQuery } from "@/lib/api/lecturesApi";

/**
 * Who attended, built from LiveKit's participant events. A row with no leave time is
 * someone still connected, which is why this polls while a lecture is live.
 */
export function AttendanceTable({ lectureID, live }: { lectureID: number; live: boolean }) {
  const { data, isLoading } = useLectureAttendanceQuery(lectureID, {
    pollingInterval: live ? 15_000 : 0,
    skipPollingIfUnfocused: true,
  });

  if (isLoading) {
    return (
      <div className="flex h-32 items-center justify-center">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    );
  }

  const rows = data ?? [];
  if (rows.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-2 rounded-lg border border-dashed px-6 py-12 text-center">
        <div className="flex h-10 w-10 items-center justify-center rounded-full bg-muted">
          <Users className="h-5 w-5 text-muted-foreground" />
        </div>
        <p className="font-medium">No attendance recorded</p>
        <p className="max-w-md text-sm text-muted-foreground">
          {live
            ? "Nobody has joined the room yet."
            : "Attendance is recorded when participants join the live room."}
        </p>
      </div>
    );
  }

  return (
    <div className="rounded-lg border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Participant</TableHead>
            <TableHead>Role</TableHead>
            <TableHead>Joined</TableHead>
            <TableHead>Left</TableHead>
            <TableHead className="text-right">Time present</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((row, i) => {
            const stillPresent = !row.left_at;
            return (
              <TableRow key={`${row.user_id}-${row.joined_at}-${i}`}>
                <TableCell className="font-medium">{row.display_name || "—"}</TableCell>
                <TableCell>
                  <Badge variant="secondary" className="capitalize">
                    {row.subject_type}
                  </Badge>
                </TableCell>
                <TableCell className="text-muted-foreground">
                  {new Date(row.joined_at).toLocaleTimeString()}
                </TableCell>
                <TableCell className="text-muted-foreground">
                  {stillPresent ? (
                    <span className="inline-flex items-center gap-1.5 text-emerald-600 dark:text-emerald-400">
                      <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-current" />
                      In the room
                    </span>
                  ) : (
                    new Date(row.left_at!).toLocaleTimeString()
                  )}
                </TableCell>
                <TableCell className="text-right text-muted-foreground">
                  {stillPresent ? "—" : formatDuration(row.seconds_present)}
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
