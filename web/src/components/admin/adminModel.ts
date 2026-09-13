import type {
  AdminCatalog,
  AdminQuestion,
  AdminRecord,
} from "../../data/adminApi";

/** Sentinel used for "this question was not answered". */
export const UNAVAILABLE = "__none__";
/** Sentinel used for submissions without a server-recorded time. */
export const TIME_UNKNOWN = "__unknown__";

export const presetTags = ["可疑", "待跟进", "已排除", "高质量样本"];

export const statusLabels = ["正常", "降智", "封号", "风控（限流）"];

export interface Dimension {
  /** "question:<key>" for questionnaire items, or one of the derived keys. */
  id: string;
  title: string;
  multiple: boolean;
}

export const STATUS_DIMENSION = "derived:status";
export const MONTH_DIMENSION = "derived:month";

export function listDimensions(catalog: AdminCatalog | null): Dimension[] {
  const questions = (catalog?.definitions ?? []).map((question) => ({
    id: `question:${question.key}`,
    title: question.title,
    multiple: question.multiple,
  }));
  return [
    { id: STATUS_DIMENSION, title: "异常状态", multiple: true },
    { id: MONTH_DIMENSION, title: "提交月份", multiple: false },
    ...questions,
  ];
}

function questionKeyOf(dimension: string) {
  return dimension.startsWith("question:") ? dimension.slice(9) : null;
}

// ---------------------------------------------------------------------------
// Display
// ---------------------------------------------------------------------------

const regionNames = (() => {
  try {
    return new Intl.DisplayNames(["zh-CN"], { type: "region" });
  } catch {
    return null;
  }
})();

const regionKeys = new Set(["country", "exitCountry"]);

/** Region answers are stored as ISO alpha-2 codes; nobody should read those. */
export function displayValue(key: string, value: string) {
  if (value === UNAVAILABLE) return "未回答";
  if (!regionKeys.has(key)) return value;
  if (value === "unknown") return "我不知道";
  try {
    return regionNames?.of(value) ?? value;
  } catch {
    return value;
  }
}

/** Statistics fold regions and numeric answers into buckets; only numeric
 * grouping is worth annotating, since region folding has no analytical value. */
export function bucketHint(key: string, display: string, bucket: string) {
  if (!bucket || regionKeys.has(key) || bucket === display) return "";
  return bucket;
}

const timeUnits: Record<string, number> = {
  小时: 1 / 24,
  天: 1,
  星期: 7,
  月: 30,
  年: 365,
};

/** Numeric magnitude of a submitted value. Time units are converted to days so
 * that "2 年" is not ordered as the bare number 2 against "34 天". */
export function magnitude(value: string): number | null {
  const match = /^\s*(-?[0-9]+(?:\.[0-9]+)?)\s*(\S*)\s*$/.exec(value);
  if (!match) return null;
  const amount = Number.parseFloat(match[1]);
  if (!Number.isFinite(amount)) return null;
  const unit = match[2];
  if (!unit) return amount;
  const scale = timeUnits[unit];
  // An unknown unit would make the comparison meaningless, so it is not one.
  return scale === undefined ? null : amount * scale;
}

export type SortKind = "number" | "text";

/** Decides how a whole column is ordered. Deciding per column keeps numbers and
 * words from being compared against each other inside one column. A column with
 * a numeric majority is numeric even when some answers are labels such as
 * "unknown"; a genuine text column has no numbers at all to begin with. */
export function columnKind(values: string[]): SortKind {
  const present = values.filter((value) => value !== "");
  if (!present.length) return "text";
  const numeric = present.filter((value) => magnitude(value) !== null).length;
  return numeric / present.length >= 0.5 ? "number" : "text";
}

export function compareCells(a: string, b: string, kind: SortKind) {
  if (kind === "text") return a.localeCompare(b, "zh-CN");
  const left = magnitude(a);
  const right = magnitude(b);
  // "unknown" and friends go to one end rather than being folded into the
  // numeric scale as zero.
  if (left === null && right === null) return a.localeCompare(b, "zh-CN");
  if (left === null) return 1;
  if (right === null) return -1;
  return left - right || a.localeCompare(b, "zh-CN");
}

export function valuesOf(record: AdminRecord, dimension: string): string[] {
  if (dimension === STATUS_DIMENSION) return record.status;
  if (dimension === MONTH_DIMENSION) {
    return [record.submittedAt ? record.submittedAt.slice(0, 7) : TIME_UNKNOWN];
  }
  const key = questionKeyOf(dimension);
  if (!key) return [];
  const values = record.raw[key] ?? [];
  return values.length ? values : [UNAVAILABLE];
}

