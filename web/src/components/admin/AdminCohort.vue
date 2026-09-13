<script setup lang="ts">
import { computed, ref } from "vue";
import type { AdminCatalog, AdminRecord } from "../../data/adminApi";
import { cohortRows } from "./adminModel";

const props = defineProps<{
  groupA: AdminRecord[];
  groupB: AdminRecord[];
  catalog: AdminCatalog | null;
  aLabel: string;
  bLabel: string;
}>();

const emit = defineEmits<{
  drill: [dimension: string, value: string];
  edit: [cohort: "a" | "b"];
}>();

const threshold = ref(0);
const collapsed = ref<Record<string, boolean>>({});

const groups = computed(() =>
  cohortRows(props.groupA, props.groupB, props.catalog).map((group) => ({
    ...group,
    rows: [...group.rows].sort(
      (left, right) => Math.abs(right.diff) - Math.abs(left.diff),
    ),
  })),
);

/** Questions start expanded; only an explicit collapse hides a section. */
function expanded(title: string) {
  return collapsed.value[title] !== true;
}

function applicable(
  group: { rows: { aTotal: number; bTotal: number }[] },
  side: "aTotal" | "bTotal",
) {
  return group.rows[0]?.[side] ?? 0;
}

function formatPercent(count: number, total: number) {
  return total ? `${((100 * count) / total).toFixed(1)}%` : "—";
}

function formatPhi(value: number | null) {
  return value === null ? "—" : `${value > 0 ? "+" : ""}${value.toFixed(3)}`;
}

function toggle(title: string) {
  collapsed.value[title] = expanded(title);
}

function strongGrade(diff: number) {
  const magnitude = Math.abs(diff);
  if (magnitude >= 25) return "is-strong";
  if (magnitude >= 10) return "is-mid";
  return "";
}
</script>

<template>
  <section class="cohort">
    <div class="cohort-head">
      <div class="cohort-groups">
        <button
          type="button"
          class="cohort-card is-a"
          @click="emit('edit', 'a')"
        >
          <small>编辑 A 组</small>
          <strong>{{ groupA.length }} 份</strong>
          <span>{{ aLabel || "未设置条件" }}</span>
        </button>
        <button
          type="button"
          class="cohort-card is-b"
          @click="emit('edit', 'b')"
        >
          <small>编辑 B 组</small>
          <strong>{{ groupB.length }} 份</strong>
          <span>{{ bLabel || "未设置条件" }}</span>
        </button>
      </div>
      <label class="cohort-threshold"
        >只看差值 ≥
        <select v-model.number="threshold">
          <option :value="0">全部</option>
          <option :value="5">5 个百分点</option>
          <option :value="10">10 个百分点</option>
          <option :value="20">20 个百分点</option>
        </select></label
      >
    </div>
    <p class="cohort-note">
      差值为 A 组占比减去 B 组占比。φ
      只在同时属于两组的样本上计算，未做显著性检验，也未控制混杂因素。
    </p>

    <p v-if="!groupA.length || !groupB.length" class="cohort-empty">
      两组都至少要有 1 份问卷才能对比。
    </p>

    <div v-else class="cohort-list">
      <article
        v-for="group in groups"
        :key="group.title"
        class="cohort-question"
      >
        <button type="button" class="cohort-title" @click="toggle(group.title)">
          <span>{{ group.title }}</span>
          <em
            >A 组 {{ applicable(group, "aTotal") }} · B 组
            {{ applicable(group, "bTotal") }}</em
          >
          <span
            class="cohort-chevron"
            :class="{ 'is-open': expanded(group.title) }"
            aria-hidden="true"
          ></span>
        </button>
        <div v-if="expanded(group.title)" class="cohort-rows">
          <div class="cohort-row is-header">
            <span>选项</span><span>A 组</span><span>B 组</span><span>差值</span
            ><span>φ</span>
          </div>
          <template
            v-for="row in group.rows"
            :key="`${group.title}:${row.value}`"
          >
            <div
              v-if="Math.abs(row.diff) >= threshold"
              class="cohort-row"
              :class="strongGrade(row.diff)"
            >
              <button
                type="button"
                class="cohort-option"
                @click="emit('drill', `question:${row.dimension}`, row.value)"
              >
                {{ row.display }}<em v-if="row.hint">{{ row.hint }}</em>
              </button>
              <span
                >{{ row.aCount }} / {{ row.aTotal
                }}<small>{{
                  formatPercent(row.aCount, row.aTotal)
                }}</small></span
              >
              <span
                >{{ row.bCount }} / {{ row.bTotal
                }}<small>{{
                  formatPercent(row.bCount, row.bTotal)
                }}</small></span
              >
              <span class="cohort-diff"
                >{{ row.diff > 0 ? "+" : "" }}{{ row.diff.toFixed(1)
                }}<small>pp</small></span
              >
              <span class="cohort-phi">{{ formatPhi(row.phi) }}</span>
            </div>
          </template>
        </div>
      </article>
    </div>
  </section>
</template>

<style scoped src="./AdminCohort.css"></style>
