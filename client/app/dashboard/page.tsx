"use client";

import { useMemo } from "react";
import dynamic from "next/dynamic";
import { CalendarDays, GraduationCap, Users, Video } from "lucide-react";

import { StatCard } from "@/components/layout/StatCard";
import { PageHeader } from "@/components/layout/PageHeader";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

// recharts is a sizeable library that only matters once the KPI tiles above
// have already painted, so it is split out of the dashboard's initial JS.
const LectureStatusChart = dynamic(
  () => import("@/components/dashboard/LectureStatusChart").then((m) => m.LectureStatusChart),
  { ssr: false, loading: () => <div className="h-full w-full animate-pulse rounded-md bg-muted" /> },
);
import { useDashboard } from "@/lib/dashboard-context";
import { useCan } from "@/lib/hooks/useCan";
import { useListBatchesQuery } from "@/lib/api/batchesApi";
import { useListLecturesQuery } from "@/lib/api/lecturesApi";
import { useListStudentsQuery } from "@/lib/api/studentsApi";
import { useListTutorsQuery } from "@/lib/api/tutorsApi";

const LECTURE_STATUSES = ["scheduled", "live", "ended", "cancelled"] as const;
const STATUS_LABEL: Record<(typeof LECTURE_STATUSES)[number], string> = {
  scheduled: "Scheduled",
  live: "Live",
  ended: "Ended",
  cancelled: "Cancelled",
};

export default function DashboardPage() {
  const { me } = useDashboard();
  const can = useCan();

  const canBatches = can("batch.view");
  const canTutors = can("tutor.view");
  const canStudents = can("student.view");
  const canLectures = can("lecture.view");

  const { data: batches, isLoading: batchesLoading } = useListBatchesQuery(
    { page: 1, page_size: 1 },
    { skip: !canBatches },
  );
  const { data: tutors, isLoading: tutorsLoading } = useListTutorsQuery(
    { page: 1, page_size: 1 },
    { skip: !canTutors },
  );
  const { data: students, isLoading: studentsLoading } = useListStudentsQuery(
    { page: 1, page_size: 1 },
    { skip: !canStudents },
  );

  const scheduled = useListLecturesQuery(
    { page: 1, page_size: 1, status: "scheduled" },
    { skip: !canLectures },
  );
  const live = useListLecturesQuery(
    { page: 1, page_size: 1, status: "live" },
    { skip: !canLectures },
  );
  const ended = useListLecturesQuery(
    { page: 1, page_size: 1, status: "ended" },
    { skip: !canLectures },
  );
  const cancelled = useListLecturesQuery(
    { page: 1, page_size: 1, status: "cancelled" },
    { skip: !canLectures },
  );

  const lectureStatusLoading =
    scheduled.isLoading || live.isLoading || ended.isLoading || cancelled.isLoading;

  const chartData = useMemo(
    () => [
      { status: STATUS_LABEL.scheduled, count: scheduled.data?.total ?? 0 },
      { status: STATUS_LABEL.live, count: live.data?.total ?? 0 },
      { status: STATUS_LABEL.ended, count: ended.data?.total ?? 0 },
      { status: STATUS_LABEL.cancelled, count: cancelled.data?.total ?? 0 },
    ],
    [scheduled.data, live.data, ended.data, cancelled.data],
  );

  const firstName = me?.first_name?.trim();
  const greetingName = firstName || me?.email?.split("@")[0];

  return (
    <div className="space-y-8">
      <PageHeader
        title={`Welcome back${greetingName ? `, ${greetingName}` : ""}`}
        subtitle="Here's what's happening across your organization today."
      />

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {canBatches && (
          <StatCard
            label="Active batches"
            value={batches?.total ?? 0}
            icon={CalendarDays}
            tone="primary"
            isLoading={batchesLoading}
          />
        )}
        {canLectures && (
          <StatCard
            label="Upcoming lectures"
            value={scheduled.data?.total ?? 0}
            icon={Video}
            tone="secondary"
            isLoading={scheduled.isLoading}
          />
        )}
        {canTutors && (
          <StatCard
            label="Tutors"
            value={tutors?.total ?? 0}
            icon={Users}
            tone="success"
            isLoading={tutorsLoading}
          />
        )}
        {canStudents && (
          <StatCard
            label="Students"
            value={students?.total ?? 0}
            icon={GraduationCap}
            tone="primary"
            isLoading={studentsLoading}
          />
        )}
      </div>

      {canLectures && (
        <Card className="animate-in fade-in slide-in-from-bottom-1 duration-300">
          <CardHeader>
            <CardTitle className="text-lg">Lecture status breakdown</CardTitle>
          </CardHeader>
          <CardContent>
            {lectureStatusLoading ? (
              <div className="h-64 animate-pulse rounded-md bg-muted" />
            ) : (
              <div className="h-64 w-full">
                <LectureStatusChart data={chartData} />
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
