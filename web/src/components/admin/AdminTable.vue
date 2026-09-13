<script setup lang="ts">
import { computed, ref } from "vue";
import type { AdminCatalog, AdminRecord } from "../../data/adminApi";
import {
  bucketHint,
  columnKind,
  displayValue,
  magnitude,
  type SortKind,
} from "./adminModel";

const props = defineProps<{
  records: AdminRecord[];
  catalog: AdminCatalog | null;
  columns: string[];
  density: "compact" | "cozy";
  sort: { key: string; desc: boolean };
  selected: number[];
}>();

const emit = defineEmits<{
  open: [id: number];
  "update:columns": [columns: string[]];
  "update:density": [density: "compact" | "cozy"];
  "update:sort": [sort: { key: string; desc: boolean }];
  "update:selected": [ids: number[]];
}>();

const settingsOpen = ref(false);

const baseColumns = [
  { key: "id", title: "编号" },
  { key: "submittedAt", title: "提交时间" },
  { key: "status", title: "异常状态" },
  { key: "tags", title: "标签" },
  { key: "note", title: "备注" },
];

const questionColumns = computed(() =>
  (props.catalog?.definitions ?? []).map((question) => ({
    key: `question:${question.key}`,
    title: question.title,
  })),
);

const columnTitle = computed(() => {
  const map = new Map<string, string>();
  for (const column of [...baseColumns, ...questionColumns.value])
    map.set(column.key, column.title);
  return map;
});

function cellValues(record: AdminRecord, key: string) {
  const question = key.startsWith("question:") ? key.slice(9) : "";
  if (!question) return [];
  return (record.raw[question] ?? []).map((value) => ({
    text: displayValue(question, value),
    hint: bucketHint(
      question,
      displayValue(question, value),
      (record.normalized[question] ?? []).join("、"),
    ),
  }));
}

function formatTime(value: string | null) {
  if (!value) return "时间未知";
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value;
  return parsed.toLocaleString("zh-CN", { hour12: false });
}

const textColumns = new Set(["submittedAt", "status", "tags", "note"]);

/** How each visible column is ordered. Deciding per column stops a numeric
 * comparison in one cell from deciding the order of a whole column. */
const columnKinds = computed(() => {
  const kinds = new Map<string, SortKind>();
  for (const key of props.columns) {
    if (key === "id") kinds.set(key, "number");
    else if (textColumns.has(key)) kinds.set(key, "text");
    else
      kinds.set(
        key,
        columnKind(props.records.map((record) => rawFirst(record, key))),
      );
  }
  return kinds;
});

function rawFirst(record: AdminRecord, key: string) {
  return (record.raw[key.slice(9)] ?? [])[0] ?? "";
}

function cellText(record: AdminRecord, key: string) {
  if (key === "id") return String(record.id);
  if (key === "submittedAt") return record.submittedAt ?? "";
  if (key === "status") return record.status.join("、");
  if (key === "tags") return record.tags.join("、");
  if (key === "note") return record.note;
  const question = key.slice(9);
  const raw = rawFirst(record, key);
  return raw ? displayValue(question, raw) : "";
}

/** Blank cells and values without a number ("unknown") stay at the end in both
 * directions, so reversing the order never promotes them to the top. */
const rows = computed(() => {
  const key = props.sort.key;
  const kind = columnKinds.value.get(key) ?? "text";
  return [...props.records].sort((left, right) => {
    const leftText = cellText(left, key);
    const rightText = cellText(right, key);
    if ((leftText === "") !== (rightText === ""))
      return leftText === "" ? 1 : -1;
    if (key === "id") {
      const order = left.id - right.id;
      return props.sort.desc ? -order : order;
    }
    if (kind === "number") {
      const leftValue = magnitude(leftText);
      const rightValue = magnitude(rightText);
      if ((leftValue === null) !== (rightValue === null))
        return leftValue === null ? 1 : -1;
      if (leftValue !== null && rightValue !== null) {
        const order =
          leftValue - rightValue || leftText.localeCompare(rightText, "zh-CN");
        return props.sort.desc ? -order : order;
      }
    }
    const order = leftText.localeCompare(rightText, "zh-CN");
    return props.sort.desc ? -order : order;
  });
});

