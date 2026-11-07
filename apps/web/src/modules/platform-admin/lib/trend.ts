export type DashboardPeriod = "month" | "year";

/** One bar of a dashboard chart; the API buckets them (GET /platform/trends). */
export interface TrendPoint {
  key: string;
  label: string;
  value: number;
}
