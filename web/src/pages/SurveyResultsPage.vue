<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import SurveyFactorCard from "../components/survey/SurveyFactorCard.vue";
import SurveyCorrelationPanel from "../components/survey/SurveyCorrelationPanel.vue";
import { surveyRequest, type SurveyStatistics } from "../data/surveyApi";

const statistics = ref<SurveyStatistics | null>(null);
const loading = ref(false);
const error = ref("");
const analysisView = ref<"distribution" | "correlation">("distribution");
async function loadStatistics() {
  if (loading.value) return;
  loading.value = true;
  error.value = "";
  try {
    statistics.value = await surveyRequest<SurveyStatistics>("statistics");
  } catch (cause) {
    statistics.value = null;
    error.value = cause instanceof Error ? cause.message : "统计加载失败，请重试。";
  } finally {
    loading.value = false;
  }
}
onMounted(loadStatistics);
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
      <a class="button primary" href="#/studies/chatgpt-account-survey">填写问卷 ↗</a>
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
            v-for="item in statuses"
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
        :aria-pressed="analysisView === 'distribution'"
        @click="analysisView = 'distribution'"
      >
        异常样本分布
      </button>
      <button
        type="button"
        :aria-pressed="analysisView === 'correlation'"
        @click="analysisView = 'correlation'"
      >
        异常相关性
      </button>
    </div>
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
    </section>

    <SurveyCorrelationPanel
      v-if="statistics"
      v-show="analysisView === 'correlation'"
      :associations="statistics.associations"
      :outcome-association="statistics.outcomeAssociation"
    />

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
  </div>
</template>

<style scoped src="./SurveyResultsPage.css"></style>
