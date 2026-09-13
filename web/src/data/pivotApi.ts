import type { AdminCatalog, AdminRecord, AdminRecords } from "./adminApi";

/** The public pivot feed. It serves the same submission data as the admin
 * panel — the study never collected identifying fields, so there is nothing
 * to protect there — except investigator tags and notes, which the backend
 * never includes in this response at all. */
async function pivotRequest<T>(path: string): Promise<T> {
  const response = await fetch(`/api/pivot/${path}`, {
    credentials: "omit",
    cache: "no-store",
  });
  if (response.status === 429) throw new Error("请求过于频繁，请稍后再试。");
  if (!response.ok) throw new Error("服务暂时不可用，请稍后重试。");
  return response.json() as Promise<T>;
}

export function fetchPivotCatalog(): Promise<AdminCatalog> {
  return pivotRequest<AdminCatalog>("catalog");
}

type PublicRecord = Omit<AdminRecord, "tags" | "note">;
type PublicRecords = Omit<AdminRecords, "records"> & {
  records: PublicRecord[];
};

/** Decorates each record with the empty tags/note the admin-only components
 * expect, so the pivot views can be reused verbatim on the public panel. */
export async function fetchPivotRecords(): Promise<AdminRecords> {
  const payload = await pivotRequest<PublicRecords>("submissions");
  return {
    ...payload,
    records: payload.records.map((record) => ({
      ...record,
      tags: [],
      note: "",
    })),
  };
}
