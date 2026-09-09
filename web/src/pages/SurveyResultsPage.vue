<script setup lang="ts">
import { computed, defineAsyncComponent, onBeforeUnmount, onMounted, ref } from "vue";
import SurveyFactorCard from "../components/survey/SurveyFactorCard.vue";
import SurveyCorrelationPanel from "../components/survey/SurveyCorrelationPanel.vue";
import UsagePattern from "../components/survey/UsagePattern.vue";
import { surveyRequest, type SurveySummary } from "../data/surveyApi";
import { calculateSurveyStatistics } from "../data/surveyStatistics";
import { loadSubmissionMarker, submittedBefore } from "../data/surveyParticipation";

const SurveyShareDialog = defineAsyncComponent(
  () => import("../components/survey/SurveyShareDialog.vue"),
);
const shareOpen = ref(false);
const shareOutcomeMask = ref(1);

const summary = ref<SurveySummary | null>(null);
const statistics = computed(() =>
  summary.value ? calculateSurveyStatistics(summary.value) : null,
);
const loading = ref(false);
const error = ref("");
const analysisView = ref<"distribution" | "correlation">("correlation");
async function loadStatistics() {
  if (loading.value) return;
  loading.value = true;
  error.value = "";
  try {
    const result = await surveyRequest<SurveySummary>("statistics");
    if (result.version !== 2) throw new Error("统计数据格式不匹配，请更新服务后刷新页面。");
    summary.value = result;
  } catch (cause) {
    summary.value = null;
    error.value = cause instanceof Error ? cause.message : "统计加载失败，请重试。";
  } finally {
    loading.value = false;
  }
}
onMounted(() => {
  loadSubmissionMarker();
  window.addEventListener("storage", loadSubmissionMarker);
  void loadStatistics();
});
onBeforeUnmount(() => window.removeEventListener("storage", loadSubmissionMarker));
const metrics = computed(() => [
  {
    label: "问卷样本",
    value: statistics.value?.total,
    note: "统计单位为问卷，不等于独立用户",
    tone: "plain",
  },
  {
    label: "报告降智",
    value: statistics.value?.degraded,
    note: "可与封号、风控同时出现",
    tone: "violet",
  },
  {
    label: "报告封号",
    value: statistics.value?.banned,
    note: "可与降智、风控同时出现",
    tone: "peach",
  },
  {
    label: "风控（限流）",
    value: statistics.value?.limited,
    note: "可与降智、封号同时出现",
    tone: "sun",
  },
  { label: "报告正常", value: statistics.value?.normal, note: "填写时自述账号正常", tone: "mint" },
]);
const statuses = computed(() => statistics.value?.statuses ?? []);
const percentage = (count: number) =>
  statistics.value?.total ? ((100 * count) / statistics.value.total).toFixed(1) : "0.0";
const formatTime = (value: string) => new Date(value).toLocaleString("zh-CN", { hour12: false });
</script>

