"use client";

import { useState } from "react";
import Link from "next/link";
import dynamic from "next/dynamic";
import { useParams } from "next/navigation";
import {
  ArrowLeft,
  CalendarDays,
  Layers,
  Loader2,
  LogIn,
  PlayCircle,
  Square,
  UserRound,
} from "lucide-react";
import { toast } from "sonner";

import { RequirePrivilege } from "@/components/auth/RequirePrivilege";
import PageTheme from "@/components/pagetheme/PageTheme";
import { AttendanceTable } from "@/components/dashboard/lectures/AttendanceTable";
import { RecordingPanel } from "@/components/dashboard/lectures/RecordingPanel";
import {
  LectureStatusBadge,
  RecordingStatusBadge,
  isRecordingPending,
} from "@/components/dashboard/lectures/LectureStatus";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { apiErrorMessage } from "@/lib/api-error";
import {
  useEndLectureMutation,
  useGetLectureQuery,
  useJoinLectureMutation,
  useStartLectureMutation,
} from "@/lib/api/lecturesApi";
import { useCan } from "@/lib/hooks/useCan";
import { PageLoader } from "@/components/layout/PageLoader";

// LiveKit's client is a large webrtc/video bundle that only matters once
// someone actually joins a room, so it is kept out of this page's initial
// JS entirely and fetched on demand.
const LiveRoom = dynamic(
  () => import("@/components/dashboard/lectures/LiveRoom").then((m) => m.LiveRoom),
  { ssr: false, loading: () => <PageLoader label="Connecting to the room…" /> },
);

