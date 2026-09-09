import type { BinaryAssociation } from "./surveyAssociation";
export type SurveyOutcome = "degraded" | "banned" | "limited";
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
  rows: {
    label: string;
    degraded: BinaryAssociation;
    banned: BinaryAssociation;
    limited: BinaryAssociation;
  }[];
}
export interface SurveyStatistics {
  total: number;
  degraded: number;
  banned: number;
  limited: number;
  statuses: { label: string; count: number; tone: string }[];
  normal: number;
  both: number;
  factors: FactorDistribution[];
  associations: AssociationGroup[];
  outcomeAssociation: BinaryAssociation;
  usagePattern: { total: number; levels: number[] };
  range: {
    firstSubmissionId: number;
    lastSubmissionId: number;
    firstSubmittedAt: string | null;
    lastSubmittedAt: string | null;
    unknownTimeCount: number;
    computedAt: string;
  };
}
export interface SurveySubmission {
  status: string[];
  answers: Record<string, string[]>;
  details: Record<string, string>;
  usagePattern?: number[];
  ipRisk?: number;
}
export async function surveyRequest<T>(path: string, submission?: SurveySubmission): Promise<T> {
  const response = await fetch(`/api/survey/${path}`, {
    method: submission ? "POST" : "GET",
    headers: submission ? { "Content-Type": "application/json" } : undefined,
    body: submission ? JSON.stringify(submission) : undefined,
    cache: "no-store",
  });
  if (!response.ok) {
    if (response.status === 400) throw new Error("请检查填写内容后重试。");
    throw new Error("服务暂时不可用，请稍后重试。");
  }
  return response.json() as Promise<T>;
}