function toggleSort(key: string) {
  emit("update:sort", {
    key,
    // A fresh column reads naturally from small to large; clicking again flips.
    desc: props.sort.key === key ? !props.sort.desc : false,
  });
}

function toggleColumn(key: string) {
  emit(
    "update:columns",
    props.columns.includes(key)
      ? props.columns.filter((item) => item !== key)
      : [...props.columns, key],
  );
}

const allSelected = computed(
  () =>
    rows.value.length > 0 &&
    rows.value.every((row) => props.selected.includes(row.id)),
);

function toggleAll() {
  emit(
    "update:selected",
    allSelected.value ? [] : rows.value.map((row) => row.id),
  );
}

function toggleRow(id: number) {
  emit(
    "update:selected",
    props.selected.includes(id)
      ? props.selected.filter((item) => item !== id)
      : [...props.selected, id],
  );
}
</script>

<template>
  <div class="table-wrap">
    <div class="table-bar">
      <div class="table-cols">
        <button
          type="button"
          class="table-btn"
          @click="settingsOpen = !settingsOpen"
        >
          列（{{ columns.length }}）
        </button>
        <button
          type="button"
          class="table-btn"
          @click="
            emit('update:density', density === 'compact' ? 'cozy' : 'compact')
          "
        >
          {{ density === "compact" ? "紧凑" : "宽松" }}
        </button>
      </div>
      <p class="table-note">点任意一行打开详情；点表头排序。</p>
      <div v-if="settingsOpen" class="table-settings">
        <div
          v-for="column in baseColumns"
          :key="column.key"
          class="table-setting"
        >
          <label
            ><input
              type="checkbox"
              :checked="columns.includes(column.key)"
              @change="toggleColumn(column.key)"
            />{{ column.title }}</label
          >
        </div>
        <p class="table-note">题目列</p>
        <div
          v-for="column in questionColumns"
          :key="column.key"
          class="table-setting"
        >
          <label
            ><input
              type="checkbox"
              :checked="columns.includes(column.key)"
              @change="toggleColumn(column.key)"
            />{{ column.title }}</label
          >
        </div>
      </div>
    </div>

    <div class="table-scroll">
      <table :class="`table is-${density}`">
        <thead>
          <tr>
            <th class="table-check">
              <input
                type="checkbox"
                :checked="allSelected"
                aria-label="全选当前结果"
                @change="toggleAll"
              />
            </th>
            <th v-for="key in columns" :key="key">
              <button type="button" class="table-sort" @click="toggleSort(key)">
                {{ columnTitle.get(key) ?? key }}
                <span v-if="sort.key === key" aria-hidden="true">{{
                  sort.desc ? "▼" : "▲"
                }}</span>
              </button>
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="record in rows" :key="record.id">
            <td class="table-check">
              <input
                type="checkbox"
                :checked="selected.includes(record.id)"
                :aria-label="`选择第 ${record.id} 份问卷`"
                @change="toggleRow(record.id)"
              />
            </td>
            <td
              v-for="key in columns"
              :key="key"
              @click="emit('open', record.id)"
            >
              <template v-if="key === 'id'">{{ record.id }}</template>
              <template v-else-if="key === 'submittedAt'">{{
                formatTime(record.submittedAt)
              }}</template>
              <template v-else-if="key === 'status'">
                <span
                  v-for="status in record.status"
                  :key="status"
                  class="table-pill"
                  >{{ status }}</span
                >
              </template>
              <template v-else-if="key === 'tags'">
                <span
                  v-for="tag in record.tags"
                  :key="tag"
                  class="table-pill is-tag"
                  >{{ tag }}</span
                >
              </template>
              <template v-else-if="key === 'note'">
                <span class="table-note-cell">{{ record.note }}</span>
              </template>
              <template v-else-if="cellValues(record, key).length">
                <span
                  v-for="(cell, index) in cellValues(record, key)"
                  :key="index"
                >
                  {{ cell.text
                  }}<em v-if="cell.hint" class="table-hint">{{ cell.hint }}</em>
                </span>
              </template>
              <template v-else>
                <span class="table-empty">未回答</span>
              </template>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <p v-if="!rows.length" class="table-none">当前条件下没有问卷。</p>
  </div>
</template>

<style scoped src="./AdminTable.css"></style>
