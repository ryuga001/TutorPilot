"use client";

import type { ReactNode } from "react";
import { AlertTriangle, Clock, FolderOpen, HardDrive, Loader2, VideoOff } from "lucide-react";
import Link from "next/link";

import { Button } from "@/components/ui/button";
import { formatBytes, formatDuration } from "@/components/dashboard/lectures/LectureStatus";
import type { Lecture } from "@/lib/types";

/**
 * The recording, or an honest account of why there isn't one yet.
 *
 * A recording does not exist the moment a lecture ends — egress finalises
 * asynchronously and reports back by webhook — so each state gets its own message
 * rather than a player that silently fails to load.
 */
export function RecordingPanel({ lecture }: { lecture: Lecture }) {
  if (!lecture.recording_enabled) {
    return (
      <Empty
        icon={<VideoOff className="h-5 w-5" />}
        title="Recording was off"
        description="This lecture was not recorded."
      />
    );
  }

  switch (lecture.recording_status) {
    case "none":
      return (
        <Empty
          icon={<Clock className="h-5 w-5" />}
          title="Not recorded yet"
          description="Recording begins when the lecture starts."
        />
      );

    case "starting":
    case "recording":
      return (
        <Empty
          icon={<Loader2 className="h-5 w-5 animate-spin" />}
          title="Recording in progress"
          description="The file becomes available a few minutes after the lecture ends."
        />
      );

    case "processing":
      return (
        <Empty
          icon={<Loader2 className="h-5 w-5 animate-spin" />}
          title="Processing the recording"
          description="The lecture has ended and the video is being finalised. This page updates itself when it is ready."
        />
      );

    case "failed":
      return (
        <Empty
          icon={<AlertTriangle className="h-5 w-5 text-destructive" />}
          title="Recording failed"
          description="Something went wrong while recording this lecture, so no video was produced."
        />
      );

    case "ready":
      break;
  }

  if (!lecture.recording_url) {
    return (
      <Empty
        icon={<AlertTriangle className="h-5 w-5 text-destructive" />}
        title="Recording is missing"
        description="The lecture was recorded but the file could not be located."
      />
    );
  }

  return (
    <div className="space-y-3">
      <div className="overflow-hidden rounded-lg border bg-black">
        {/*
          Played straight from the object store. A <video> element cannot attach an
          Authorization header, so it cannot go through the API's own
          /lectures/:id/recording route — that one exists for authenticated clients
          that follow its redirect. Once the recordings bucket is made private this
          has to become a presigned URL fetched from the API first.
        */}
        <video
          controls
          preload="metadata"
          className="aspect-video w-full"
          src={lecture.recording_url}
        >
          <track kind="captions" />
        </video>
      </div>

      <div className="flex flex-wrap items-center gap-x-5 gap-y-2 text-sm text-muted-foreground">
        <span className="inline-flex items-center gap-1.5">
          <Clock className="h-3.5 w-3.5" />
          {formatDuration(lecture.recording_duration_seconds)}
        </span>
        <span className="inline-flex items-center gap-1.5">
          <HardDrive className="h-3.5 w-3.5" />
          {formatBytes(lecture.recording_size_bytes)}
        </span>
        <Button variant="link" size="sm" className="h-auto p-0" asChild>
          <Link href={`/dashboard/batches/${lecture.batch_id}?tab=drive`}>
            <FolderOpen className="mr-1.5 h-3.5 w-3.5" />
            Find it in the batch drive
          </Link>
        </Button>
      </div>
    </div>
  );
}

function Empty({
  icon,
  title,
  description,
}: {
  icon: ReactNode;
  title: string;
  description: string;
}) {
  return (
    <div className="flex flex-col items-center justify-center gap-2 rounded-lg border border-dashed px-6 py-12 text-center">
      <div className="flex h-10 w-10 items-center justify-center rounded-full bg-muted text-muted-foreground">
        {icon}
      </div>
      <p className="font-medium">{title}</p>
      <p className="max-w-md text-sm text-muted-foreground">{description}</p>
    </div>
  );
}
