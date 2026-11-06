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
import {
  ENGAGEMENT_STATUS_LABELS,
  ISSUE_IMPACT_LABELS,
  ISSUE_STATUS_LABELS,
  MILESTONE_STATUS_LABELS,
  PROJECT_STATUS_LABELS,
  labelFor,
  type Audience,
} from "@/shared/lib/labels";

// Every badge reads its wording from shared/lib/labels.ts. `audience`
// "client" switches to the Client Portal wording and renders nothing for
// states the client is not shown.

export function ProjectStatusBadge({ status, audience = "staff" }: { status: ProjectStatus; audience?: Audience }) {
  const label = labelFor(PROJECT_STATUS_LABELS, status, audience);
  return label ? <Badge tone={PROJECT_STATUS_TONE[status]}>{label}</Badge> : null;
}

export function EngagementStatusBadge({ status, audience = "staff" }: { status: EngagementStatus; audience?: Audience }) {
  const label = labelFor(ENGAGEMENT_STATUS_LABELS, status, audience);
  return label ? <Badge tone={ENGAGEMENT_STATUS_TONE[status]}>{label}</Badge> : null;
}

export function MilestoneStatusBadge({ status, audience = "staff" }: { status: MilestoneStatus; audience?: Audience }) {
  const label = labelFor(MILESTONE_STATUS_LABELS, status, audience);
  // A blocked step reads "Sedang berjalan" for clients, so it takes the same tone.
  const tone = audience === "client" && status === "Blocked" ? MILESTONE_STATUS_TONE["In Progress"] : MILESTONE_STATUS_TONE[status];
  return label ? <Badge tone={tone}>{label}</Badge> : null;
}

export function IssueImpactBadge({ impact, audience = "staff" }: { impact: IssueImpact; audience?: Audience }) {
  const label = labelFor(ISSUE_IMPACT_LABELS, impact, audience);
  return label ? <Badge tone={ISSUE_IMPACT_TONE[impact]}>{label}</Badge> : null;
}

export function IssueStatusBadge({ status, audience = "staff" }: { status: IssueStatus; audience?: Audience }) {
  const label = labelFor(ISSUE_STATUS_LABELS, status, audience);
  return label ? <Badge tone={ISSUE_STATUS_TONE[status]}>{label}</Badge> : null;
}

export function ConditionBadge({ condition }: { condition: ProjectCondition }) {
  return <Badge tone={CONDITION_TONE[condition]}>{CONDITION_LABEL[condition]}</Badge>;
}
