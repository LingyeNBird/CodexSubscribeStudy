<script setup lang="ts">
import {
  computed,
  onBeforeUnmount,
  onMounted,
  ref,
  shallowRef,
  watch,
} from "vue";
import type { SurveyOutcome, SurveySummary } from "../../data/surveyApi";
import {
  effectiveOutcomeMask,
  surveyOutcomes,
} from "../../data/surveyStatistics";
import {
  buildSurveyPoster,
  posterToPng,
  type PosterDetail,
  type SurveyPoster,
} from "../../data/surveySharePoster";
import claudeIcon from "../icons/ClaudeCodeIcon.vue?raw";
import ompIcon from "../icons/OhMyPiIcon.vue?raw";
import openCodeIcon from "../icons/OpenCodeIcon.vue?raw";
import cursorIcon from "../icons/CursorIcon.vue?raw";
import clineIcon from "../icons/ClineIcon.vue?raw";
import rooIcon from "../icons/RooCodeIcon.vue?raw";
import aiderIcon from "../icons/AiderIcon.vue?raw";

const props = defineProps<{ summary: SurveySummary; outcomeMask: number }>();
const emit = defineEmits<{ close: [] }>();
const dialog = ref<HTMLDialogElement | null>(null);
const overview = ref(true);
const factorIds = ref<string[]>([]);
const poster = shallowRef<SurveyPoster | null>(null);
const previewUrl = ref("");
const error = ref("");
const saving = ref(false);
const backdropPressed = ref(false);
const selectedOutcomes = ref<SurveyOutcome[]>(
  surveyOutcomes
    .filter((item) => item.bit & props.outcomeMask)
    .map((item) => item.id),
);
const lastSelectedOutcome = ref<SurveyOutcome>(
  selectedOutcomes.value.at(-1) ?? "degraded",
);
const outcomeMask = computed(() =>
  effectiveOutcomeMask(selectedOutcomes.value, lastSelectedOutcome.value),
);
const detail = ref<PosterDetail>("default");
const detailOptions: { value: PosterDetail; label: string }[] = [
  { value: "default", label: "默认" },
  { value: "coefficient", label: "稍详细" },
  { value: "full", label: "更详细" },
];
function toggleOutcome(id: SurveyOutcome) {
  if (selectedOutcomes.value.includes(id)) {
    selectedOutcomes.value = selectedOutcomes.value.filter(
      (value) => value !== id,
    );
  } else {
    lastSelectedOutcome.value = id;
    selectedOutcomes.value = surveyOutcomes
      .filter(
        (item) => item.id === id || selectedOutcomes.value.includes(item.id),
      )
      .map((item) => item.id);
  }
}
const hasSelection = computed(
  () => overview.value || factorIds.value.length > 0,
);
const icons = {
  "Claude Code": claudeIcon,
  "oh-my-pi": ompIcon,
  OpenCode: openCodeIcon,
  Cursor: cursorIcon,
  Cline: clineIcon,
  "Roo Code": rooIcon,
  Aider: aiderIcon,
};
let disposed = false;
let measureContext: CanvasRenderingContext2D | null = null;
let downloadUrl = "";

function regenerate() {
  error.value = "";
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value);
  previewUrl.value = "";
  poster.value = null;
  if (!hasSelection.value) return;
  try {
    const measure = measureContext
      ? (text: string, size: number, weight: number) => {
          measureContext!.font = `${weight} ${size}px Inter, "Noto Sans SC", "Microsoft YaHei", sans-serif`;
          return measureContext!.measureText(text).width;
        }
      : undefined;
    const result = buildSurveyPoster(
      props.summary,
      {
        overview: overview.value,
        factorIds: factorIds.value,
        outcomeMask: outcomeMask.value,
        detail: detail.value,
      },
      icons,
      measure,
    );
    poster.value = result;
    previewUrl.value = URL.createObjectURL(
      new Blob([result.svg], { type: "image/svg+xml;charset=utf-8" }),
    );
  } catch (cause) {
    error.value =
      cause instanceof Error ? cause.message : "海报生成失败，请重试。";
  }
}

async function download() {
  if (!poster.value || saving.value) return;
  saving.value = true;
  error.value = "";
  try {
    const png = await posterToPng(poster.value);
    if (disposed) return;
    if (downloadUrl) URL.revokeObjectURL(downloadUrl);
    downloadUrl = URL.createObjectURL(png);
    const link = document.createElement("a");
    link.href = downloadUrl;
    link.download = `共研-问卷统计-${props.summary.range.computedAt.slice(0, 10)}.png`;
    document.body.append(link);
    link.click();
    link.remove();
  } catch (cause) {
    if (!disposed)
      error.value =
        cause instanceof Error ? cause.message : "图片导出失败，请重试。";
  } finally {
    saving.value = false;
  }
}