<template>
  <div class="survey-results">
    <a class="back" href="#/">← 全部研究</a>
    <header class="results-heading">
      <div>
        <span class="pill plain">研究 02 · 统计结果</span>
        <h1>ChatGPT 套餐<br />降智封号统计</h1>
        <p>看看不同账号情况、模型与使用方式，在样本中如何分布。</p>
      </div>
      <a
        class="participation-card"
        :class="{ 'is-submitted': submittedBefore }"
        href="#/studies/chatgpt-account-survey"
      >
        <template v-if="submittedBefore">
          <span>您已填写了问卷，</span>
          <strong>非常感谢您的填写。</strong>
          <span>如果还有其他账号的情况，</span>
          <span>可以点此继续填写 <b aria-hidden="true">↗</b></span>
        </template>
        <template v-else>
          <span>你还未参与调查统计，</span>
          <strong>可以点此参与问卷吗，</strong>
          <span>求求你了 <b aria-hidden="true">↗</b></span>
        </template>
      </a>
    </header>
    <nav class="research-tabs" aria-label="研究二页面">
      <a href="#/studies/chatgpt-account-survey">调查问卷</a>
      <a href="#/studies/chatgpt-account-survey/results" aria-current="page">统计结果</a>
    </nav>

    <div class="results-mode" role="status">
      <div>
        <strong>{{
          loading
            ? "正在加载统计…"
            : error
              ? "统计暂时不可用"
              : statistics?.total
                ? "问卷统计结果"
                : "尚未收到问卷"
        }}</strong>
        <p>{{ error || "展示已提交问卷的汇总结果。你可以不填问卷直接浏览。" }}</p>
      </div>
      <button class="button small" type="button" :disabled="loading" @click="loadStatistics">
        {{ error ? "重试" : "刷新统计" }}
      </button>
    </div>
    <p v-if="statistics" class="results-caption">
      统计范围：{{ statistics.total }} 份问卷<template v-if="statistics.total"
        >，提交编号 {{ statistics.range.firstSubmissionId }}–{{
          statistics.range.lastSubmissionId
        }}</template
      >。
      <template v-if="statistics.range.firstSubmittedAt && statistics.range.lastSubmittedAt">
        已知提交时间：{{ formatTime(statistics.range.firstSubmittedAt) }} 至
        {{ formatTime(statistics.range.lastSubmittedAt) }}（本地时间）。
      </template>
      <template v-if="statistics.range.unknownTimeCount"
        >{{ statistics.range.unknownTimeCount }} 份历史问卷的提交时间未知。</template
      >
    </p>

    <section class="result-metrics" aria-label="问卷统计概览">
      <article v-for="metric in metrics" :key="metric.label" :class="metric.tone">
        <span>{{ metric.label }}</span>
        <strong>{{ metric.value ?? "—" }}<small>份</small></strong>
        <p>{{ metric.note }}</p>
      </article>
    </section>
    <p class="results-caption">各异常状态可能重叠，不能直接相加作为异常总数。“—”表示无可用统计。</p>

    <section class="result-panel status-panel">
      <div class="result-panel-heading">
        <div>
          <h2>账号情况分布</h2>
          <p>按状态组合统计，每份问卷归入一种组合。</p>
        </div>
        <span class="pill plain">{{ statistics ? `${statistics.total} 份问卷` : "等待统计" }}</span>
      </div>
      <template v-if="statistics && statistics.total > 0">
        <div class="status-strip" aria-hidden="true">
          <span
            v-for="item in statuses.filter((item) => item.count > 0)"
            :key="item.label"
            :class="item.tone"
            :style="{ flexGrow: item.count }"
          ></span>
        </div>
        <ul class="status-legend">
          <li v-for="item in statuses" :key="item.label">
            <span class="status-dot" :class="item.tone"></span>
            <div>
              <span>{{ item.label }}</span
              ><strong
                >{{ item.count }} <small>份 · {{ percentage(item.count) }}%</small></strong
              >
            </div>
          </li>
        </ul>
      </template>
      <div v-else class="results-empty">
        <strong>{{ statistics ? "还没有问卷样本" : "暂无可用统计" }}</strong>
        <p>
          {{
            statistics
              ? "提交问卷后，这里会展示账号状态的分布。"
              : "请等待加载完成，或使用上方按钮重试。"
          }}
        </p>
      </div>
    </section>

    <div class="analysis-switch" role="group" aria-label="统计分析板块">
      <button
        type="button"
        :aria-pressed="analysisView === 'correlation'"
        @click="analysisView = 'correlation'"
      >
        异常相关性
      </button>
      <button
        type="button"
        :aria-pressed="analysisView === 'distribution'"
        @click="analysisView = 'distribution'"
      >
        异常样本分布
      </button>
      <button
        class="share-button"
        type="button"
        :disabled="!summary"
        aria-haspopup="dialog"
        @click="shareOpen = true"
      >
        分享统计 <span aria-hidden="true">↗</span>
      </button>
    </div>
    <SurveyCorrelationPanel
      v-if="summary"
      v-show="analysisView === 'correlation'"
      :factors="summary.factors"
      @outcome-change="shareOutcomeMask = $event"
    />
    <section
      v-show="analysisView === 'distribution'"
      class="factors-section"
      aria-labelledby="factors-heading"
    >
      <div class="factors-heading">
        <h2 id="factors-heading">使用情况与异常样本分布</h2>
        <p>
          每张卡片可切换异常状态。百分比表示该状态的有效回答中各选项所占比例，不能据此认定原因。
        </p>
      </div>
      <div class="result-distributions">
        <SurveyFactorCard
          v-for="distribution in statistics?.factors ?? []"
          :key="distribution.id"
          :distribution="distribution"
        />
      </div>
      <section v-if="statistics" class="result-panel">
        <div class="result-panel-heading">
          <div>
            <h2>平时使用规律</h2>
            <p>各小时使用时间占比的平均值。</p>
          </div>
          <span class="pill plain">{{ statistics.usagePattern.total }} 份有效回答</span>
        </div>
        <UsagePattern
          v-if="statistics.usagePattern.total"
          :model-value="statistics.usagePattern.levels"
          readonly
        />
        <p v-else>暂无使用规律数据。</p>
      </section>
    </section>

    <section class="results-reading">
      <h2>这些数字应当怎样理解？</h2>
      <div>
        <article>
          <h3>样本分布，不是总体发生率</h3>
          <p>
            自愿填写的问卷存在选择偏差，异常用户可能更愿意参与。样本占比不能直接代表所有账号的风险。
          </p>
        </article>
        <article>
          <h3>共同出现，不代表因果</h3>
          <p>
            工具、套餐与异常一起出现，不等于它们导致异常。“降智”也是填写者的观察判断，而非已确认的能力变化。
          </p>
        </article>
      </div>
    </section>
    <div class="results-bottom">
      <p>不填写问卷，也可以查看统计结果。</p>
      <a class="button" href="#/studies/chatgpt-account-survey">返回调查问卷 →</a>
    </div>
    <SurveyShareDialog
      v-if="shareOpen && summary"
      :summary="summary"
      :outcome-mask="shareOutcomeMask"
      @close="shareOpen = false"
    />
  </div>
</template>

<style scoped src="./SurveyResultsPage.css"></style>
