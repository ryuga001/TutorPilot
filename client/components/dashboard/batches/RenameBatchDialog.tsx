"use client";

import { useState, type FormEvent } from "react";
import { Loader2, Pencil } from "lucide-react";
import { toast } from "sonner";
import type { FetchBaseQueryError } from "@reduxjs/toolkit/query";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { useUpdateBatchMutation } from "@/lib/api/batchesApi";
import { apiErrorMessage } from "@/lib/api-error";

export function RenameBatchDialog({ id, currentName }: { id: number; currentName: string }) {
  const [open, setOpen] = useState(false);
  const [name, setName] = useState(currentName);
  const [updateBatch, { isLoading }] = useUpdateBatchMutation();

  async function submit(e: FormEvent) {
    e.preventDefault();
    try {
      await updateBatch({ id, body: { name } }).unwrap();
      toast.success("Batch renamed");
      setOpen(false);
    } catch (err) {
      toast.error(apiErrorMessage(err as FetchBaseQueryError));
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        setOpen(o);
        if (o) setName(currentName);
      }}
    >
      <DialogTrigger asChild>
        <Button variant="ghost" size="icon" className="h-8 w-8" aria-label="Rename batch">
          <Pencil className="h-4 w-4" />
        </Button>
      </DialogTrigger>
      <DialogContent>
        <form onSubmit={submit}>
          <DialogHeader>
            <DialogTitle>Rename batch</DialogTitle>
          </DialogHeader>
          <div className="py-4">
            <Input value={name} onChange={(e) => setName(e.target.value)} required />
          </div>
          <DialogFooter>
            <Button type="submit" disabled={isLoading || !name.trim()}>
              {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Save
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
