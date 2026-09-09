<script setup lang="ts">
import { computed, type Component } from "vue";
import type { AssociationGroup, SurveyOutcome } from "../../data/surveyApi";
import { tone, direction, coefficient, percent } from "../../data/surveyAssociation";
const props = defineProps<{
  row: AssociationGroup["rows"][number];
  outcome: SurveyOutcome;
  layout: string;
  icon?: Component;
}>();
const emit = defineEmits<{ inspect: [event: MouseEvent] }>();
const current = computed(() => props.row[props.outcome]);
</script>

<template>
  <button
    type="button"
    aria-haspopup="dialog"
    :aria-label="`查看${row.label}的异常比较`"
    @click="emit('inspect', $event)"
    class="association-card"
    :class="[`tone-${tone(current.phi)}`, `layout-${layout}`]"
    :data-option="row.label"
  >
    <span class="option-summary">
      <span class="option-identity">
        <component :is="icon" v-if="icon" class="tool-icon" />
        <span v-else-if="layout === 'tool'" class="tool-monogram" aria-hidden="true">{{
          row.label.slice(0, 2)
        }}</span>
        <strong class="option-label">{{ row.label }}</strong>
      </span>
      <span class="option-signal"
        ><span class="direction">{{ direction(current.phi) }}</span
        ><span class="phi">φ {{ coefficient(current.phi) }}</span></span
      >
      <span class="option-rates"
        ><span
          ><span class="rate-label">选择该项</span><strong>{{ percent(current.selected) }}</strong
          ><small>{{ current.selected.events }}/{{ current.selected.total }} 份</small></span
        ><span
          ><span class="rate-label">未选该项</span><strong>{{ percent(current.unselected) }}</strong
          ><small>{{ current.unselected.events }}/{{ current.unselected.total }} 份</small></span
        ></span
      >
    </span>
  </button>
</template>

<style scoped src="./SurveyAssociationCard.css"></style>
<style scoped src="./SurveyAssociationTone.css"></style>
