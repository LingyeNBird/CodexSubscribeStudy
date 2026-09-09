<script setup lang="ts">
import {
  computed,
  defineAsyncComponent,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  shallowRef,
  watch,
} from "vue";
import SurveyAssociationCard from "./SurveyAssociationCard.vue";
import { toolIcons } from "../icons/toolIcons";
import type { SurveyFactorSummary, SurveyOutcome } from "../../data/surveyApi";
import {
  calculateAssociations,
  effectiveOutcomeMask,
  surveyOutcomes,
  type AssociationGroup,
} from "../../data/surveyStatistics";
const SurveyAssociationDialog = defineAsyncComponent(
  () => import("./SurveyAssociationDialog.vue"),
);
const props = defineProps<{ factors: SurveyFactorSummary[] }>();
const emit = defineEmits<{ outcomeChange: [mask: number] }>();
const onlyAssociated = ref(false);
const factor = ref("all");
const selectedOutcomes = ref<SurveyOutcome[]>(["degraded"]);
const lastSelectedOutcome = ref<SurveyOutcome>("degraded");
const outcomeMask = computed(() =>
  effectiveOutcomeMask(selectedOutcomes.value, lastSelectedOutcome.value),
);
watch(outcomeMask, (mask) => emit("outcomeChange", mask), { immediate: true });
const outcomeLabel = computed(() =>
  surveyOutcomes
    .filter((item) => (item.bit & outcomeMask.value) !== 0)
    .map((item) => item.label)
    .join("或"),
);
const associations = computed(() =>
  calculateAssociations(props.factors, outcomeMask.value),
);
function toggleOutcome(id: SurveyOutcome) {
  if (selectedOutcomes.value.includes(id)) {
    selectedOutcomes.value = selectedOutcomes.value.filter(
      (value) => value !== id,
    );
  } else {
    lastSelectedOutcome.value = id;
    selectedOutcomes.value = surveyOutcomes
      .filter(
        (item) => item.id === id || selectedOutcomes.value.includes(item.id),
      )
      .map((item) => item.id);
  }
}
const collapsedSections = ref<Record<string, boolean>>({});
const collapsedGroups = ref<Record<string, boolean>>({});
const factorPicker = ref<HTMLElement | null>(null);
const factorTrigger = ref<HTMLButtonElement | null>(null);
const menuOpen = ref(false);
const factorLabel = computed(
  () =>
    associations.value.find((group) => group.id === factor.value)?.title ??
    "全部因素",
);
const comparison = shallowRef<{
  group: AssociationGroup;
  row: AssociationGroup["rows"][number];
} | null>(null);
function openComparison(
  group: AssociationGroup,
  row: AssociationGroup["rows"][number],
  event: MouseEvent,
) {
  if (event.currentTarget instanceof HTMLElement)
    event.currentTarget.focus({ preventScroll: true });
  comparison.value = { group, row };
}
async function toggleFactors() {
  menuOpen.value = !menuOpen.value;
  if (menuOpen.value) {
    await nextTick();
    factorPicker.value
      ?.querySelector<HTMLButtonElement>('.factor-option[aria-pressed="true"]')
      ?.focus();
  }
}
function selectFactor(id: string) {
  factor.value = id;
  menuOpen.value = false;
  factorTrigger.value?.focus({ preventScroll: true });
}
function dismissFactors(event: PointerEvent) {
  if (
    event.target instanceof Node &&
    !factorPicker.value?.contains(event.target)
  )
    menuOpen.value = false;
}
function navigateFactors(event: KeyboardEvent) {
  if (!menuOpen.value) return;
  if (event.key === "Escape") {
    event.preventDefault();
    menuOpen.value = false;
    factorTrigger.value?.focus({ preventScroll: true });
    return;
  }
  if (!["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) return;
  const options = Array.from(
    factorPicker.value?.querySelectorAll<HTMLButtonElement>(".factor-option") ??
      [],
  );
  if (!options.length) return;
  event.preventDefault();
  const current = options.indexOf(document.activeElement as HTMLButtonElement);
  const index =
    event.key === "Home"
      ? 0
      : event.key === "End"
        ? options.length - 1
        : (current + (event.key === "ArrowUp" ? -1 : 1) + options.length) %
          options.length;
  options[index]?.focus();
}
onMounted(() => document.addEventListener("pointerdown", dismissFactors));
onBeforeUnmount(() =>
  document.removeEventListener("pointerdown", dismissFactors),
);
const sections = [
  {
    id: "connection",
    title: "连接与工具",
    groups: [
      "usage",
      "proxy",
      "tools",
      "official",
      "desktopMode",
      "cliMode",
      "thirdMode",
    ],
  },
  {
    id: "account",
    title: "账号与套餐",
    groups: ["plans", "activation", "country", "duration"],
  },
  {
    id: "network",
    title: "网络与 IP",
    groups: ["network", "ipStability", "ipRisk", "exitCountry"],
  },
  {
    id: "sharing",
    title: "共享与使用经历",
    groups: ["shared", "people", "concurrency", "warning", "truncated"],
  },
];
const assigned = new Set(sections.flatMap((section) => section.groups));
const layout = (id: string) => {
  if (id === "tools" || id === "official") return "tool";
  if (
    [
      "usage",
      "desktopMode",
      "cliMode",
      "thirdMode",
      "ipStability",
      "shared",
      "warning",
      "truncated",
    ].includes(id)
  )
    return "pair";
  if (["duration", "ipRisk", "people", "concurrency"].includes(id))
    return "range";
  if (["activation", "country", "exitCountry"].includes(id)) return "list";
  return "tile";
};
const groups = computed(() =>
  associations.value
    .filter((group) => factor.value === "all" || group.id === factor.value)
    .map((group) => ({
      ...group,
      layout: layout(group.id),
      rows: group.rows.filter(
        (row) =>
          !onlyAssociated.value || Math.abs(row.association.phi ?? 0) >= 0.1,
      ),
    }))
    .filter((group) => group.rows.length),
);
const groupedSections = computed(() =>
  [
    ...sections,
    {
      id: "other",
      title: "其他因素",
      groups: associations.value
        .filter((group) => !assigned.has(group.id))
        .map((group) => group.id),
    },
  ]
    .map((section) => ({
      ...section,
      items: section.groups.flatMap((id) =>
        groups.value.filter((group) => group.id === id),
      ),
    }))
    .filter((section) => section.items.length),
);
const optionCount = computed(() =>
  groups.value.reduce((total, group) => total + group.rows.length, 0),
);
</script>

<template>
  <section class="correlations" aria-label="异常相关性">
    <div class="correlation-key">
      <div class="color-key" aria-label="相对异常方向颜色说明">
        <span class="key-more">异常更多</span
        ><span class="key-less">异常更少</span
        ><span class="key-neutral">关联较弱</span
        ><span class="key-unknown">暂无有效比较</span>
      </div>
    </div>
    <div class="card-filter">
      <label
        ><input v-model="onlyAssociated" type="checkbox" />仅显示当前异常组合
        |φ| ≥ 0.10 的选项</label
      >
      <span aria-live="polite"
        >{{ groups.length }} 个因素 · {{ optionCount }} 个选项</span
      >
    </div>
    <section
      v-for="(section, index) in groupedSections"
      :key="section.id"
      class="correlation-section"
      :aria-labelledby="`correlation-${section.id}`"
    >
      <header class="section-heading">
        <span class="section-number">{{
          String(index + 1).padStart(2, "0")
        }}</span>
        <div>
          <h3 :id="`correlation-${section.id}`">
            <button
              type="button"
              class="category-toggle"
              :aria-expanded="!collapsedSections[section.id]"
              :aria-controls="`section-content-${section.id}`"
              @click="
                collapsedSections[section.id] = !collapsedSections[section.id]
              "
            >
              {{ section.title
              }}<span class="collapse-chevron" aria-hidden="true"></span>
            </button>
          </h3>
        </div>
      </header>
      <div
        v-show="!collapsedSections[section.id]"
        :id="`section-content-${section.id}`"
        class="factor-groups"
      >
        <section
          v-for="group in section.items"
          :key="group.id"
          class="factor-group"
          :class="[
            `group-${group.layout}`,
            {
              'group-wide': ['tools', 'plans', 'official', 'duration'].includes(
                group.id,
              ),
            },
          ]"
          :data-factor="group.id"
          :aria-labelledby="`factor-${group.id}`"
        >
          <header class="factor-heading">
            <h4 :id="`factor-${group.id}`">
              <button
                type="button"
                class="category-toggle"
                :aria-expanded="!collapsedGroups[group.id]"
                :aria-controls="`factor-content-${group.id}`"
                @click="collapsedGroups[group.id] = !collapsedGroups[group.id]"
              >
                {{ group.title
                }}<span class="collapse-chevron" aria-hidden="true"></span>
              </button>
            </h4>
            <details class="group-scope">
              <summary>{{ group.total }} 份有效回答 · 比较范围</summary>
              <p>{{ group.scope }}</p>
            </details>
          </header>
          <div
            v-show="!collapsedGroups[group.id]"
            :id="`factor-content-${group.id}`"
            class="factor-options"
          >
            <SurveyAssociationCard
              v-for="row in group.rows"
              :key="row.label"
              :row="row"
              :outcome-label="outcomeLabel"
              :layout="group.layout"
              :icon="group.layout === 'tool' ? toolIcons[row.label] : undefined"
              @inspect="openComparison(group, row, $event)"
            />
          </div>
        </section>
      </div>
    </section>
    <p v-if="!groups.length" class="correlation-empty">
      当前异常组合没有达到此关联强度的选项，可取消筛选查看全部。
    </p>
    <details class="correlation-method">
      <summary>如何理解颜色、相关系数与比较范围？</summary>
      <p>
        φ 的范围是 −1 到
        +1：正值表示选择该项的样本更常报告对应异常，负值表示更少。色阶按 |φ|
        分为较弱（0.10–0.30）、中等（0.30–0.50）、较强（≥ 0.50）；低于 0.10
        使用中性色，无法计算时留白。它不是概率，也不是已通过显著性检验的结论。
      </p>
      <p>
        每个选项与同一道题中“未选择该项”的有效回答比较。多选题按问卷去重；反代工具等条件题只在适用人群内比较，不把没有看到题目的人当成否定回答。任一变量没有变化时不计算系数。
      </p>
      <p>
        模型、降智发现方式和风控识别方式依赖对应异常状态才有回答，缺少正常对照，仅展示样本分布。数值区间按原有顺序排列，不按关联大小重排。
      </p>
      <p>
        相关性不等于因果，也不表示时间先后。未控制套餐、使用强度等混杂因素，统计单位为问卷，不等于独立用户。少量样本也可能产生较大系数；绿色不代表安全，中性色不代表已证明无关联。
      </p>
    </details>
    <div v-show="!comparison" class="correlation-toolbar">
      <div class="outcome-picker">
        <div
          class="outcome-switch"
          role="group"
          aria-label="异常类型（多选并集，可全部取消）"
        >
          <button
            v-for="item in surveyOutcomes"
            :key="item.id"
            type="button"
            :aria-pressed="selectedOutcomes.includes(item.id)"
            @click="toggleOutcome(item.id)"
          >
            <span class="outcome-check" aria-hidden="true">{{
              selectedOutcomes.includes(item.id) ? "✓" : ""
            }}</span
            >{{ item.label }}
          </button>
        </div>
        <span
          v-if="!selectedOutcomes.length"
          class="outcome-fallback"
          role="status"
          >按{{ outcomeLabel }}显示</span
        >
      </div>
      <div
        ref="factorPicker"
        class="factor-picker"
        @keydown="navigateFactors"
        @focusout="
          !factorPicker?.contains($event.relatedTarget as Node) &&
          (menuOpen = false)
        "
      >
        <button
          ref="factorTrigger"
          type="button"
          class="factor-trigger"
          :aria-expanded="menuOpen"
          aria-controls="correlation-factor-menu"
          @click="toggleFactors"
        >
          <span class="factor-trigger-label">比较因素</span
          ><strong>{{ factorLabel }}</strong
          ><span
            class="factor-chevron"
            :class="{ 'is-open': menuOpen }"
            aria-hidden="true"
            >⌃</span
          >
        </button>
        <div
          v-if="menuOpen"
          id="correlation-factor-menu"
          class="factor-menu"
          role="group"
          aria-label="比较因素"
        >
          <button
            type="button"
            class="factor-option"
            :aria-pressed="factor === 'all'"
            @click="selectFactor('all')"
          >
            <span>全部因素</span
            ><span v-if="factor === 'all'" aria-hidden="true">✓</span>
          </button>
          <button
            v-for="group in associations"
            :key="group.id"
            type="button"
            class="factor-option"
            :aria-pressed="factor === group.id"
            @click="selectFactor(group.id)"
          >
            <span>{{ group.title }}</span
            ><span v-if="factor === group.id" aria-hidden="true">✓</span>
          </button>
        </div>
      </div>
    </div>
    <SurveyAssociationDialog
      v-if="comparison"
      :row="comparison.row"
      :title="comparison.group.title"
      :scope="comparison.group.scope"
      :outcome-label="outcomeLabel"
      @close="comparison = null"
    />
  </section>
</template>

<style scoped src="./SurveyCorrelationPanel.css"></style>
