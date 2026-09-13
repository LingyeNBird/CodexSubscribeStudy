<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import {
  fetchPivotCatalog,
  fetchPivotRecords,
} from "../data/pivotApi";
import type { AdminCatalog, AdminRecord } from "../data/adminApi";
import AdminFiltersPanel from "../components/admin/AdminFilters.vue";
import AdminTable from "../components/admin/AdminTable.vue";
import AdminCross from "../components/admin/AdminCross.vue";
import AdminCohort from "../components/admin/AdminCohort.vue";
import AdminTimeline from "../components/admin/AdminTimeline.vue";
import AdminDrawer from "../components/admin/AdminDrawer.vue";
import {
  activeFilterCount,
  applyFilters,
  cloneFilters,
  decodeState,
  displayOf,
  download,
  emptyFilters,
  encodeState,
  toCsv,
  toJson,
  type AdminFilters,
} from "../components/admin/adminModel";

const state = ref<"loading" | "ready" | "error">("loading");
const notice = ref("");
const busy = ref(false);
const catalog = ref<AdminCatalog | null>(null);
const records = ref<AdminRecord[]>([]);

const mode = ref<"filter" | "cohort-a" | "cohort-b">("filter");
const filters = ref<AdminFilters>(emptyFilters());
const cohortA = ref<AdminFilters>(emptyFilters());
const cohortB = ref<
  { mode: "complement" } | { mode: "custom"; filters: AdminFilters }
>({ mode: "complement" });

const tab = ref("table");
const tabs = [
  { id: "table", label: "明细表" },
  { id: "cross", label: "交叉透视" },
  { id: "cohort", label: "两组对比" },
  { id: "time", label: "时间" },
];

const selected = ref<number[]>([]);
const openId = ref<number | null>(null);
const density = ref<"compact" | "cozy">("compact");
const sort = ref({ key: "submittedAt", desc: true });
const defaultColumns = [
  "id",
  "submittedAt",
  "status",
  "question:plans",
  "question:country",
  "question:usage",
  "question:duration",
  "question:concurrency",
  "question:ipRisk",
];
const columns = ref<string[]>([...defaultColumns]);

const visible = computed(() => applyFilters(records.value, filters.value));
const editing = computed<AdminFilters>({
  get: () =>
    mode.value === "filter"
      ? filters.value
      : mode.value === "cohort-a"
        ? cohortA.value
        : cohortB.value.mode === "custom"
          ? cohortB.value.filters
          : emptyFilters(),
  set: (value) => {
    if (mode.value === "filter") filters.value = value;
    else if (mode.value === "cohort-a") cohortA.value = value;
    else cohortB.value = { mode: "custom", filters: value };
  },
});
const groupA = computed(() => applyFilters(records.value, cohortA.value));
const groupB = computed(() => {
  if (cohortB.value.mode === "custom")
    return applyFilters(records.value, cohortB.value.filters);
  const ids = new Set(groupA.value.map((record) => record.id));
  return records.value.filter((record) => !ids.has(record.id));
});
const bIsComplement = computed(() => cohortB.value.mode === "complement");

function describeFilters(source: AdminFilters) {
  const parts = [
    ...source.statuses,
    ...(source.from || source.to
      ? [`${source.from || "…"} 至 ${source.to || "…"}`]
      : []),
    ...source.clauses.map((clause) => {
      const options = clause.options.map((value) =>
        displayOf(clause.dimension, value),
      );
      return `${clause.mode === "include" ? "" : "非 "}${options.join("/")}`;
    }),
  ];
  return parts.join(" · ");
}

const openIndex = computed(() =>
  visible.value.findIndex((record) => record.id === openId.value),
);
const openRecord = computed(
  () =>
    visible.value[openIndex.value] ??
    records.value.find((record) => record.id === openId.value) ??
    null,
);

const panelState = computed(() => ({
  filters: filters.value,
  tab: tab.value,
  cohorts: { a: cohortA.value, b: cohortB.value },
}));

function writeUrl() {
  const path = location.hash.slice(1).split("?")[0] || "/";
  history.replaceState(
    null,
    "",
    `${location.pathname}${location.search}#${path}?s=${encodeState(panelState.value)}`,
  );
}

watch(panelState, writeUrl, { deep: true });

function readUrl() {
  const query = location.hash.split("?")[1];
  if (!query) return;
  const stored = decodeState(new URLSearchParams(query).get("s") ?? "");
  if (!stored) return;
  filters.value = stored.filters;
  if (stored.tab && tabs.some((item) => item.id === stored.tab))
    tab.value = stored.tab;
  if (stored.cohorts) {
    cohortA.value = stored.cohorts.a ?? cohortA.value;
    cohortB.value = stored.cohorts.b ?? cohortB.value;
  }
}

