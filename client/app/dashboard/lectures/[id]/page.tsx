"use client";

import { useMemo, useState } from "react";
import { useParams } from "next/navigation";
import { ExternalLink, Loader2, Video } from "lucide-react";
import { LiveKitRoom, VideoConference } from "@livekit/components-react";
import "@livekit/components-styles";

function RecordingPlayer({ src }: { src: string }) {
  return (
    <div className="overflow-hidden rounded-lg border bg-black">
      <video controls className="aspect-video w-full" src={src}>
        <track kind="captions" />
      </video>
    </div>
  );
}

function resolveLiveKitServerUrl() {
  const raw = process.env.NEXT_PUBLIC_LIVEKIT_URL;
  if (raw) {
    if (raw.startsWith("https://")) return raw.replace("https://", "wss://");
    if (raw.startsWith("http://")) return raw.replace("http://", "ws://");
    return raw;
  }

  if (typeof window !== "undefined") {
    const host = window.location.hostname === "localhost" || window.location.hostname === "127.0.0.1" || window.location.hostname === "::1"
      ? "127.0.0.1"
      : window.location.hostname;
    return `${window.location.protocol === "https:" ? "wss" : "ws"}://${host}:7880`;
  }

  return "ws://127.0.0.1:7880";
}

function LiveRoomPreview({ roomName, token }: { roomName?: string; token?: string }) {
  const [connectionError, setConnectionError] = useState<string | null>(null);

  if (!roomName && !token) {
    return (
      <div className="rounded-lg border border-primary/20 bg-background/80 p-4">
        <div className="mb-2 flex items-center justify-between">
          <div className="font-medium">Live room preview</div>
          <div className="rounded-full bg-primary/10 px-2 py-1 text-xs text-primary">{roomName || "room unavailable"}</div>
        </div>
        <div className="rounded-md border border-dashed bg-muted/40 p-3 text-sm text-muted-foreground">
          Join the room to generate a session token and open the live room experience.
        </div>
      </div>
    );
  }

  const serverUrl = resolveLiveKitServerUrl();

  return (
    <div className="space-y-3 rounded-lg border border-primary/20 bg-background/80 p-4">
      <div className="flex items-center justify-between">
        <div className="font-medium">Live room</div>
        <div className="rounded-full bg-primary/10 px-2 py-1 text-xs text-primary">{roomName}</div>
      </div>
      {connectionError ? (
        <div className="rounded-md border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
          {connectionError}
        </div>
      ) : null}
      <LiveKitRoom
        token={token}
        serverUrl={serverUrl}
        connect={Boolean(token)}
        audio={true}
        video={true}
        onError={(error) => setConnectionError(error.message || "Unable to connect to the live room")}
        options={{ adaptiveStream: true, dynacast: true }}
      >
        <VideoConference />
      </LiveKitRoom>
    </div>
  );
}

import { RequirePrivilege } from "@/components/auth/RequirePrivilege";
import PageTheme from "@/components/pagetheme/PageTheme";
import { Button } from "@/components/ui/button";
import { useGetLectureQuery, useJoinLectureMutation } from "@/lib/api/lecturesApi";

function LectureDetailPageContent() {
  const params = useParams<{ id: string }>();
  const id = Number(params.id);
  const { data, isLoading } = useGetLectureQuery(id);
  const [joinLecture, { isLoading: isJoining }] = useJoinLectureMutation();
  const [token, setToken] = useState<string | null>(null);

  const canJoin = useMemo(() => Boolean(data?.roomName), [data?.roomName]);

  const handleJoin = async () => {
    if (!id) return;
    const res = await joinLecture(id);
    const payload = res.data as { token?: string } | undefined;
    if (payload?.token) {
      setToken(payload.token);
    }
  };

  if (isLoading || !data) {
    return (
      <PageTheme title="Lecture" subtitle="Loading lecture details.">
        <div className="flex h-40 items-center justify-center">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      </PageTheme>
    );
  }

  return (
    <PageTheme title={data.title} subtitle="Join the lecture room, view recordings, and stream the session.">
      <div className="space-y-4 rounded-lg border bg-card p-6 shadow-sm">
        <div className="flex items-center gap-2">
          <Video className="h-5 w-5 text-primary" />
          <div>
            <div className="font-semibold">{data.title}</div>
            <div className="text-sm text-muted-foreground">{data.description || "No description provided."}</div>
          </div>
        </div>

        <div className="grid gap-3 rounded-md border bg-muted/30 p-4 md:grid-cols-2">
          <div><strong>Status:</strong> {data.status}</div>
          <div><strong>Room:</strong> {data.roomName || "Not available"}</div>
          <div><strong>Recording enabled:</strong> {data.recordingEnabled ? "Yes" : "No"}</div>
          <div><strong>Started:</strong> {new Date(data.startTime).toLocaleString()}</div>
        </div>

        <div className="flex flex-wrap gap-2">
          <Button onClick={handleJoin} disabled={!canJoin || isJoining}>
            {isJoining ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
            Join room
          </Button>
          {data.recordingUrl && (
            <Button variant="outline" asChild>
              <a href={data.recordingUrl} target="_blank" rel="noreferrer">
                <ExternalLink className="mr-2 h-4 w-4" /> Open recording
              </a>
            </Button>
          )}
        </div>

        {token || data.roomName ? (
          <LiveRoomPreview roomName={data.roomName} token={token} />
        ) : data.recordingUrl ? (
          <RecordingPlayer src={data.recordingUrl} />
        ) : (
          <div className="rounded-md border border-dashed bg-muted/30 p-4 text-sm text-muted-foreground">No recording is available yet.</div>
        )}

        {token && (
          <div className="rounded-md border border-dashed bg-muted/30 p-3 text-sm">
            <div className="font-medium">Session token ready</div>
            <div className="mt-1 break-all font-mono text-xs">{token}</div>
          </div>
        )}
      </div>
    </PageTheme>
  );
}

export default function LectureDetailPage() {
  return (
    <RequirePrivilege need="lecture.view">
      <LectureDetailPageContent />
    </RequirePrivilege>
  );
}
