import { ref } from "vue";

const storageKey = "chatgpt-account-survey:submitted";
export const submittedBefore = ref(false);

export function loadSubmissionMarker() {
  try {
    submittedBefore.value =
      submittedBefore.value || window.localStorage.getItem(storageKey) === "1";
  } catch {
    // Storage restrictions do not invalidate a submission completed in this session.
  }
}

export function markSurveySubmitted() {
  submittedBefore.value = true;
  try {
    window.localStorage.setItem(storageKey, "1");
  } catch {
    // Keep the in-memory acknowledgment when browser storage is unavailable.
  }
}
