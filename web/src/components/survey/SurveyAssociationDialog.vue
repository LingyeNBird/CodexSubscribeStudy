<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from "vue";
import type { AssociationGroup } from "../../data/surveyStatistics";
import { coefficient, direction, explanation, percent, tone } from "../../data/surveyAssociation";
defineProps<{
  row: AssociationGroup["rows"][number];
  title: string;
  scope: string;
  outcomeLabel: string;
}>();
const emit = defineEmits<{ close: [] }>();
const dialog = ref<HTMLDialogElement | null>(null);
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
        <section class="outcome-detail" :aria-label="`${row.label}与${outcomeLabel}的关联`">
          <header>
            <h3>{{ outcomeLabel }}</h3>
            <span>φ {{ coefficient(row.association.phi) }}</span>
          </header>
          <span class="detail-direction" :class="`tone-${tone(row.association.phi)}`">{{
            direction(row.association.phi)
          }}</span>
          <dl>
            <div>
              <dt>选择该项</dt>
              <dd>
                <strong>{{ percent(row.association.selected) }}</strong
                ><span
                  >{{ row.association.selected.events }}/{{
                    row.association.selected.total
                  }}
                  份</span
                >
              </dd>
            </div>
            <div>
              <dt>未选该项</dt>
              <dd>
                <strong>{{ percent(row.association.unselected) }}</strong
                ><span
                  >{{ row.association.unselected.events }}/{{
                    row.association.unselected.total
                  }}
                  份</span
                >
              </dd>
            </div>
          </dl>
          <p>{{ explanation(row.association) }}</p>
        </section>
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
