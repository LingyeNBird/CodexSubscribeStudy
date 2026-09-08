<script setup lang="ts">
import { computed } from "vue";
import type { BinaryAssociation } from "../../data/surveyAssociation";
const props = defineProps<{
  label: string;
  title: string;
  scope: string;
  degraded: BinaryAssociation;
  banned: BinaryAssociation;
  showExample: boolean;
}>();
const outcomes = computed(() => [
  { label: "降智", value: props.degraded },
  { label: "封号", value: props.banned },
]);
const level = (phi: number | null) => {
  if (!props.showExample || phi === null) return "unknown";
  const magnitude = Math.abs(phi);
  return magnitude >= 0.5
    ? "strong"
    : magnitude >= 0.3
      ? "moderate"
      : magnitude >= 0.1
        ? "weak"
        : "negligible";
};
const labels = {
  strong: "强关联",
  moderate: "中等关联",
  weak: "较弱关联",
  negligible: "未见明显关联",
  unknown: "暂无有效比较",
};
const strongest = computed(() => {
  const values = outcomes.value.flatMap(({ value }) =>
    value.phi === null ? [] : [Math.abs(value.phi)],
  );
  return level(values.length ? Math.max(...values) : null);
});
const coefficient = (phi: number | null) =>
  !props.showExample ? "—" : phi === null ? "无法计算" : `${phi > 0 ? "+" : ""}${phi.toFixed(3)}`;
const rate = (value: BinaryAssociation["selected"]) =>
  value.total
    ? `${value.events}/${value.total}（${((100 * value.events) / value.total).toFixed(1)}%）`
    : "无可比较样本";
const explanation = (value: BinaryAssociation, outcome: string) => {
  if (!props.showExample) return "接入有效问卷后展示对比结果。";
  if (value.phi === null) return "缺少可比较样本，或变量没有变化，无法判断关联。";
  const difference =
    100 *
    (value.selected.events / value.selected.total -
      value.unselected.events / value.unselected.total);
  if (difference === 0) return `选择与未选择该项的样本，报告${outcome}的比例相同。`;
  return `选择该项的样本，报告${outcome}的比例比未选择者${difference > 0 ? "高" : "低"} ${Math.abs(difference).toFixed(1)} 个百分点。`;
};
</script>

<template>
  <article class="association-card" :class="`tone-${strongest}`">
    <header>
      <span class="factor-title">{{ title }}</span>
      <span class="strength-label">{{ labels[strongest] }}</span>
      <h3>{{ label }}</h3>
    </header>
    <section
      v-for="outcome in outcomes"
      :key="outcome.label"
      class="outcome-detail"
      :class="`tone-${level(outcome.value.phi)}`"
      :aria-label="`与${outcome.label}的关联`"
    >
      <div class="score-line">
        <strong>{{ outcome.label }}</strong
        ><span>φ {{ coefficient(outcome.value.phi) }}</span>
      </div>
      <div class="direction-label">
        {{ labels[level(outcome.value.phi)]
        }}<template v-if="showExample && outcome.value.phi !== null && outcome.value.phi !== 0">
          · {{ outcome.value.phi > 0 ? "正相关" : "负相关" }}</template
        >
      </div>
      <p>{{ explanation(outcome.value, outcome.label) }}</p>
      <dl v-if="showExample">
        <div>
          <dt>选择该项</dt>
          <dd>{{ rate(outcome.value.selected) }}</dd>
        </div>
        <div>
          <dt>未选该项</dt>
          <dd>{{ rate(outcome.value.unselected) }}</dd>
        </div>
      </dl>
    </section>
    <details class="comparison-scope">
      <summary>比较人群</summary>
      <p>{{ scope }}</p>
    </details>
  </article>
</template>

<style scoped src="./SurveyAssociationCard.css"></style>