export function displayOf(dimension: string, value: string) {
  if (value === TIME_UNKNOWN) return "时间未知";
  if (dimension === STATUS_DIMENSION || dimension === MONTH_DIMENSION)
    return value;
  return displayValue(questionKeyOf(dimension) ?? "", value);
}

export function groupTitle(record: AdminRecord, key: string, value: string) {
  return bucketHint(
    key,
    displayValue(key, value),
    (record.normalized[key] ?? []).join("、"),
  );
}

// ---------------------------------------------------------------------------
// Filters
// ---------------------------------------------------------------------------

export interface FilterClause {
  dimension: string;
  options: string[];
  mode: "include" | "exclude";
}

export interface AdminFilters {
  statuses: string[];
  from: string;
  to: string;
  search: string;
  tags: string[];
  clauses: FilterClause[];
}

export function emptyFilters(): AdminFilters {
  return { statuses: [], from: "", to: "", search: "", tags: [], clauses: [] };
}

/** Filters are plain data, so a JSON round trip is the predictable copy;
 * `structuredClone` rejects the reactive proxy Vue hands back. */
export function cloneFilters(source: AdminFilters): AdminFilters {
  return JSON.parse(JSON.stringify(source)) as AdminFilters;
}

export function activeFilterCount(filters: AdminFilters) {
  return (
    filters.statuses.length +
    filters.tags.length +
    filters.clauses.length +
    (filters.from || filters.to ? 1 : 0) +
    (filters.search.trim() ? 1 : 0)
  );
}

function dayOf(record: AdminRecord) {
  return record.submittedAt ? record.submittedAt.slice(0, 10) : "";
}

function searchTargets(record: AdminRecord) {
  const parts: string[] = [
    ...record.status,
    ...record.tags,
    record.note,
    ...Object.values(record.details),
  ];
  for (const values of Object.values(record.raw)) parts.push(...values);
  return parts.join("\n").toLowerCase();
}

export function matchesFilters(
  record: AdminRecord,
  filters: AdminFilters,
): boolean {
  if (
    filters.statuses.length &&
    !record.status.some((status) => filters.statuses.includes(status))
  ) {
    return false;
  }
  if (
    filters.tags.length &&
    !record.tags.some((tag) => filters.tags.includes(tag))
  )
    return false;
  if (filters.from || filters.to) {
    const day = dayOf(record);
    // Submissions without a recorded time cannot be placed on a timeline, so a
    // date range excludes them instead of silently including them.
    if (!day) return false;
    if (filters.from && day < filters.from) return false;
    if (filters.to && day > filters.to) return false;
  }
  const search = filters.search.trim().toLowerCase();
  if (search && !searchTargets(record).includes(search)) return false;
  for (const clause of filters.clauses) {
    const values = valuesOf(record, clause.dimension);
    const hit = values.some((value) => clause.options.includes(value));
    if (clause.mode === "include" && !hit) return false;
    if (clause.mode === "exclude" && hit) return false;
  }
  return true;
}

export function applyFilters(records: AdminRecord[], filters: AdminFilters) {
  return records.filter((record) => matchesFilters(record, filters));
}

// ---------------------------------------------------------------------------
// Option counts
// ---------------------------------------------------------------------------

export interface OptionRow {
  value: string;
  display: string;
  count: number;
  hint: string;
}

export function optionsFor(
  records: AdminRecord[],
  dimension: string,
  catalog: AdminCatalog | null,
): OptionRow[] {
  const key = questionKeyOf(dimension);
  const choices = key ? (catalog?.choices[key] ?? []) : [];
  const counts = new Map<string, number>();
  const buckets = new Map<string, Set<string>>();
  for (const record of records) {
    const values = valuesOf(record, dimension);
    const normalized = key ? (record.normalized[key] ?? []) : [];
    values.forEach((value, index) => {
      counts.set(value, (counts.get(value) ?? 0) + 1);
      const bucket = normalized[index];
      if (bucket && bucket !== value) {
        const set = buckets.get(value) ?? new Set<string>();
        set.add(bucket);
        buckets.set(value, set);
      }
    });
  }
  const observed = [...counts.keys()];
  // Questions answered from the option list keep every option visible; free
  // values only list what was actually submitted, ordered by frequency.
  const aligned =
    key !== null && observed.every((value) => choices.includes(value));
  const kind = columnKind(observed);
  const ordered = aligned
    ? [...choices, ...observed.filter((value) => !choices.includes(value))]
    : observed.sort(
        (a, b) =>
          (counts.get(b) ?? 0) - (counts.get(a) ?? 0) ||
          compareCells(a, b, kind),
      );
  return ordered.map((value) => {
    const display = displayOf(dimension, value);
    return {
      value,
      display,
      count: counts.get(value) ?? 0,
      hint: key
        ? bucketHint(key, display, [...(buckets.get(value) ?? [])].join("、"))
        : "",
    };
  });
}

