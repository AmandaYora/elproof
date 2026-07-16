import { Badge } from "@/shared/components/ui/Badge";
import type {
  ProjectStatus,
  EngagementStatus,
  MilestoneStatus,
  IssueImpact,
  IssueStatus,
  ProjectCondition,
} from "@/modules/projects/types";
import {
  PROJECT_STATUS_TONE,
  ENGAGEMENT_STATUS_TONE,
  MILESTONE_STATUS_TONE,
  ISSUE_IMPACT_TONE,
  ISSUE_STATUS_TONE,
  CONDITION_TONE,
  CONDITION_LABEL,
} from "@/modules/projects/lib/status";

export function ProjectStatusBadge({ status }: { status: ProjectStatus }) {
  return <Badge tone={PROJECT_STATUS_TONE[status]}>{status}</Badge>;
}

export function EngagementStatusBadge({ status }: { status: EngagementStatus }) {
  return <Badge tone={ENGAGEMENT_STATUS_TONE[status]}>{status}</Badge>;
}

export function MilestoneStatusBadge({ status }: { status: MilestoneStatus }) {
  return <Badge tone={MILESTONE_STATUS_TONE[status]}>{status}</Badge>;
}

export function IssueImpactBadge({ impact }: { impact: IssueImpact }) {
  return <Badge tone={ISSUE_IMPACT_TONE[impact]}>{impact}</Badge>;
}

export function IssueStatusBadge({ status }: { status: IssueStatus }) {
  return <Badge tone={ISSUE_STATUS_TONE[status]}>{status}</Badge>;
}

export function ConditionBadge({ condition }: { condition: ProjectCondition }) {
  return <Badge tone={CONDITION_TONE[condition]}>{CONDITION_LABEL[condition]}</Badge>;
}
