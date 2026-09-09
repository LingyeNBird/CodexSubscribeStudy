<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import katex from "katex";
import "katex/dist/katex.min.css";
import type { AssociationGroup } from "../../data/surveyStatistics";
import {
  coefficient,
  direction,
  explanation,
  percent,
  tone,
} from "../../data/surveyAssociation";
const props = defineProps<{
  row: AssociationGroup["rows"][number];
  title: string;
  scope: string;
  outcomeLabel: string;
}>();
const counts = computed(() => {
  const { selected, unselected } = props.row.association;
  return {
    a: selected.events,
    b: selected.total - selected.events,
    c: unselected.events,
    d: unselected.total - unselected.events,
  };
});
const renderFormula = (formula: string) =>
  katex.renderToString(formula, {
    displayMode: true,
    trust: false,
    throwOnError: true,
  });
const generalFormula = renderFormula(
  String.raw`\varphi = \frac{A \times D - B \times C}{\sqrt{(A+B)(C+D)(A+C)(B+D)}}`,
);
const substitutedFormula = computed(() => {
  const { a, b, c, d } = counts.value;
  return renderFormula(
    String.raw`\varphi = \frac{${a} \times ${d} - ${b} \times ${c}}{\sqrt{(${a}+${b})(${c}+${d})(${a}+${c})(${b}+${d})}}`,
  );
});
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
        <section
          class="outcome-detail"
          :aria-label="`${row.label}与${outcomeLabel}的关联`"
        >
          <header>
            <h3>{{ outcomeLabel }}</h3>
            <span>φ {{ coefficient(row.association.phi) }}</span>
          </header>
          <span
            class="detail-direction"
            :class="`tone-${tone(row.association.phi)}`"
            >{{ direction(row.association.phi) }}</span
          >
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
        <section
          class="phi-calculation"
          aria-labelledby="phi-calculation-heading"
        >
          <h3 id="phi-calculation-heading">φ 计算过程</h3>
          <table class="phi-counts">
            <thead>
              <tr>
                <th scope="col">问卷分组</th>
                <th scope="col">报告该异常</th>
                <th scope="col">未报告该异常</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <th scope="row">选择该项</th>
                <td><strong>A</strong> = {{ counts.a }} 份</td>
                <td><strong>B</strong> = {{ counts.b }} 份</td>
              </tr>
              <tr>
                <th scope="row">未选该项</th>
                <td><strong>C</strong> = {{ counts.c }} 份</td>
                <td><strong>D</strong> = {{ counts.d }} 份</td>
              </tr>
            </tbody>
          </table>
          <h4>公式</h4>
          <div
            class="phi-formula-scroll"
            tabindex="0"
            role="region"
            aria-label="关联系数公式"
          >
            <div class="phi-formula" v-html="generalFormula"></div>
          </div>
          <h4>代入数值</h4>
          <div
            class="phi-formula-scroll"
            tabindex="0"
            role="region"
            aria-label="代入问卷份数的计算公式"
          >
            <div class="phi-formula" v-html="substitutedFormula"></div>
          </div>
          <div class="phi-result">
            <span>结果</span>
            <strong v-if="row.association.phi !== null"
              >φ ≈ {{ coefficient(row.association.phi) }}</strong
            >
            <strong v-else>无法计算（分母为 0）</strong>
          </div>
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
