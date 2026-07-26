import {
  File,
  FileArchive,
  FileAudio,
  FileImage,
  FileSpreadsheet,
  FileText,
  FileVideo,
  Folder,
  type LucideIcon,
} from "lucide-react";

/** Picks a representative icon for a file's content type. */
export function getFileIcon(contentType?: string): LucideIcon {
  const ct = contentType ?? "";
  if (ct.startsWith("image/")) return FileImage;
  if (ct === "application/pdf") return FileText;
  if (ct.includes("sheet") || ct.includes("excel") || ct.includes("csv")) return FileSpreadsheet;
  if (ct.includes("zip") || ct.includes("compressed") || ct.includes("archive")) return FileArchive;
  if (ct.startsWith("video/")) return FileVideo;
  if (ct.startsWith("audio/")) return FileAudio;
  return File;
}

export { Folder };

export function formatBytes(bytes?: number) {
  if (!bytes) return "";
  const units = ["B", "KB", "MB", "GB"];
  let n = bytes;
  let i = 0;
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024;
    i++;
  }
  return `${n.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}
