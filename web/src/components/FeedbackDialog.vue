<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from "vue";
import { feedbackContentLimit, submitFeedback } from "../data/feedbackApi";

const emit = defineEmits<{ close: [] }>();
const dialog = ref<HTMLDialogElement | null>(null);
const textarea = ref<HTMLTextAreaElement | null>(null);
const content = ref("");
const phase = ref<"editing" | "sending" | "success" | "error">("editing");
const message = ref("");
const backdropPressed = ref(false);
let dismissTimer: ReturnType<typeof setTimeout> | undefined;

async function confirm() {
  const trimmed = content.value.trim();
  if (!trimmed || phase.value === "sending") return;
  phase.value = "sending";
  message.value = "";
  try {
    await submitFeedback(trimmed);
    phase.value = "success";
    dismissTimer = setTimeout(() => dialog.value?.close(), 2000);
  } catch (error) {
    phase.value = "error";
    message.value =
      error instanceof Error ? error.message : "提交失败，请稍后重试。";
  }
}

function cancel() {
  dialog.value?.close();
}

onMounted(async () => {
  dialog.value?.showModal();
  await nextTick();
  textarea.value?.focus();
});
onBeforeUnmount(() => {
  if (dismissTimer) clearTimeout(dismissTimer);
});
</script>

<template>
  <Teleport to="body">
    <dialog
      ref="dialog"
      class="feedback-dialog"
      aria-labelledby="feedback-heading"
      @close="emit('close')"
      @pointerdown="backdropPressed = $event.target === dialog"
      @click.self="backdropPressed && dialog?.close()"
    >
      <h2 id="feedback-heading">意见反馈</h2>
      <p v-if="phase === 'success'" class="feedback-status" role="status">
        发送成功
      </p>
      <template v-else>
        <textarea
          ref="textarea"
          v-model="content"
          :maxlength="feedbackContentLimit"
          :disabled="phase === 'sending'"
          rows="6"
          placeholder="欢迎反馈问题、建议或使用中遇到的情况…"
        ></textarea>
        <div class="feedback-meta">
          <span>{{ content.length }} / {{ feedbackContentLimit }}</span>
          <span v-if="phase === 'error'" role="alert" class="feedback-error">{{
            message
          }}</span>
        </div>
        <div class="feedback-actions">
          <button
            type="button"
            :disabled="phase === 'sending'"
            @click="cancel"
          >
            取消
          </button>
          <button
            type="button"
            class="primary"
            :disabled="phase === 'sending' || !content.trim()"
            @click="confirm"
          >
            {{ phase === "sending" ? "发送中" : "确认" }}
          </button>
        </div>
      </template>
    </dialog>
  </Teleport>
</template>

<style scoped src="./FeedbackDialog.css"></style>
