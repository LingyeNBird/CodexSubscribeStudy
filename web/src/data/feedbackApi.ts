const CLIENT_ID_KEY = "feedback-client-id";
const CLIENT_ID_ALPHABET = "abcdefghijklmnopqrstuvwxyz0123456789";

function randomClientId(length = 24): string {
  const bytes = new Uint8Array(length);
  crypto.getRandomValues(bytes);
  let id = "";
  for (const byte of bytes) id += CLIENT_ID_ALPHABET[byte % CLIENT_ID_ALPHABET.length];
  return id;
}

/** Persisted per browser so repeat feedback from the same visitor can be
 * associated upstream; not identifying on its own, and never sent anywhere
 * except this app's own feedback endpoint. */
function clientId(): string {
  try {
    const stored = localStorage.getItem(CLIENT_ID_KEY);
    if (stored) return stored;
    const fresh = randomClientId();
    localStorage.setItem(CLIENT_ID_KEY, fresh);
    return fresh;
  } catch {
    return randomClientId();
  }
}

/** The app's own UX limit. The upstream API's hard cap is 1000 characters. */
export const feedbackContentLimit = 800;

export interface FeedbackResult {
  /** true when the server queued this for a later retry instead of sending
   * it immediately, because another submission used the shared cooldown
   * window first. Either way the feedback is accepted. */
  queued: boolean;
}

export async function submitFeedback(content: string): Promise<FeedbackResult> {
  const response = await fetch("/api/feedback", {
    method: "POST",
    credentials: "omit",
    cache: "no-store",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ content, clientId: clientId() }),
  });
  if (!response.ok) throw new Error("反馈提交失败，请稍后重试。");
  const payload = (await response.json()) as { queued?: boolean };
  return { queued: Boolean(payload.queued) };
}
