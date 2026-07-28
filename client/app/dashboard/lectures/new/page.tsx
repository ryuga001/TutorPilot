import { redirect } from "next/navigation";

/**
 * Scheduling now happens in a dialog on the lectures page, so there is no separate
 * create screen. The route is kept as a redirect for anything still linking here.
 */
export default function NewLecturePage() {
  redirect("/dashboard/lectures");
}
