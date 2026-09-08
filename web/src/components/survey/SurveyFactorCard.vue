<script setup lang="ts">
import { computed, ref } from "vue";
import type { FactorDistribution, SurveyOutcome } from "../../data/surveyExampleStatistics";
const props = defineProps<{ distribution: FactorDistribution; showExample: boolean }>();
const outcome = ref<SurveyOutcome>("degraded");
const group = computed(() => props.distribution.groups[outcome.value]);
const percentage = (count: number) =>
  group.value.total > 0 ? (count / group.value.total) * 100 : 0;
</script>

<template>
  <section class="factor-card" :aria-labelledby="`factor-${distribution.id}`">
    <header>
      <h2 :id="`factor-${distribution.id}`">{{ distribution.title }}</h2>
      <p>{{ distribution.description }}</p>
    </header>
    <div class="outcome-switch" role="group" :aria-label="`${distribution.title}统计状态`">
      <button type="button" :aria-pressed="outcome === 'degraded'" @click="outcome = 'degraded'">
        降智
      </button>
      <button type="button" :aria-pressed="outcome === 'banned'" @click="outcome = 'banned'">
        封号
      </button>
    </div>
    <ul class="distribution-list">
      <li v-for="row in group.rows" :key="row.label">
        <div class="distribution-label">
          <span>{{ row.label }}</span
          ><strong>{{
            showExample && group.total > 0
              ? `${row.count} 份 · ${percentage(row.count).toFixed(1)}%`
              : "—"
          }}</strong>
        </div>
        <div class="distribution-track" aria-hidden="true">
          <span
            :class="outcome"
            :style="{ width: showExample ? `${percentage(row.count)}%` : '0%' }"
          ></span>
        </div>
      </li>
    </ul>
  </section>
</template>

<style scoped src="./SurveyFactorCard.css"></style>