function LectureDetailContent() {
  const params = useParams<{ id: string }>();
  const id = Number(params.id);
  const can = useCan();

  const [session, setSession] = useState<{ token: string; canPublish: boolean } | null>(null);

  const { data: lecture, isLoading } = useGetLectureQuery(id, {
    // A live lecture and a recording being processed both change on their own, so
    // this refreshes while either is true and settles once the lecture is done.
    pollingInterval: 15_000,
    skipPollingIfUnfocused: true,
  });

  const [startLecture, { isLoading: isStarting }] = useStartLectureMutation();
  const [endLecture, { isLoading: isEnding }] = useEndLectureMutation();
  const [joinLecture, { isLoading: isJoining }] = useJoinLectureMutation();

  if (isLoading || !lecture) {
    return (
      <PageTheme title="Lecture" subtitle="Loading lecture details.">
        <div className="flex h-64 items-center justify-center">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      </PageTheme>
    );
  }

  const isLive = lecture.status === "live";
  const control = can("lecture.control");

  async function handleStart() {
    try {
      await startLecture(id).unwrap();
      toast.success("Lecture started — the room is open");
    } catch (err) {
      toast.error(apiErrorMessage(err as never));
    }
  }

  async function handleEnd() {
    try {
      await endLecture(id).unwrap();
      setSession(null);
      toast.success("Lecture ended. The recording will appear once it is processed.");
    } catch (err) {
      toast.error(apiErrorMessage(err as never));
    }
  }

  async function handleJoin() {
    try {
      const res = await joinLecture(id).unwrap();
      setSession({ token: res.token, canPublish: res.can_publish });
    } catch (err) {
      toast.error(apiErrorMessage(err as never));
    }
  }

  return (
    <PageTheme title={lecture.title} subtitle={lecture.description || "No description provided."}>
      <div className="mb-5">
        <Button variant="ghost" size="sm" asChild className="-ml-2">
          <Link href="/dashboard/lectures">
            <ArrowLeft className="mr-2 h-4 w-4" /> All lectures
          </Link>
        </Button>
      </div>

      <div className="mb-5 flex flex-col gap-4 rounded-xl border bg-card p-5 shadow-sm sm:flex-row sm:items-center sm:justify-between">
        <div className="space-y-3">
          <div className="flex flex-wrap items-center gap-2">
            <LectureStatusBadge status={lecture.status} />
            <RecordingStatusBadge status={lecture.recording_status} />
          </div>
          <dl className="grid gap-1.5 text-sm text-muted-foreground sm:grid-cols-2 sm:gap-x-8">
            <div className="flex items-center gap-2">
              <CalendarDays className="h-3.5 w-3.5 shrink-0" />
              <span>{new Date(lecture.start_time).toLocaleString()}</span>
            </div>
            {lecture.batch_name ? (
              <div className="flex items-center gap-2">
                <Layers className="h-3.5 w-3.5 shrink-0" />
                <Link
                  href={`/dashboard/batches/${lecture.batch_id}`}
                  className="truncate hover:text-primary hover:underline"
                >
                  {lecture.batch_name}
                </Link>
              </div>
            ) : null}
            {lecture.tutor_name ? (
              <div className="flex items-center gap-2">
                <UserRound className="h-3.5 w-3.5 shrink-0" />
                <span className="truncate">{lecture.tutor_name}</span>
              </div>
            ) : null}
            {lecture.module_title ? (
              <div className="flex items-center gap-2">
                <span className="truncate">{lecture.module_title}</span>
              </div>
            ) : null}
          </dl>
        </div>

        <div className="flex flex-wrap gap-2">
          {control && lecture.status === "scheduled" && (
            <Button onClick={handleStart} disabled={isStarting}>
              {isStarting ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <PlayCircle className="mr-2 h-4 w-4" />
              )}
              Start lecture
            </Button>
          )}
          {isLive && !session && can("lecture.join") && (
            <Button onClick={handleJoin} disabled={isJoining}>
              {isJoining ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <LogIn className="mr-2 h-4 w-4" />
              )}
              Join room
            </Button>
          )}
          {control && isLive && (
            <Button variant="destructive" onClick={handleEnd} disabled={isEnding}>
              {isEnding ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Square className="mr-2 h-4 w-4" />
              )}
              End lecture
            </Button>
          )}
        </div>
      </div>

      {/* The live room takes over the page while connected: it is the whole task. */}
      {session ? (
        <LiveRoom
          token={session.token}
          canPublish={session.canPublish}
          onLeave={() => setSession(null)}
        />
      ) : (
        <Tabs defaultValue={isLive ? "room" : "recording"}>
          <TabsList>
            <TabsTrigger value="room">Room</TabsTrigger>
            <TabsTrigger value="recording">
              Recording
              {isRecordingPending(lecture.recording_status) ? " ·" : ""}
            </TabsTrigger>
            <TabsTrigger value="attendance">Attendance</TabsTrigger>
          </TabsList>

          <TabsContent value="room" className="mt-4">
            {isLive ? (
              <div className="flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed px-6 py-12 text-center">
                <p className="font-medium">This lecture is live</p>
                <p className="max-w-md text-sm text-muted-foreground">
                  Join to see and hear the session.
                </p>
                {can("lecture.join") && (
                  <Button onClick={handleJoin} disabled={isJoining}>
                    <LogIn className="mr-2 h-4 w-4" /> Join room
                  </Button>
                )}
              </div>
            ) : lecture.status === "scheduled" ? (
              <div className="flex flex-col items-center justify-center gap-2 rounded-lg border border-dashed px-6 py-12 text-center">
                <p className="font-medium">The room is not open yet</p>
                <p className="max-w-md text-sm text-muted-foreground">
                  It opens when the lecture is started
                  {control ? "" : " by the tutor"}.
                </p>
              </div>
            ) : (
              <div className="flex flex-col items-center justify-center gap-2 rounded-lg border border-dashed px-6 py-12 text-center">
                <p className="font-medium">
                  This lecture has {lecture.status === "cancelled" ? "been cancelled" : "ended"}
                </p>
                <p className="max-w-md text-sm text-muted-foreground">
                  {lecture.status === "ended"
                    ? "The room is closed. Check the recording tab."
                    : "It was called off before it ran."}
                </p>
              </div>
            )}
          </TabsContent>

          <TabsContent value="recording" className="mt-4">
            <RecordingPanel lecture={lecture} />
          </TabsContent>

          <TabsContent value="attendance" className="mt-4">
            <AttendanceTable lectureID={id} live={isLive} />
          </TabsContent>
        </Tabs>
      )}
    </PageTheme>
  );
}

export default function LectureDetailPage() {
  return (
    <RequirePrivilege need="lecture.view">
      <LectureDetailContent />
    </RequirePrivilege>
  );
}
