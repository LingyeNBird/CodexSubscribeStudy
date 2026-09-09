<script setup lang="ts">
import { computed, ref } from "vue";
import SurveyAssociationCard from "./SurveyAssociationCard.vue";
import type { AssociationGroup } from "../../data/surveyApi";
const props = defineProps<{
  associations: AssociationGroup[];
}>();
const onlyAssociated = ref(false);
const factor = ref("all");
const groups = computed(() =>
  factor.value === "all"
    ? props.associations
    : props.associations.filter((group) => group.id === factor.value),
);
const cards = computed(() => {
  const entries = groups.value.flatMap((group) =>
    group.rows.map((row) => ({
      ...row,
      id: `${group.id}:${row.label}`,
      title: group.title,
      scope: group.scope,
      strength: Math.max(
        Math.abs(row.degraded.phi ?? 0),
        Math.abs(row.banned.phi ?? 0),
        Math.abs(row.limited.phi ?? 0),
      ),
    })),
  );
  return entries
    .filter((entry) => !onlyAssociated.value || entry.strength >= 0.1)
    .sort((a, b) => b.strength - a.strength);
});
</script>

<template>
  <section class="correlations" aria-labelledby="correlation-heading">
    <header class="correlation-heading">
      <h2 id="correlation-heading">异常相关性</h2>
      <p>
        对比选择某个因素选项与未选择该项的问卷，观察降智、封号和风控（限流）。相关性不等于因果，也不表示时间先后。
      </p>
    </header>
    <div class="correlation-toolbar">
      <label for="correlation-factor"
        >比较因素<select id="correlation-factor" v-model="factor">
          <option value="all">全部因素</option>
          <option v-for="group in associations" :key="group.id" :value="group.id">
            {{ group.title }}
          </option>
        </select></label
      ><span>样本关联 · 未经混杂因素调整</span>
    </div>
    <div class="correlation-legend" aria-label="关联强度颜色说明">
      <span class="legend-strong">强关联 ≥ 0.50</span>
      <span class="legend-moderate">中等关联 ≥ 0.30</span>
      <span class="legend-weak">较弱关联 ≥ 0.10</span>
      <span class="legend-negligible">未见明显关联 &lt; 0.10</span>
      <span class="legend-unknown">暂无有效比较</span>
    </div>
    <div class="card-filter">
      <label><input v-model="onlyAssociated" type="checkbox" />仅显示 |φ| ≥ 0.10 的选项</label>
      <span>显示 {{ cards.length }} 项 · 按关联强度排序</span>
    </div>
    <p class="color-explanation">
      颜色表示关联强弱，不表示危险或安全；负相关也按强度着色。卡片底色取关联最强的一项。
    </p>
    <div class="association-grid">
      <SurveyAssociationCard
        v-for="card in cards"
        :key="card.id"
        :label="card.label"
        :title="card.title"
        :scope="card.scope"
        :degraded="card.degraded"
        :banned="card.banned"
        :limited="card.limited"
      />
    </div>
    <p v-if="!cards.length" class="correlation-empty">
      当前因素没有达到此关联强度的选项，可取消筛选查看全部。
    </p>
    <details class="correlation-method">
      <summary>如何理解相关系数与比较范围？</summary>
      <p>
        φ 的范围是 −1 到 +1：正值表示选择该项的样本更常报告对应异常，负值表示更少；接近 0
        表示本样本中的二元线性关联较弱。它不是概率，也不是已通过显著性检验的结论。
      </p>
      <p>
        每个选项与同一道题中“未选择该项”的有效回答比较。多选题按问卷去重；反代工具等条件题只在适用人群内比较，不把没有看到题目的人当成否定回答。任一变量没有变化时不计算系数。
      </p>
      <p>
        模型、降智发现方式和风控识别方式依赖对应异常状态才有回答，缺少正常对照，仅展示样本分布。
      </p>
      <p>
        未控制套餐、使用强度等混杂因素，没有给出总体风险或显著性结论。统计单位为问卷，不等于独立用户。
        颜色分档仅为经验性展示规则；接近零不代表已证明无关联或安全，少量样本也可能产生较大系数。卡片中的对比解释不是导致异常的原因。
      </p>
    </details>
  </section>
</template>

<style scoped src="./SurveyCorrelationPanel.css"></style>