function restore() {
  try {
    const storedColumns = localStorage.getItem("survey-pivot-columns");
    if (storedColumns) columns.value = JSON.parse(storedColumns);
    const storedDensity = localStorage.getItem("survey-pivot-density");
    if (storedDensity === "cozy" || storedDensity === "compact")
      density.value = storedDensity;
  } catch {
    /* Preferences are optional; defaults are already in place. */
  }
}

watch(columns, (value) =>
  localStorage.setItem("survey-pivot-columns", JSON.stringify(value)),
);
watch(density, (value) => localStorage.setItem("survey-pivot-density", value));

function switchMode(next: "filter" | "cohort-a" | "cohort-b") {
  if (
    next !== "filter" &&
    mode.value === "filter" &&
    !activeFilterCount(cohortA.value)
  )
    cohortA.value = cloneFilters(filters.value);
  mode.value = next;
  tab.value = next === "filter" ? "table" : "cohort";
}

function selectTab(id: string) {
  if (id === "cohort" && mode.value === "filter") switchMode("cohort-a");
  tab.value = id;
}

function makeBCustom() {
  cohortB.value = { mode: "custom", filters: emptyFilters() };
}

function editingFilters(): AdminFilters | null {
  if (mode.value === "filter") return filters.value;
  if (mode.value === "cohort-a") return cohortA.value;
  return cohortB.value.mode === "custom" ? cohortB.value.filters : null;
}

function drill(dimension: string, value: string) {
  if (mode.value === "cohort-b" && cohortB.value.mode === "complement")
    makeBCustom();
  const target = editingFilters();
  if (!target) return;
  const existing = target.clauses.find(
    (clause) => clause.dimension === dimension,
  );
  if (existing) {
    if (!existing.options.includes(value))
      existing.options = [...existing.options, value];
  } else {
    target.clauses = [
      ...target.clauses,
      { dimension, options: [value], mode: "include" },
    ];
  }
  tab.value = mode.value === "filter" ? "table" : "cohort";
}

function drillRange(from: string, to: string) {
  filters.value.from = from;
  filters.value.to = to;
  tab.value = "table";
}

function move(direction: -1 | 1) {
  const next = openIndex.value + direction;
  if (next < 0 || next >= visible.value.length) return;
  openId.value = visible.value[next].id;
}

function exportCsv() {
  const scope = selected.value.length
    ? visible.value.filter((record) => selected.value.includes(record.id))
    : visible.value;
  download(
    `survey-${scope.length}-${stamp()}.csv`,
    toCsv(scope, catalog.value, { annotations: false }),
    "text/csv;charset=utf-8",
  );
}

function exportJson() {
  const scope = selected.value.length
    ? visible.value.filter((record) => selected.value.includes(record.id))
    : visible.value;
  download(
    `survey-${scope.length}-${stamp()}.json`,
    toJson(scope, filters.value, catalog.value),
    "application/json",
  );
}

function stamp() {
  return new Date().toISOString().slice(0, 16).replace(/[:T]/g, "-");
}

async function load() {
  busy.value = true;
  try {
    const [definitions, payload] = await Promise.all([
      fetchPivotCatalog(),
      fetchPivotRecords(),
    ]);
    catalog.value = definitions;
    records.value = payload.records;
    state.value = "ready";
    notice.value = payload.truncated
      ? `记录较多，仅载入最新 ${payload.returned} 份。`
      : "";
  } catch (error) {
    state.value = "error";
    notice.value = error instanceof Error ? error.message : "请求失败。";
  } finally {
    busy.value = false;
  }
}

let meta: HTMLMetaElement | null = null;

onMounted(async () => {
  meta = document.createElement("meta");
  meta.name = "robots";
  meta.content = "noindex,nofollow";
  document.head.append(meta);
  restore();
  readUrl();
  await load();
});

onBeforeUnmount(() => {
  meta?.remove();
});
</script>

