"use client";

import { useState } from "react";
import { LiveKitRoom, VideoConference } from "@livekit/components-react";
import "@livekit/components-styles";
import { AlertTriangle, Eye } from "lucide-react";

/**
 * Resolves the LiveKit websocket address.
 *
 * LiveKit is a separate service on its own port, so unlike the API its host is not
 * derived from the tenant subdomain — every organization shares one media server and
 * rooms are already namespaced by name.
 */
function liveKitServerUrl(): string {
  const configured = process.env.NEXT_PUBLIC_LIVEKIT_URL;
  if (configured) {
    return configured
      .replace(/^https:\/\//, "wss://")
      .replace(/^http:\/\//, "ws://");
  }

  if (typeof window === "undefined") return "ws://127.0.0.1:7880";

  const { hostname, protocol } = window.location;
  const isLoopback = hostname === "localhost" || hostname === "127.0.0.1" || hostname === "::1";
  const host = isLoopback ? "127.0.0.1" : hostname;
  return `${protocol === "https:" ? "wss" : "ws"}://${host}:7880`;
}

interface Props {
  token: string;
  /** Students receive a subscribe-only token; opening their devices would fail. */
  canPublish: boolean;
  onLeave: () => void;
}

export function LiveRoom({ token, canPublish, onLeave }: Props) {
  const [error, setError] = useState<string | null>(null);

  return (
    <div className="space-y-3">
      {!canPublish && (
        <div className="flex items-start gap-2 rounded-md border bg-muted/40 p-3 text-sm text-muted-foreground">
          <Eye className="mt-0.5 h-4 w-4 shrink-0" />
          <span>
            You are watching this lecture. Your camera and microphone stay off, and you
            can still use chat.
          </span>
        </div>
      )}

      {error && (
        <div className="flex items-start gap-2 rounded-md border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
          <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
          <span>{error}</span>
        </div>
      )}

      <div className="overflow-hidden rounded-lg border" data-lk-theme="default">
        <LiveKitRoom
          token={token}
          serverUrl={liveKitServerUrl()}
          connect
          // Only request devices the token actually grants, so a student is not
          // prompted for camera access they cannot use.
          audio={canPublish}
          video={canPublish}
          options={{ adaptiveStream: true, dynacast: true }}
          onError={(e) => setError(e.message || "Could not connect to the live room")}
          onDisconnected={onLeave}
          style={{ height: "70vh" }}
        >
          <VideoConference />
        </LiveKitRoom>
      </div>
    </div>
  );
}
