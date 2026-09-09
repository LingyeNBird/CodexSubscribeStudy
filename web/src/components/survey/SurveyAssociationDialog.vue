<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import type { AssociationGroup, SurveyOutcome } from "../../data/surveyApi";
import { coefficient, direction, explanation, percent, tone } from "../../data/surveyAssociation";
const props = defineProps<{
  row: AssociationGroup["rows"][number];
  title: string;
  scope: string;
  outcome: SurveyOutcome;
}>();
const emit = defineEmits<{ close: [] }>();
const dialog = ref<HTMLDialogElement | null>(null);
const outcomes = computed(() => [
  { id: "degraded", label: "降智", value: props.row.degraded },
  { id: "banned", label: "封号", value: props.row.banned },
  { id: "limited", label: "风控（限流）", value: props.row.limited },
]);
const backdropPressed = ref(false);
onMounted(() => dialog.value?.showModal());
onBeforeUnmount(() => {
  if (dialog.value?.open) dialog.value.close();
});
</script>

<template>
  <Teleport to="body">
    <dialog
      ref="dialog"
      class="association-dialog"
      aria-labelledby="association-dialog-heading"
      @close="emit('close')"
      @pointerdown="backdropPressed = $event.target === dialog"
      @click.self="backdropPressed && dialog?.close()"
    >
      <div class="dialog-content">
        <header class="dialog-heading">
          <div>
            <p>{{ title }}</p>
            <h2 id="association-dialog-heading">{{ row.label }}</h2>
          </div>
          <button
            type="button"
            class="dialog-close"
            aria-label="关闭比较详情"
            autofocus
            @click="dialog?.close()"
          >
            关闭 <span aria-hidden="true">×</span>
          </button>
        </header>
        <div class="option-comparisons">
          <section
            v-for="item in outcomes"
            :key="item.id"
            class="outcome-detail"
            :class="{ 'is-current': outcome === item.id }"
            :aria-label="`${row.label}与${item.label}的关联`"
          >
            <header>
              <h3>{{ item.label }}</h3>
              <span>φ {{ coefficient(item.value.phi) }}</span>
            </header>
            <span class="detail-direction" :class="`tone-${tone(item.value.phi)}`">{{
              direction(item.value.phi)
            }}</span>
            <dl>
              <div>
                <dt>选择该项</dt>
                <dd>
                  <strong>{{ percent(item.value.selected) }}</strong
                  ><span>{{ item.value.selected.events }}/{{ item.value.selected.total }} 份</span>
                </dd>
              </div>
              <div>
                <dt>未选该项</dt>
                <dd>
                  <strong>{{ percent(item.value.unselected) }}</strong
                  ><span
                    >{{ item.value.unselected.events }}/{{ item.value.unselected.total }} 份</span
                  >
                </dd>
              </div>
            </dl>
            <p>{{ explanation(item.value) }}</p>
          </section>
        </div>
        <footer class="dialog-scope">
          <strong>比较人群</strong>
          <p>{{ scope }}</p>
        </footer>
      </div>
    </dialog>
  </Teleport>
</template>

<style scoped src="./SurveyAssociationDialog.css"></style>
<style scoped src="./SurveyAssociationTone.css"></style>
