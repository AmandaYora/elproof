import { todayISO } from "@/shared/lib/formatters";
import type { SubscriptionTransaction, Tenant } from "@/modules/platform-admin/data/types";

export type DashboardPeriod = "month" | "year";

export interface TrendPoint {
  key: string;
  label: string;
  value: number;
}

const MONTHS_ID = ["Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"];

function parseISODate(iso: string) {
  const [y, m, d] = iso.split("T")[0].split("-").map(Number);
  return { year: y, month: m, day: d };
}

// Points only run through today (month-to-date) or the current month (year-to-date) —
// showing future, not-yet-happened days/months as empty bars would be misleading.
function emptyPoints(period: DashboardPeriod): TrendPoint[] {
  const { month: currentMonth, day: currentDay } = parseISODate(todayISO());
  if (period === "month") {
    return Array.from({ length: currentDay }, (_, i) => ({ key: `d${i + 1}`, label: String(i + 1), value: 0 }));
  }
  return Array.from({ length: currentMonth }, (_, i) => ({ key: `m${i + 1}`, label: MONTHS_ID[i], value: 0 }));
}

function addToPoints(points: TrendPoint[], iso: string, period: DashboardPeriod, amount: number): void {
  const { year: currentYear, month: currentMonth, day: currentDay } = parseISODate(todayISO());
  const { year, month, day } = parseISODate(iso);
  if (year !== currentYear) return;

  if (period === "month") {
    if (month === currentMonth && day <= currentDay) {
      points[day - 1].value += amount;
    }
  } else if (month <= currentMonth) {
    points[month - 1].value += amount;
  }
}

export function buildRevenueTrend(transactions: SubscriptionTransaction[], period: DashboardPeriod): TrendPoint[] {
  const points = emptyPoints(period);
  for (const tx of transactions) {
    if (tx.status === "paid" && tx.paidAt) {
      addToPoints(points, tx.paidAt, period, tx.amount);
    }
  }
  return points;
}

export function buildTenantGrowthTrend(tenants: Tenant[], period: DashboardPeriod): TrendPoint[] {
  const points = emptyPoints(period);
  for (const tenant of tenants) {
    addToPoints(points, tenant.joinedAt, period, 1);
  }
  return points;
}
