export type SurveyOutcome = "degraded" | "banned" | "limited";
export type StatusCounts = [number, number, number, number, number, number, number, number];
export interface SurveyFactorSummary {
  id: string;
  title: string;
  description: string;
  multiple: boolean;
  distributionScope: string;
  associationScope: string;
  comparable: boolean;
  applicable: StatusCounts;
  options: { label: string; counts: StatusCounts }[];
}
export interface SurveySummary {
  version: 2;
  catalogDigest: string;
  statuses: StatusCounts;
  factors: SurveyFactorSummary[];
  usagePattern: { total: number; sums: number[] };
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
