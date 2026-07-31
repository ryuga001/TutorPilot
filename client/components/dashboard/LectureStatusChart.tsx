"use client";

import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

export interface LectureStatusChartProps {
  data: { status: string; count: number }[];
}

export function LectureStatusChart({ data }: LectureStatusChartProps) {
  return (
    <ResponsiveContainer width="100%" height="100%">
      <BarChart data={data}>
        <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
        <XAxis
          dataKey="status"
          tickLine={false}
          axisLine={false}
          className="fill-muted-foreground text-xs"
        />
        <YAxis
          allowDecimals={false}
          tickLine={false}
          axisLine={false}
          className="fill-muted-foreground text-xs"
        />
        <Tooltip
          cursor={{ fill: "hsl(var(--accent))" }}
          contentStyle={{
            borderRadius: "var(--radius)",
            border: "1px solid hsl(var(--border))",
            background: "hsl(var(--popover))",
            color: "hsl(var(--popover-foreground))",
          }}
        />
        <Bar dataKey="count" fill="hsl(var(--primary))" radius={[6, 6, 0, 0]} />
      </BarChart>
    </ResponsiveContainer>
  );
}

export default LectureStatusChart;