watch(
  [overview, factorIds, outcomeMask, detail, () => props.summary],
  regenerate,
);
onMounted(async () => {
  dialog.value?.showModal();
  measureContext = document.createElement("canvas").getContext("2d");
  await document.fonts?.ready;
  if (!disposed) regenerate();
});
onBeforeUnmount(() => {
  disposed = true;
  if (dialog.value?.open) dialog.value.close();
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value);
  if (downloadUrl) URL.revokeObjectURL(downloadUrl);
});
</script>

<template>
  <Teleport to="body">
    <dialog
      ref="dialog"
      class="share-dialog"
      aria-labelledby="share-heading"
      @close="emit('close')"
      @pointerdown="backdropPressed = $event.target === dialog"
      @click.self="backdropPressed && dialog?.close()"
    >
      <header class="share-heading">
        <div>
          <h2 id="share-heading">把这份统计，分享出去。</h2>
        </div>
        <fieldset
          class="share-outcome-picker"
          :disabled="saving"
          aria-label="关联视角"
        >
          <label
            v-for="outcome in surveyOutcomes"
            :key="outcome.id"
            :class="{ selected: selectedOutcomes.includes(outcome.id) }"
          >
            <input
              type="checkbox"
              :checked="selectedOutcomes.includes(outcome.id)"
              @change="toggleOutcome(outcome.id)"
            />{{ outcome.label }}
          </label>
        </fieldset>
        <button
          type="button"
          class="share-close"
          aria-label="关闭分享"
          autofocus
          @click="dialog?.close()"
        >
          关闭 <span aria-hidden="true">×</span>
        </button>
      </header>
      <div class="share-body">
        <aside class="share-options">
          <fieldset :disabled="saving">
            <legend>选择分享内容</legend>
            <label class="overview-choice" :class="{ selected: overview }"
              ><input v-model="overview" type="checkbox" /><span
                ><strong>问卷统计</strong></span
              ></label
            >
            <fieldset class="share-detail">
              <legend>详细程度</legend>
              <div class="share-detail-options">
                <label
                  v-for="option in detailOptions"
                  :key="option.value"
                  :class="{ selected: detail === option.value }"
                >
                  <input
                    v-model="detail"
                    type="radio"
                    name="share-detail"
                    :value="option.value"
                  />
                  {{ option.label }}
                </label>
              </div>
            </fieldset>
            <div class="share-factor-heading">
              <strong>具体因素</strong
              ><button
                type="button"
                @click="
                  factorIds =
                    factorIds.length === summary.factors.length
                      ? []
                      : summary.factors.map((factor) => factor.id)
                "
              >
                {{
                  factorIds.length === summary.factors.length
                    ? "清空因素"
                    : "全选因素"
                }}
              </button>
            </div>
            <div class="share-factor-list">
              <label
                v-for="factor in summary.factors"
                :key="factor.id"
                :class="{ selected: factorIds.includes(factor.id) }"
                ><input
                  v-model="factorIds"
                  type="checkbox"
                  :value="factor.id"
                /><span>{{ factor.title }}</span></label
              >
            </div>
          </fieldset>
        </aside>
        <section
          class="share-preview"
          aria-label="海报预览"
          :aria-busy="saving"
        >
          <img
            v-if="previewUrl && poster"
            :src="previewUrl"
            :width="poster.width"
            :height="poster.height"
            alt="所选问卷统计的分享海报预览"
          />
          <p v-else class="share-empty">
            {{
              error || (hasSelection ? "正在准备海报…" : "请勾选要分享的内容。")
            }}
          </p>
        </section>
      </div>
      <footer class="share-footer">
        <div>
          <strong v-if="poster"
            >{{ poster.width }} × {{ poster.height }} PNG</strong
          >
          <p v-if="error" role="alert">{{ error }}</p>
        </div>
        <button
          class="button primary"
          type="button"
          :disabled="!poster || saving"
          @click="download"
        >
          {{ saving ? "正在导出…" : "下载分享图片 ↓" }}
        </button>
      </footer>
    </dialog>
  </Teleport>
</template>

<style scoped src="./SurveyShareDialog.css"></style>