export function knownTags(records: AdminRecord[]) {
  const found = new Set<string>(presetTags);
  for (const record of records) for (const tag of record.tags) found.add(tag);
  return [...found].sort((a, b) => a.localeCompare(b, "zh-CN"));
}

// ---------------------------------------------------------------------------
// Statistics
// ---------------------------------------------------------------------------

function phi(a: number, b: number, c: number, d: number): number | null {
  const denominator = Math.sqrt((a + b) * (c + d) * (a + c) * (b + d));
  return denominator ? (a * d - b * c) / denominator : null;
}

export interface CrossTable {
  rowValues: string[];
  colValues: string[];
  counts: number[][];
  rowTotals: number[];
  colTotals: number[];
  /** Submissions that entered the table. */
  total: number;
  /** Sum of the margins; exceeds `total` when a dimension repeats a record. */
  grand: number;
  /** True when either dimension can match a record more than once, so the
   * margins can exceed the number of selected submissions. */
  repeats: boolean;
}

export function crossTab(
  records: AdminRecord[],
  rowDimension: string,
  colDimension: string,
  catalog: AdminCatalog | null,
): CrossTable {
  const rowValues: string[] = [];
  const colValues: string[] = [];
  const index = (list: string[], value: string) => {
    const found = list.indexOf(value);
    if (found >= 0) return found;
    list.push(value);
    return list.length - 1;
  };
  const counts: number[][] = [];
  for (const record of records) {
    for (const row of valuesOf(record, rowDimension)) {
      const rowIndex = index(rowValues, row);
      counts[rowIndex] ??= [];
      for (const col of valuesOf(record, colDimension)) {
        const colIndex = index(colValues, col);
        counts[rowIndex][colIndex] = (counts[rowIndex][colIndex] ?? 0) + 1;
      }
    }
  }
  for (const row of counts)
    for (let i = 0; i < colValues.length; i++) row[i] ??= 0;
  const rowTotals = counts.map((row) =>
    row.reduce((sum, value) => sum + value, 0),
  );
  const colTotals = colValues.map((_, i) =>
    counts.reduce((sum, row) => sum + (row[i] ?? 0), 0),
  );
  const repeats =
    multiValued(rowDimension, catalog) || multiValued(colDimension, catalog);
  return {
    rowValues,
    colValues,
    counts,
    rowTotals,
    colTotals,
    total: records.length,
    grand: colTotals.reduce((sum, value) => sum + value, 0),
    repeats,
  };
}

/** A record can land in more than one cell when a dimension is multi-valued;
 * the margins then legitimately exceed the number of selected submissions. */
function multiValued(dimension: string, catalog: AdminCatalog | null) {
  if (dimension === STATUS_DIMENSION) return true;
  if (!dimension.startsWith("question:")) return false;
  const key = dimension.slice(9);
  return (
    catalog?.definitions.find((question) => question.key === key)?.multiple ??
    false
  );
}

export interface CohortRow {
  dimension: string;
  title: string;
  value: string;
  display: string;
  hint: string;
  aCount: number;
  aTotal: number;
  bCount: number;
  bTotal: number;
  aShare: number;
  bShare: number;
  diff: number;
  phi: number | null;
}

export function cohortRows(
  groupA: AdminRecord[],
  groupB: AdminRecord[],
  catalog: AdminCatalog | null,
): { title: string; rows: CohortRow[] }[] {
  const inA = new Set(groupA.map((record) => record.id));
  const inB = new Set(groupB.map((record) => record.id));
  const union = [...groupA, ...groupB.filter((record) => !inA.has(record.id))];
  return (catalog?.definitions ?? []).map((question: AdminQuestion) => {
    const rows = optionsFor(union, `question:${question.key}`, catalog).map(
      (option) => {
        let aCount = 0;
        let bCount = 0;
        let aTotal = 0;
        let bTotal = 0;
        for (const record of union) {
          const values = valuesOf(record, `question:${question.key}`);
          // Records that skipped the question are excluded from both sides.
          if (values.length === 1 && values[0] === UNAVAILABLE) continue;
          const hit = values.includes(option.value);
          // A record can belong to both cohorts when B is a custom filter
          // rather than A's complement, so membership is checked independently.
          if (inA.has(record.id)) {
            aTotal++;
            if (hit) aCount++;
          }
          if (inB.has(record.id)) {
            bTotal++;
            if (hit) bCount++;
          }
        }
        const aShare = aTotal ? aCount / aTotal : 0;
        const bShare = bTotal ? bCount / bTotal : 0;
        return {
          dimension: question.key,
          title: question.title,
          value: option.value,
          display: option.display,
          hint: option.hint,
          aCount,
          aTotal,
          bCount,
          bTotal,
          aShare,
          bShare,
          diff: 100 * (aShare - bShare),
          phi: phi(aCount, aTotal - aCount, bCount, bTotal - bCount),
        };
      },
    );
    return { title: question.title, rows };
  });
}

