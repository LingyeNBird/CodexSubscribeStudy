export interface Cause {
  id: string; label: string; support: number | null;
  score_mean: number | null; score_low: number | null; score_high: number | null;
  factor_estimates: number[];
}
export interface Study {
  id: string; title: string; state: string; method: string; method_digest: string;
  updated_at: string; window_days: number; minimum_contributors: number; confidence_meaning: string;
  totals: { contributors: number; eligible_contributors: number; requests: number; gpt6_requests: number;
    baseline_requests: number; raw_usd: number; gpt6_raw_usd: number; quota_points: number; cycles: number; blocks: number; };
  causes: Cause[]; quality: Record<string, number>; identifiable_sites: number[];
}
