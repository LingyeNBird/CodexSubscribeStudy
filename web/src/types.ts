export interface Cause {
  id: string;
  label: string;
  support: number | null;
  factor_estimates: number[];
}
export interface Parameter {
  name: string;
  mean: number;
  low: number;
  high: number;
}
export interface Study {
  id: string;
  title: string;
  state: "no_data" | "uninformative" | "conditional" | "sensitive";
  method: string;
  method_digest: string;
  updated_at: string;
  confidence_meaning: string;
  totals: {
    contributors: number;
    batches: number;
    requests: number;
    gpt6_requests: number;
    other_requests: number;
    raw_usd: number;
    gpt6_raw_usd: number;
    quota_points: number;
    intervals: number;
    contrasts: number;
  };
  causes: Cause[];
  parameters: Parameter[];
  drift_support: number[][];
  gpt6_quota: Parameter | null;
  information_rank: number;
  maximum_source_information_share: number;
  quality: Record<string, number>;
  warnings: string[];
}
