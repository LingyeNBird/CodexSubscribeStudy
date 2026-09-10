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
import {
  PhCaretDown,
  PhCheckSquare,
  PhCircle,
  PhRadioButton,
  PhSquare,
} from "@phosphor-icons/vue";
import SurveyAssociationCard from "./SurveyAssociationCard.vue";
import { toolIcons } from "../icons/toolIcons";
import {
  querySurveyStatistics,
  type SurveyPrerequisite,
  type SurveySummary,
  type SurveyOutcome,
} from "../../data/surveyApi";
import {
  calculateAssociations,
  effectiveOutcomeMask,
  surveyOutcomes,
  type AssociationGroup,
} from "../../data/surveyStatistics";
const SurveyAssociationDialog = defineAsyncComponent(
  () => import("./SurveyAssociationDialog.vue"),
);
const props = defineProps<{ summary: SurveySummary }>();
const emit = defineEmits<{
  outcomeChange: [mask: number];
  scopeChange: [summary: SurveySummary];
}>();
const prerequisites = ref<Record<string, string[]>>({});
const scopedSummary = shallowRef<SurveySummary | null>(null);
const premiseLoading = ref(false);
const premiseError = ref("");
const premiseMenuOpen = ref(false);
const premisePicker = ref<HTMLElement | null>(null);
const contextMenu = shallowRef<{
  group: AssociationGroup;
  row: AssociationGroup["rows"][number];
  x: number;
  y: number;
} | null>(null);
let queryController: AbortController | null = null;
const activeSummary = computed(() => scopedSummary.value ?? props.summary);
const prerequisiteQuery = computed<SurveyPrerequisite[]>(() =>
  Object.entries(prerequisites.value)
    .filter(([, options]) => options.length)
    .map(([factor, options]) => ({ factor, options: [...options] })),
);
const prerequisiteCount = computed(() =>
  prerequisiteQuery.value.reduce(
    (total, condition) => total + condition.options.length,
    0,
  ),
);
const prerequisiteLabels = computed(() =>
  prerequisiteQuery.value.flatMap((condition) => {
    const title =
      props.summary.factors.find((factor) => factor.id === condition.factor)
        ?.title ?? condition.factor;
    return condition.options.map((option) => ({
      factor: condition.factor,
      option,
      title,
    }));
  }),
);
watch(
  () =>
    [prerequisiteQuery.value, props.summary.range.lastSubmissionId] as const,
  async ([conditions]) => {
    queryController?.abort();
    premiseError.value = "";
    if (!conditions.length) {
      scopedSummary.value = null;
      premiseLoading.value = false;
      emit("scopeChange", props.summary);
      return;
    }
    const controller = new AbortController();
    queryController = controller;
    premiseLoading.value = true;
    try {
      const result = await querySurveyStatistics(conditions, controller.signal);
      if (controller.signal.aborted) return;
      scopedSummary.value = result;
      emit("scopeChange", result);
    } catch (cause) {
      if (!controller.signal.aborted)
        premiseError.value =
          cause instanceof Error ? cause.message : "前提统计加载失败。";
    } finally {
      if (queryController === controller) premiseLoading.value = false;
    }
  },
  { deep: true },
);
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
  calculateAssociations(activeSummary.value.factors, outcomeMask.value),
);
const baseAssociations = computed(
  () =>
    new Map(
      calculateAssociations(props.summary.factors, outcomeMask.value).map(
        (group) => [group.id, group],
      ),
    ),
);
function premiseTone(group: string, option: string) {
  if (!hasPrerequisite(group)) return undefined;
  return baseAssociations.value
    .get(group)
    ?.rows.find((row) => row.label === option)?.association.phi;
}
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
function openPremiseMenu(
  group: AssociationGroup,
  row: AssociationGroup["rows"][number],
  event: MouseEvent,
) {
  contextMenu.value = {
    group,
    row,
    x: Math.max(8, Math.min(event.clientX, window.innerWidth - 190)),
    y: Math.max(8, Math.min(event.clientY, window.innerHeight - 90)),
  };
}
function isPrerequisite(group: string, option: string) {
  return prerequisites.value[group]?.includes(option) ?? false;
}
function hasPrerequisite(group: string) {
  return (prerequisites.value[group]?.length ?? 0) > 0;
}
function addPrerequisite(group: string, option: string) {
  const options = prerequisites.value[group] ?? [];
  if (!options.includes(option))
    prerequisites.value = {
      ...prerequisites.value,
      [group]: [...options, option],
    };
  contextMenu.value = null;
}
function removePrerequisite(group: string, option: string) {
  const options = (prerequisites.value[group] ?? []).filter(
    (value) => value !== option,
  );
  const next = { ...prerequisites.value };
  if (options.length) next[group] = options;
  else delete next[group];
  prerequisites.value = next;
  contextMenu.value = null;
}
function clearPrerequisites() {
  prerequisites.value = {};
  premiseMenuOpen.value = false;
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
function dismissMenus(event: PointerEvent) {
  if (!(event.target instanceof Node)) return;
  if (!factorPicker.value?.contains(event.target)) menuOpen.value = false;
  if (!premisePicker.value?.contains(event.target))
    premiseMenuOpen.value = false;
  contextMenu.value = null;
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
onMounted(() => document.addEventListener("pointerdown", dismissMenus));
onBeforeUnmount(() => {
  queryController?.abort();
  document.removeEventListener("pointerdown", dismissMenus);
});
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
    <div
      v-if="prerequisiteCount || premiseError"
      class="premise-status"
      role="status"
    >
      <span v-if="premiseError">{{ premiseError }}</span>
      <span v-else>
        已按 {{ prerequisiteCount }} 个前提筛选 ·
        {{ activeSummary.statuses.reduce((total, count) => total + count, 0) }}
        份问卷
        <template v-if="premiseLoading"> · 正在更新</template>
      </span>
    </div>
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
              :prerequisite="isPrerequisite(group.id, row.label)"
              :muted="
                hasPrerequisite(group.id) &&
                !isPrerequisite(group.id, row.label)
              "
              :tone-phi="premiseTone(group.id, row.label)"
              @premise="openPremiseMenu(group, row, $event)"
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
    <div
      v-if="contextMenu"
      class="premise-context-menu"
      role="menu"
      :style="{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px` }"
      @pointerdown.stop
    >
      <button
        type="button"
        role="menuitem"
        @click="
          isPrerequisite(contextMenu.group.id, contextMenu.row.label)
            ? removePrerequisite(contextMenu.group.id, contextMenu.row.label)
            : addPrerequisite(contextMenu.group.id, contextMenu.row.label)
        "
      >
        {{
          isPrerequisite(contextMenu.group.id, contextMenu.row.label)
            ? "移除前提"
            : "设为前提"
        }}
      </button>
    </div>
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
            <component
              :is="
                selectedOutcomes.includes(item.id) ? PhCheckSquare : PhSquare
              "
              class="outcome-check"
              :weight="selectedOutcomes.includes(item.id) ? 'fill' : 'regular'"
              :size="16"
              aria-hidden="true"
            />
            {{ item.label }}
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
        v-if="prerequisiteCount"
        ref="premisePicker"
        class="factor-picker premise-picker"
        @pointerdown.stop
      >
        <button
          type="button"
          class="factor-trigger premise-trigger"
          :aria-expanded="premiseMenuOpen"
          aria-controls="correlation-premise-menu"
          @click="premiseMenuOpen = !premiseMenuOpen"
        >
          <span class="factor-trigger-label">前提</span>
          <strong>{{ prerequisiteCount }} 项</strong>
          <PhCaretDown
            class="factor-chevron"
            :class="{ 'is-open': premiseMenuOpen }"
            :size="16"
            weight="bold"
            aria-hidden="true"
          />
        </button>
        <div
          v-if="premiseMenuOpen"
          id="correlation-premise-menu"
          class="factor-menu premise-menu"
          aria-label="当前前提"
        >
          <div
            v-for="item in prerequisiteLabels"
            :key="`${item.factor}:${item.option}`"
            class="premise-option"
          >
            <span
              ><small>{{ item.title }}</small
              ><strong>{{ item.option }}</strong></span
            >
            <button
              type="button"
              :aria-label="`移除前提${item.title}${item.option}`"
              @click="removePrerequisite(item.factor, item.option)"
            >
              ×
            </button>
          </div>
          <button
            type="button"
            class="premise-clear"
            @click="clearPrerequisites"
          >
            清空全部前提
          </button>
        </div>
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
          ><PhCaretDown
            class="factor-chevron"
            :class="{ 'is-open': menuOpen }"
            :size="16"
            weight="bold"
            aria-hidden="true"
          />
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
            <span>全部因素</span>
            <component
              :is="factor === 'all' ? PhRadioButton : PhCircle"
              class="factor-option-icon"
              :weight="factor === 'all' ? 'fill' : 'regular'"
              :size="16"
              aria-hidden="true"
            />
          </button>
          <button
            v-for="group in associations"
            :key="group.id"
            type="button"
            class="factor-option"
            :aria-pressed="factor === group.id"
            @click="selectFactor(group.id)"
          >
            <span>{{ group.title }}</span>
            <component
              :is="factor === group.id ? PhRadioButton : PhCircle"
              class="factor-option-icon"
              :weight="factor === group.id ? 'fill' : 'regular'"
              :size="16"
              aria-hidden="true"
            />
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
