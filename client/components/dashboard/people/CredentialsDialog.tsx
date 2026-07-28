"use client";

import { useState } from "react";
import { Check, Copy, KeyRound } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

/**
 * Shows the temporary password once, right after a tutor or student is created.
 *
 * The login is created atomically with the record and the server returns this
 * password exactly once, never on a read path — if it is lost the only recovery
 * is deleting and recreating the record, so the dialog makes it hard to miss.
 */
export function CredentialsDialog({
  tempPassword,
  email,
  personName,
  onClose,
}: {
  tempPassword: string | null;
  email?: string;
  personName?: string;
  onClose: () => void;
}) {
  const [copied, setCopied] = useState(false);

  async function copy() {
    if (!tempPassword) return;
    try {
      await navigator.clipboard.writeText(tempPassword);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Clipboard access can be refused; the password is on screen regardless.
    }
  }

  return (
    <Dialog
      open={Boolean(tempPassword)}
      onOpenChange={(open) => {
        if (!open) {
          setCopied(false);
          onClose();
        }
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <div className="mb-2 flex h-9 w-9 items-center justify-center rounded-full bg-primary/10">
            <KeyRound className="h-4 w-4 text-primary" />
          </div>
          <DialogTitle>Sign-in details created</DialogTitle>
          <DialogDescription>
            Share these with {personName || "them"} so they can sign in.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          {email && (
            <div className="rounded-md border bg-muted/40 p-3 text-sm">
              <div className="text-xs uppercase tracking-wide text-muted-foreground">
                Email
              </div>
              <div className="mt-0.5 font-medium">{email}</div>
            </div>
          )}

          <div className="rounded-md border bg-muted/40 p-3">
            <div className="text-xs uppercase tracking-wide text-muted-foreground">
              Temporary password
            </div>
            <div className="mt-1 flex items-center gap-2">
              <code className="flex-1 break-all rounded bg-background px-2 py-1 font-mono text-sm">
                {tempPassword}
              </code>
              <Button variant="outline" size="icon" className="h-8 w-8" onClick={copy}>
                {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                <span className="sr-only">Copy password</span>
              </Button>
            </div>
          </div>

          <p className="text-sm text-muted-foreground">
            This is shown only now and cannot be retrieved later.
          </p>
        </div>

        <DialogFooter>
          <Button onClick={onClose}>Done</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
