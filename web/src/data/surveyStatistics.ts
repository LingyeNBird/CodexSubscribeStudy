import type { StatusCounts, SurveyFactorSummary, SurveyOutcome, SurveySummary } from "./surveyApi";
import type { BinaryAssociation } from "./surveyAssociation";

export const surveyOutcomes: { id: SurveyOutcome; label: string; bit: number }[] = [
  { id: "degraded", label: "降智", bit: 1 },
  { id: "banned", label: "封号", bit: 2 },
  { id: "limited", label: "风控（限流）", bit: 4 },
];
export interface FactorDistribution {
  id: string;
  title: string;
  description: string;
  multiple: boolean;
  eligibility: string;
  groups: Record<SurveyOutcome, { total: number; rows: { label: string; count: number }[] }>;
}
export interface AssociationGroup {
  id: string;
  title: string;
  scope: string;
  total: number;
  rows: { label: string; association: BinaryAssociation }[];
}
export interface SurveyStatistics {
  total: number;
  degraded: number;
  banned: number;
  limited: number;
  normal: number;
  statuses: { label: string; count: number; tone: string }[];
  factors: FactorDistribution[];
  usagePattern: { total: number; levels: number[] };
  range: SurveySummary["range"];
}

export function countStatuses(counts: StatusCounts, mask?: number): number {
  let total = 0;
  for (let state = 0; state < 8; state++) {
    if (mask === undefined || (state & mask) !== 0) total += counts[state]!;
  }
  return total;
}

export function effectiveOutcomeMask(
  selected: readonly SurveyOutcome[],
  lastSelected: SurveyOutcome,
): number {
  let mask = 0;
  for (const item of surveyOutcomes) if (selected.includes(item.id)) mask |= item.bit;
  return mask || surveyOutcomes.find((item) => item.id === lastSelected)!.bit;
}

export function calculateAssociation(
  selected: StatusCounts,
  applicable: StatusCounts,
  mask: number,
): BinaryAssociation {
  const selectedTotal = countStatuses(selected);
  const total = countStatuses(applicable);
  const a = countStatuses(selected, mask);
  const b = selectedTotal - a;
  const c = countStatuses(applicable, mask) - a;
  const d = total - a - b - c;
  const denominator = Math.sqrt((a + b) * (c + d) * (a + c) * (b + d));
  return {
    selected: { events: a, total: selectedTotal },
    unselected: { events: c, total: total - selectedTotal },
    phi: denominator > 0 ? (a * d - b * c) / denominator : null,
  };
}

export function calculateAssociations(
  factors: SurveyFactorSummary[],
  mask: number,
): AssociationGroup[] {
  return factors
    .filter((factor) => factor.comparable)
    .map((factor) => ({
      id: factor.id,
      title: factor.title,
      scope: factor.associationScope,
      total: countStatuses(factor.applicable),
      rows: factor.options.map((option) => ({
        label: option.label,
        association: calculateAssociation(option.counts, factor.applicable, mask),
      })),
    }));
}

const statusLabels = [
  "正常",
  "降智",
  "封号",
  "降智、封号",
  "风控（限流）",
  "降智、风控（限流）",
  "封号、风控（限流）",
  "降智、封号、风控（限流）",
];
const statusTones = ["mint", "violet", "peach", "peach", "sun", "sun", "sun", "sun"];

export function calculateSurveyStatistics(summary: SurveySummary): SurveyStatistics {
  const factors = summary.factors.map((factor): FactorDistribution => {
    const groups = {} as FactorDistribution["groups"];
    for (const outcome of surveyOutcomes) {
      groups[outcome.id] = {
        total: countStatuses(factor.applicable, outcome.bit),
        rows: factor.options.map((option) => ({
          label: option.label,
          count: countStatuses(option.counts, outcome.bit),
        })),
      };
    }
    return {
      id: factor.id,
      title: factor.title,
      description: factor.description,
      multiple: factor.multiple,
      eligibility: factor.distributionScope,
      groups,
    };
  });
  return {
    total: countStatuses(summary.statuses),
    degraded: countStatuses(summary.statuses, 1),
    banned: countStatuses(summary.statuses, 2),
    limited: countStatuses(summary.statuses, 4),
    normal: summary.statuses[0],
    statuses: summary.statuses.map((count, mask) => ({
      label: statusLabels[mask]!,
      tone: statusTones[mask]!,
      count,
    })),
    factors,
    usagePattern: {
      total: summary.usagePattern.total,
      levels: summary.usagePattern.sums.map((sum) =>
        summary.usagePattern.total ? sum / summary.usagePattern.total : 0,
      ),
    },
    range: summary.range,
  };
}