<template>
  <div class="panel">
    <header class="panel-bar">
      <strong class="panel-title">问卷数据透视</strong>
      <a class="panel-btn" href="#/studies/chatgpt-account-survey/results"
        >← 返回统计结果</a
      >
      <nav class="panel-tabs" aria-label="视图">
        <button
          v-for="item in tabs"
          :key="item.id"
          type="button"
          class="panel-tab"
          :class="{ 'is-on': tab === item.id }"
          :aria-current="tab === item.id ? 'page' : undefined"
          @click="selectTab(item.id)"
        >
          {{ item.label }}
        </button>
      </nav>
      <div class="panel-actions">
        <template v-if="state === 'ready'">
          <span class="panel-count"
            >{{ visible.length }} / {{ records.length }} 份</span
          >
          <button
            type="button"
            class="panel-btn"
            :disabled="busy"
            @click="exportCsv"
          >
            导出 CSV
          </button>
          <button
            type="button"
            class="panel-btn"
            :disabled="busy"
            @click="exportJson"
          >
            导出 JSON
          </button>
          <button
            type="button"
            class="panel-btn"
            :disabled="busy"
            @click="load"
          >
            刷新
          </button>
        </template>
      </div>
    </header>

    <p v-if="notice" class="panel-notice" role="status">{{ notice }}</p>

    <section v-if="state === 'loading'" class="panel-disabled">
      <h1>正在载入…</h1>
    </section>

    <section v-else-if="state === 'error'" class="panel-disabled">
      <h1>暂时无法读取数据</h1>
      <button type="button" class="panel-btn is-primary" @click="load">
        重试
      </button>
    </section>

    <div v-else class="panel-body" :class="{ 'has-drawer': openRecord }">
      <aside class="panel-rail">
        <div class="rail-mode">
          <button
            type="button"
            :class="{ 'is-on': mode === 'filter' }"
            @click="switchMode('filter')"
          >
            筛选
          </button>
          <button
            type="button"
            :class="{ 'is-on': mode === 'cohort-a' }"
            @click="switchMode('cohort-a')"
          >
            A 组
          </button>
          <button
            type="button"
            :class="{ 'is-on': mode === 'cohort-b' }"
            @click="switchMode('cohort-b')"
          >
            B 组
          </button>
        </div>
        <p v-if="mode === 'cohort-b' && bIsComplement" class="rail-banner">
          B 组现在是「其余全部」，即不属于 A 组的问卷。
          <button type="button" @click="makeBCustom">改成自定义条件</button>
        </p>
        <p v-if="mode === 'cohort-a'" class="rail-banner is-quiet">
          A 组当前条件：{{
            describeFilters(cohortA) || "未设置条件（等于全部问卷）"
          }}
        </p>
        <AdminFiltersPanel
          v-if="!(mode === 'cohort-b' && bIsComplement)"
          v-model="editing"
          :catalog="catalog"
          :records="records"
          :tag-options="[]"
          :matches="
            mode === 'filter'
              ? visible.length
              : mode === 'cohort-a'
                ? groupA.length
                : groupB.length
          "
          :total="records.length"
        />
      </aside>

      <main class="panel-main">
        <div v-if="tab === 'table'" class="panel-stack">
          <AdminTable
            :records="visible"
            :catalog="catalog"
            :columns="columns"
            :density="density"
            :sort="sort"
            :selected="selected"
            readonly
            @update:columns="columns = $event"
            @update:density="density = $event"
            @update:sort="sort = $event"
            @update:selected="selected = $event"
            @open="openId = $event"
          />
        </div>
        <AdminCross
          v-else-if="tab === 'cross'"
          :records="visible"
          :catalog="catalog"
          @drill="drill"
        />
        <AdminCohort
          v-else-if="tab === 'cohort'"
          :group-a="groupA"
          :group-b="groupB"
          :catalog="catalog"
          :a-label="describeFilters(cohortA)"
          :b-label="
            bIsComplement
              ? '其余全部'
              : describeFilters(
                  cohortB.mode === 'custom' ? cohortB.filters : emptyFilters(),
                )
          "
          @drill="drill"
          @edit="switchMode($event === 'a' ? 'cohort-a' : 'cohort-b')"
        />
        <AdminTimeline v-else :records="visible" @drill="drillRange" />

        <footer class="panel-foot">
          <span v-if="activeFilterCount(filters)">
            筛选条件 {{ activeFilterCount(filters) }} 项
          </span>
          <span class="panel-hint">
            地址栏保存当前条件，可直接收藏；CSV 带 UTF-8 BOM，Excel
            打开不乱码；选中若干行时导出只包含选中的问卷。这是公开只读视图，不提供标签与备注功能。
          </span>
        </footer>
      </main>

      <AdminDrawer
        v-if="openRecord"
        :record="openRecord"
        :catalog="catalog"
        :tag-options="[]"
        :has-prev="openIndex > 0"
        :has-next="openIndex >= 0 && openIndex < visible.length - 1"
        :saving="false"
        readonly
        @close="openId = null"
        @move="move"
      />
    </div>
  </div>
</template>

<style scoped src="./AdminPanelPage.css"></style>
