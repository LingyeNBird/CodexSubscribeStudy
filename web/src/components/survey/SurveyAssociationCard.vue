<script setup lang="ts">
import type { Component } from "vue";
import type { AssociationGroup } from "../../data/surveyStatistics";
import {
  tone,
  direction,
  coefficient,
  percent,
} from "../../data/surveyAssociation";
defineProps<{
  row: AssociationGroup["rows"][number];
  outcomeLabel: string;
  layout: string;
  icon?: Component;
  prerequisite?: boolean;
  muted?: boolean;
  tonePhi?: number | null;
}>();
const emit = defineEmits<{
  inspect: [event: MouseEvent];
  premise: [event: MouseEvent];
}>();
</script>

<template>
  <button
    type="button"
    aria-haspopup="dialog"
    :aria-label="`查看${row.label}的${outcomeLabel}比较`"
    @contextmenu.prevent="emit('premise', $event)"
    @click="emit('inspect', $event)"
    class="association-card"
    :class="[
      `tone-${tone(tonePhi === undefined ? row.association.phi : tonePhi)}`,
      `layout-${layout}`,
      { 'is-prerequisite': prerequisite, 'is-muted': muted },
    ]"
    :data-option="row.label"
  >
    <span class="option-summary">
      <span class="option-identity">
        <component :is="icon" v-if="icon" class="tool-icon" />
        <span
          v-else-if="layout === 'tool'"
          class="tool-monogram"
          aria-hidden="true"
          >{{ row.label.slice(0, 2) }}</span
        >
        <strong class="option-label">{{ row.label }}</strong>
        <span v-if="prerequisite" class="premise-mark">前提</span>
      </span>
      <span class="option-signal"
        ><span class="outcome-label">{{ outcomeLabel }}</span
        ><span class="direction">{{ direction(row.association.phi) }}</span
        ><span class="phi">φ {{ coefficient(row.association.phi) }}</span></span
      >
      <span class="option-rates">
        <span
          ><span class="rate-label">选择该项</span
          ><strong>{{ percent(row.association.selected) }}</strong
          ><small
            >{{ row.association.selected.events }}/{{
              row.association.selected.total
            }}
            份</small
          ></span
        >
        <span
          ><span class="rate-label">未选该项</span
          ><strong>{{ percent(row.association.unselected) }}</strong
          ><small
            >{{ row.association.unselected.events }}/{{
              row.association.unselected.total
            }}
            份</small
          ></span
        >
      </span>
    </span>
  </button>
</template>

<style scoped src="./SurveyAssociationCard.css"></style>
<style scoped src="./SurveyAssociationTone.css"></style>
