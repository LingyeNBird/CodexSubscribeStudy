<script setup lang="ts">
import { computed, ref, watch } from "vue";
import type { AdminCatalog, AdminRecord } from "../../data/adminApi";
import { bucketHint, displayValue } from "./adminModel";

const props = defineProps<{
  record: AdminRecord;
  catalog: AdminCatalog | null;
  tagOptions: string[];
  hasPrev: boolean;
  hasNext: boolean;
  saving: boolean;
}>();

const emit = defineEmits<{
  close: [];
  move: [direction: -1 | 1];
  save: [change: { addTags?: string[]; removeTags?: string[]; note?: string }];
}>();

const draft = ref("");
const note = ref("");
const adding = ref("");

watch(
  () => props.record.id,
  () => {
    note.value = props.record.note;
    adding.value = "";
    draft.value = "";
  },
  { immediate: true },
);

const noteDirty = computed(() => note.value !== props.record.note);

const answers = computed(() =>
  (props.catalog?.definitions ?? [])
    .map((question) => {
      const values = props.record.raw[question.key] ?? [];
      const display = values.map((value) => displayValue(question.key, value));
      return {
        key: question.key,
        title: question.title,
        values: display,
        bucket: bucketHint(
          question.key,
          display.join("、"),
          (props.record.normalized[question.key] ?? []).join("、"),
        ),
      };
    })
    .filter((entry) => entry.values.length),
);

const detailLabels: Record<string, string> = {
  country: "账号地区代码",
  exitCountry: "出口地区代码",
  duration: "存活时长",
  durationUnit: "时长单位",
  people: "分发人数",
  concurrency: "最高并发",
};

const stored = computed(() =>
  Object.entries(props.record.details).map(([key, value]) => ({
    key,
    label: key.startsWith("toolMode:")
      ? `${key.slice(9)} 连接方式`
      : detailLabels[key]
        ? `${key} · ${detailLabels[key]}`
        : key,
    value,
  })),
);

function formatTime(value: string | null) {
  if (!value) return "时间未知";
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value;
  return parsed.toLocaleString("zh-CN", { hour12: false });
}

function addTag(tag: string) {
  const value = tag.trim();
  if (!value || props.record.tags.includes(value)) {
    adding.value = "";
    return;
  }
  emit("save", { addTags: [value] });
  adding.value = "";
}

function removeTag(tag: string) {
  emit("save", { removeTags: [tag] });
}

function saveNote() {
  emit("save", { note: note.value });
}

function addFromInput() {
  const value = draft.value
    .split(/[,，\s]+/)
    .map((item) => item.trim())
    .filter(Boolean);
  if (!value.length) return;
  emit("save", { addTags: value });
  draft.value = "";
}
</script>

<template>
  <aside class="drawer" aria-label="问卷详情">
    <header class="drawer-head">
      <div>
        <strong>第 {{ record.id }} 份</strong>
        <small>{{ formatTime(record.submittedAt) }}</small>
      </div>
      <div class="drawer-nav">
        <button
          type="button"
          :disabled="!hasPrev"
          aria-label="上一份"
          @click="emit('move', -1)"
        >
          ↑
        </button>
        <button
          type="button"
          :disabled="!hasNext"
          aria-label="下一份"
          @click="emit('move', 1)"
        >
          ↓
        </button>
        <button type="button" aria-label="关闭详情" @click="emit('close')">
          ×
        </button>
      </div>
    </header>

    <div class="drawer-body">
      <section class="drawer-block">
        <h3>异常状态</h3>
        <div class="drawer-chips">
          <span
            v-for="status in record.status"
            :key="status"
            class="drawer-pill"
            >{{ status }}</span
          >
        </div>
      </section>

      <section class="drawer-block">
        <h3>标签</h3>
        <div class="drawer-chips">
          <span
            v-for="tag in record.tags"
            :key="tag"
            class="drawer-pill is-tag"
          >
            {{ tag }}
            <button
              type="button"
              :aria-label="`移除标签${tag}`"
              @click="removeTag(tag)"
            >
              ×
            </button>
          </span>
          <select
            class="drawer-add"
            :value="adding"
            :disabled="saving"
            aria-label="添加标签"
            @change="addTag(($event.target as HTMLSelectElement).value)"
          >
            <option value="">+ 标签</option>
            <option
              v-for="tag in tagOptions"
              :key="tag"
              :value="tag"
              :disabled="record.tags.includes(tag)"
            >
              {{ tag }}
            </option>
          </select>
        </div>
        <div class="drawer-new">
          <input
            v-model="draft"
            type="text"
            placeholder="自定义标签，可逗号分隔"
            :disabled="saving"
            @keydown.enter.prevent="addFromInput"
          />
          <button
            type="button"
            class="drawer-btn"
            :disabled="saving || !draft.trim()"
            @click="addFromInput"
          >
            添加
          </button>
        </div>
      </section>

      <section class="drawer-block">
        <h3>备注</h3>
        <textarea
          v-model="note"
          rows="3"
          placeholder="记录你对这份问卷的判断"
          :disabled="saving"
        ></textarea>
        <div class="drawer-actions">
          <button
            type="button"
            class="drawer-btn is-primary"
            :disabled="saving || !noteDirty"
            @click="saveNote"
          >
            {{ noteDirty ? "保存备注" : "已保存" }}
          </button>
          <button
            v-if="noteDirty"
            type="button"
            class="drawer-btn"
            @click="note = record.note"
          >
            撤销
          </button>
        </div>
      </section>

      <section class="drawer-block">
        <h3>全部答案</h3>
        <dl class="drawer-answers">
          <div v-for="entry in answers" :key="entry.key">
            <dt>{{ entry.title }}</dt>
            <dd>
              {{ entry.values.join("、")
              }}<em v-if="entry.bucket">统计口径：{{ entry.bucket }}</em>
            </dd>
          </div>
        </dl>
        <p v-if="!answers.length" class="drawer-note">没有可归类的答案。</p>
      </section>

      <details class="drawer-block">
        <summary>数据库存储字段</summary>
        <dl class="drawer-answers">
          <div v-for="entry in stored" :key="entry.key">
            <dt>{{ entry.label }}</dt>
            <dd>{{ entry.value }}</dd>
          </div>
        </dl>
        <p v-if="!stored.length" class="drawer-note">没有补充字段。</p>
      </details>

      <details v-if="record.usagePattern" class="drawer-block">
        <summary>使用规律（24 小时网格）</summary>
        <p class="drawer-grid">
          <span
            v-for="(level, hour) in record.usagePattern"
            :key="hour"
            :style="{ opacity: 0.15 + (level / 24) * 0.85 }"
            >{{ level }}</span
          >
        </p>
      </details>

      <details class="drawer-block">
        <summary>原始 JSON</summary>
        <pre class="drawer-json">{{ JSON.stringify(record, null, 2) }}</pre>
      </details>
    </div>
  </aside>
</template>

<style scoped src="./AdminDrawer.css"></style>