export interface TrendBucket {
  key: string;
  total: number;
  degraded: number;
  banned: number;
  limited: number;
}

function bucketKey(record: AdminRecord, granularity: "day" | "week" | "month") {
  const day = dayOf(record);
  if (!day) return TIME_UNKNOWN;
  if (granularity === "month") return day.slice(0, 7);
  if (granularity === "day") return day;
  const date = new Date(`${day}T00:00:00Z`);
  const weekday = (date.getUTCDay() + 6) % 7;
  date.setUTCDate(date.getUTCDate() - weekday);
  return date.toISOString().slice(0, 10);
}

export function trend(
  records: AdminRecord[],
  granularity: "day" | "week" | "month",
): { buckets: TrendBucket[]; unknown: TrendBucket | null } {
  const found = new Map<string, TrendBucket>();
  for (const record of records) {
    const key = bucketKey(record, granularity);
    const bucket = found.get(key) ?? {
      key,
      total: 0,
      degraded: 0,
      banned: 0,
      limited: 0,
    };
    bucket.total++;
    if (record.status.includes("降智")) bucket.degraded++;
    if (record.status.includes("封号")) bucket.banned++;
    if (record.status.includes("风控（限流）")) bucket.limited++;
    found.set(key, bucket);
  }
  const unknown = found.get(TIME_UNKNOWN) ?? null;
  found.delete(TIME_UNKNOWN);
  return {
    buckets: [...found.values()].sort((a, b) => a.key.localeCompare(b.key)),
    unknown,
  };
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

function csvCell(value: string) {
  return /[",\n\r]/.test(value) ? `"${value.replace(/"/g, '""')}"` : value;
}

export function toCsv(
  records: AdminRecord[],
  catalog: AdminCatalog | null,
  options: { annotations: boolean },
): string {
  const questions = catalog?.definitions ?? [];
  const header = ["编号", "提交时间", "异常状态"];
  for (const question of questions) header.push(question.title);
  header.push("补充字段");
  if (options.annotations) header.push("标签", "备注");
  const lines = [header.map(csvCell).join(",")];
  for (const record of records) {
    const row = [
      String(record.id),
      record.submittedAt ?? "",
      record.status.join(" | "),
    ];
    for (const question of questions) {
      row.push(
        (record.raw[question.key] ?? [])
          .map((value) => displayValue(question.key, value))
          .join(" | "),
      );
    }
    row.push(
      Object.entries(record.details)
        .map(([key, value]) => `${key}=${value}`)
        .join("; "),
    );
    if (options.annotations) row.push(record.tags.join(" | "), record.note);
    lines.push(row.map(csvCell).join(","));
  }
  // The byte order mark keeps Excel from mangling Chinese column headers.
  return `\ufeff${lines.join("\r\n")}\r\n`;
}

export function toJson(
  records: AdminRecord[],
  filters: AdminFilters,
  catalog: AdminCatalog | null,
): string {
  return JSON.stringify(
    {
      exportedAt: new Date().toISOString(),
      questionCount: catalog?.definitions.length ?? 0,
      filters,
      count: records.length,
      records,
    },
    null,
    2,
  );
}

export function download(name: string, content: string, type: string) {
  const url = URL.createObjectURL(new Blob([content], { type }));
  const link = document.createElement("a");
  link.href = url;
  link.download = name;
  link.click();
  URL.revokeObjectURL(url);
}

// ---------------------------------------------------------------------------
// Shareable state
// ---------------------------------------------------------------------------

export interface PanelState {
  filters: AdminFilters;
  tab: string;
  cohorts?: {
    a: AdminFilters;
    b: { mode: "complement" } | { mode: "custom"; filters: AdminFilters };
  };
  rows?: string;
}

export function encodeState(state: PanelState) {
  const bytes = new TextEncoder().encode(JSON.stringify(state));
  let binary = "";
  for (const byte of bytes) binary += String.fromCharCode(byte);
  return btoa(binary)
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
}

export function decodeState(value: string): PanelState | null {
  try {
    const normalized = value.replace(/-/g, "+").replace(/_/g, "/");
    const binary = atob(
      normalized + "=".repeat((4 - (normalized.length % 4)) % 4),
    );
    const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0));
    const parsed = JSON.parse(new TextDecoder().decode(bytes)) as PanelState;
    return parsed?.filters ? parsed : null;
  } catch {
    return null;
  }
}
