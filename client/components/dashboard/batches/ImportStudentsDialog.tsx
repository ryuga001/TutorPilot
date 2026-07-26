"use client";

import { useState, type ChangeEvent } from "react";
import { AlertTriangle, CheckCircle2, Loader2, Upload } from "lucide-react";
import { toast } from "sonner";
import type { FetchBaseQueryError } from "@reduxjs/toolkit/query";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { useImportBatchStudentsMutation } from "@/lib/api/batchesApi";
import { apiErrorMessage } from "@/lib/api-error";
import type { ImportResult } from "@/lib/types";

export function ImportStudentsDialog({ batchId }: { batchId: number }) {
  const [open, setOpen] = useState(false);
  const [file, setFile] = useState<File | null>(null);
  const [result, setResult] = useState<ImportResult | null>(null);
  const [importStudents, { isLoading }] = useImportBatchStudentsMutation();

  function reset() {
    setFile(null);
    setResult(null);
  }

  function onFile(e: ChangeEvent<HTMLInputElement>) {
    setFile(e.target.files?.[0] ?? null);
  }

  async function submit() {
    if (!file) return;
    try {
      const res = await importStudents({ batchId, file }).unwrap();
      setResult(res);
      if (res.skipped.length === 0) {
        toast.success(`${res.imported} student${res.imported === 1 ? "" : "s"} imported`);
      }
    } catch (err) {
      toast.error(apiErrorMessage(err as FetchBaseQueryError));
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        setOpen(o);
        if (!o) reset();
      }}
    >
      <DialogTrigger asChild>
        <Button size="sm">
          <Upload className="mr-2 h-4 w-4" />
          Import CSV
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Import students</DialogTitle>
          <DialogDescription>
            CSV with columns <code>first_name</code>, <code>last_name</code>,{" "}
            <code>email</code>, <code>phone</code> (any order). Students are
            matched by email — re-uploading updates existing records.
          </DialogDescription>
        </DialogHeader>

        {!result ? (
          <div className="space-y-4 py-4">
            <input
              type="file"
              accept=".csv,text/csv"
              onChange={onFile}
              className="block w-full text-sm text-muted-foreground file:mr-4 file:border file:border-input file:bg-background file:px-3 file:py-2 file:text-sm file:font-medium"
            />
          </div>
        ) : (
          <div className="space-y-3 py-4">
            <div className="flex items-center gap-2 text-sm">
              <CheckCircle2 className="h-4 w-4 text-primary" />
              {result.imported} student{result.imported === 1 ? "" : "s"} imported
              successfully.
            </div>
            {result.skipped.length > 0 && (
              <div className="space-y-2">
                <div className="flex items-center gap-2 text-sm text-destructive">
                  <AlertTriangle className="h-4 w-4" />
                  {result.skipped.length} row{result.skipped.length === 1 ? "" : "s"} skipped
                </div>
                <div className="max-h-48 overflow-y-auto border">
                  <table className="w-full text-sm">
                    <tbody className="divide-y">
                      {result.skipped.map((s) => (
                        <tr key={s.row}>
                          <td className="px-3 py-1.5 text-muted-foreground">Row {s.row}</td>
                          <td className="px-3 py-1.5">{s.reason}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </div>
        )}

        <DialogFooter>
          {!result ? (
            <Button onClick={submit} disabled={!file || isLoading}>
              {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Upload &amp; import
            </Button>
          ) : (
            <Button onClick={() => setOpen(false)}>Done</Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
