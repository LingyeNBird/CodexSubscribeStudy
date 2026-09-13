<script setup lang="ts">
import { computed, ref } from "vue";
import type { AdminCatalog, AdminRecord } from "../../data/adminApi";
import {
  activeFilterCount,
  emptyFilters,
  listDimensions,
  optionsFor,
  statusLabels,
  type AdminFilters,
  type FilterClause,
} from "./adminModel";

const filters = defineModel<AdminFilters>({ required: true });
const props = defineProps<{
  catalog: AdminCatalog | null;
  records: AdminRecord[];
  tagOptions: string[];
  matches: number;
  total: number;
}>();

const dimensions = computed(() => listDimensions(props.catalog));
const optionCache = computed(() => {
  const cache = new Map<string, ReturnType<typeof optionsFor>>();
  for (const dimension of dimensions.value)
    cache.set(
      dimension.id,
      optionsFor(props.records, dimension.id, props.catalog),
    );
  return cache;
});
const available = computed(() => {
  const used = new Set(filters.value.clauses.map((clause) => clause.dimension));
  return dimensions.value.filter((dimension) => !used.has(dimension.id));
});
const pending = ref("");

function toggleStatus(status: string) {
  const current = filters.value.statuses;
  filters.value.statuses = current.includes(status)
    ? current.filter((item) => item !== status)
    : [...current, status];
}

function toggleTag(tag: string) {
  const current = filters.value.tags;
  filters.value.tags = current.includes(tag)
    ? current.filter((item) => item !== tag)
    : [...current, tag];
}

function addClause() {
  const dimension = pending.value || available.value[0]?.id;
  if (!dimension) return;
  filters.value.clauses = [
    ...filters.value.clauses,
    { dimension, options: [], mode: "include" },
  ];
  pending.value = "";
}

function removeClause(clause: FilterClause) {
  filters.value.clauses = filters.value.clauses.filter(
    (item) => item !== clause,
  );
}

function toggleClauseOption(clause: FilterClause, value: string) {
  clause.options = clause.options.includes(value)
    ? clause.options.filter((item) => item !== value)
    : [...clause.options, value];
}

function toggleClauseMode(clause: FilterClause) {
  clause.mode = clause.mode === "include" ? "exclude" : "include";
}

function setRange(days: number) {
  const end = new Date();
  const start = new Date(end.getTime() - (days - 1) * 86400000);
  filters.value.from = start.toISOString().slice(0, 10);
  filters.value.to = end.toISOString().slice(0, 10);
}

function clearRange() {
  filters.value.from = "";
  filters.value.to = "";
}

function reset() {
  const cleared = emptyFilters();
  filters.value.statuses = cleared.statuses;
  filters.value.tags = cleared.tags;
  filters.value.clauses = cleared.clauses;
  filters.value.from = "";
  filters.value.to = "";
  filters.value.search = "";
}

/** A clause without options is inert; say so instead of pretending it filters. */
function clauseIncomplete(clause: FilterClause) {
  return clause.options.length === 0;
}

function dimensionTitle(id: string) {
  return dimensions.value.find((dimension) => dimension.id === id)?.title ?? id;
}
</script>

<template>
  <div class="rail">
    <header class="rail-head">
      <strong>命中 {{ matches }} / {{ total }} 份</strong>
      <button
        v-if="activeFilterCount(filters)"
        class="rail-reset"
        type="button"
        @click="reset"
      >
        清空条件
      </button>
    </header>

    <section class="rail-block">
      <h3>异常状态</h3>
      <div class="rail-chips">
        <button
          v-for="status in statusLabels"
          :key="status"
          type="button"
          class="pick"
          :class="{ 'is-on': filters.statuses.includes(status) }"
          :aria-pressed="filters.statuses.includes(status)"
          @click="toggleStatus(status)"
        >
          {{ status }}
        </button>
      </div>
    </section>

    <section class="rail-block">
      <h3>提交时间</h3>
      <div class="rail-range">
        <label>从<input v-model="filters.from" type="date" /></label>
        <label>到<input v-model="filters.to" type="date" /></label>
      </div>
      <div class="rail-chips">
        <button type="button" class="pick" @click="setRange(7)">
          最近 7 天
        </button>
        <button type="button" class="pick" @click="setRange(30)">
          最近 30 天
        </button>
        <button
          v-if="filters.from || filters.to"
          type="button"
          class="pick"
          @click="clearRange"
        >
          清除时间
        </button>
      </div>
    </section>

    <section class="rail-block">
      <h3>题目条件</h3>
      <div class="rail-chips">
        <select
          v-model="pending"
          class="rail-select"
          aria-label="选择要添加的题目"
        >
          <option value="">选择题目…</option>
          <option
            v-for="dimension in available"
            :key="dimension.id"
            :value="dimension.id"
          >
            {{ dimension.title }}
          </option>
        </select>
        <button
          type="button"
          class="pick"
          :disabled="!available.length"
          @click="addClause"
        >
          添加
        </button>
      </div>

      <article
        v-for="clause in filters.clauses"
        :key="clause.dimension"
        class="clause"
        :class="{ 'is-empty': clauseIncomplete(clause) }"
      >
        <header>
          <span class="clause-title">{{
            dimensionTitle(clause.dimension)
          }}</span>
          <button
            type="button"
            class="clause-mode"
            @click="toggleClauseMode(clause)"
          >
            {{ clause.mode === "include" ? "包含" : "排除" }}
          </button>
          <button
            type="button"
            class="clause-drop"
            :aria-label="`删除${dimensionTitle(clause.dimension)}条件`"
            @click="removeClause(clause)"
          >
            ×
          </button>
        </header>
        <p v-if="clauseIncomplete(clause)" class="clause-hint">
          选择至少一个选项后才会生效。
        </p>
        <div class="rail-chips">
          <button
            v-for="option in optionCache.get(clause.dimension) ?? []"
            :key="option.value"
            type="button"
            class="pick small"
            :class="{ 'is-on': clause.options.includes(option.value) }"
            :aria-pressed="clause.options.includes(option.value)"
            @click="toggleClauseOption(clause, option.value)"
          >
            {{ option.display }}<em v-if="option.hint">{{ option.hint }}</em
            ><b>{{ option.count }}</b>
          </button>
        </div>
      </article>
    </section>

    <section class="rail-block">
      <h3>文本搜索</h3>
      <input
        v-model="filters.search"
        type="search"
        class="rail-search"
        placeholder="搜答案、补充文本、备注"
      />
    </section>

    <section v-if="tagOptions.length" class="rail-block">
      <h3>标签</h3>
      <div class="rail-chips">
        <button
          v-for="tag in tagOptions"
          :key="tag"
          type="button"
          class="pick"
          :class="{ 'is-on': filters.tags.includes(tag) }"
          :aria-pressed="filters.tags.includes(tag)"
          @click="toggleTag(tag)"
        >
          {{ tag }}
        </button>
      </div>
    </section>
  </div>
</template>

<style scoped src="./AdminFilters.css"></style>
