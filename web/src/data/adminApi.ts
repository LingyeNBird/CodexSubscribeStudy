export interface AdminQuestion {
  key: string;
  title: string;
  description: string;
  multiple: boolean;
  eligibility?: string;
}

export interface AdminCatalog {
  choices: Record<string, string[]>;
  definitions: AdminQuestion[];
}

export interface AdminRecord {
  id: number;
  submittedAt: string | null;
  status: string[];
  answers: Record<string, string[]>;
  details: Record<string, string>;
  /** Values exactly as submitted, before statistics bucketing. */
  raw: Record<string, string[]>;
  /** The same answers mapped onto the questionnaire options used by statistics. */
  normalized: Record<string, string[]>;
  usagePattern?: number[];
  ipRisk?: number;
  legacyIPQuality?: unknown;
  /** Administrator-only labels. Never part of the submitted payload. */
  tags: string[];
  note: string;
}

export interface AdminRecords {
  total: number;
  returned: number;
  truncated: boolean;
  records: AdminRecord[];
}

/** Thrown when the deployment has no administrator credentials configured. */
export class AdminDisabled extends Error {}
/** Thrown when the session is missing or has expired. */
export class AdminUnauthorized extends Error {}

async function adminRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`/api/admin/${path}`, {
    credentials: "same-origin",
    cache: "no-store",
    ...init,
  });
  if (response.status === 404) throw new AdminDisabled("服务未启用管理面板。");
  if (response.status === 401) throw new AdminUnauthorized("登录状态已失效。");
  if (response.status === 429) throw new Error("尝试次数过多，请稍后再试。");
  if (response.status >= 400 && response.status < 500)
    throw new Error("请求参数有误，请检查后重试。");
  if (!response.ok) throw new Error("服务暂时不可用，请稍后重试。");
  return response.json() as Promise<T>;
}

export async function fetchAdminSession(): Promise<boolean> {
  const session = await adminRequest<{ authenticated: boolean }>("session");
  return session.authenticated;
}

export async function signInAdmin(
  username: string,
  password: string,
): Promise<void> {
  try {
    await adminRequest<{ authenticated: boolean }>("session", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password }),
    });
  } catch (error) {
    if (error instanceof AdminUnauthorized)
      throw new Error("用户名或密码错误。");
    throw error;
  }
}

export async function signOutAdmin(): Promise<void> {
  await adminRequest<{ authenticated: boolean }>("session", {
    method: "DELETE",
  });
}

export function fetchAdminCatalog(): Promise<AdminCatalog> {
  return adminRequest<AdminCatalog>("catalog");
}

export function fetchAdminRecords(): Promise<AdminRecords> {
  return adminRequest<AdminRecords>("submissions");
}

/** Fields that can be changed; `ids` selects the submissions to change. */
export interface AnnotationPatch {
  addTags?: string[];
  removeTags?: string[];
  note?: string;
}

export interface AnnotationChange extends AnnotationPatch {
  ids: number[];
}

/** Applies one annotation change to one or many submissions. */
export async function annotate(
  change: AnnotationChange,
): Promise<{ updated: number }> {
  return adminRequest<{ updated: number }>("annotations", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(change),
  });
}
