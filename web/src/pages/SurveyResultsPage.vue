<script setup lang="ts">
import { computed, ref } from "vue";
import SurveyFactorCard from "../components/survey/SurveyFactorCard.vue";
import SurveyCorrelationPanel from "../components/survey/SurveyCorrelationPanel.vue";
import {
  surveyExampleOverview as example,
  surveyExampleFactors,
} from "../data/surveyExampleStatistics";

const showExample = ref(false);
const analysisView = ref<"distribution" | "correlation">("distribution");
const affected = example.statuses[1]!.count + example.statuses[3]!.count;
const banned = example.statuses[2]!.count + example.statuses[3]!.count;
const metrics = computed(() => [
  {
    label: "问卷样本",
    value: example.total,
    note: "统计单位为问卷，不等于独立用户",
    tone: "plain",
  },
  { label: "报告降智", value: affected, note: "包含「降智并封号」", tone: "violet" },
  { label: "报告封号", value: banned, note: "包含「降智并封号」", tone: "peach" },
  {
    label: "报告正常",
    value: example.statuses[0]!.count,
    note: "填写时自述账号正常",
    tone: "mint",
  },
]);
const percentage = (count: number, total: number) => ((count / total) * 100).toFixed(1);
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

    <div class="results-mode" :class="{ example: showExample }" role="status">
      <div>
        <strong>{{
          showExample ? "示例数据 · 不是真实调查结果" : "统计功能预览 · 尚未接入真实数据"
        }}</strong>
        <p>
          {{
            showExample
              ? "以下为 200 份虚构问卷的展示示例，仅用于预览布局，不参与研究，也不包含你刚填写的内容。"
              : "目前没有连接问卷提交与统计服务。你可以不填问卷直接浏览，或打开示例查看图表效果。"
          }}
        </p>
      </div>
      <button
        class="button small"
        type="button"
        :aria-pressed="showExample"
        @click="showExample = !showExample"
      >
        {{ showExample ? "返回未接入状态" : "查看示例效果" }}
      </button>
    </div>

    <section class="result-metrics" aria-label="问卷统计概览">
      <article v-for="metric in metrics" :key="metric.label" :class="metric.tone">
        <span>{{ metric.label }}</span>
        <strong>{{ showExample ? metric.value : "—" }}<small>份</small></strong>
        <p>{{ metric.note }}</p>
      </article>
    </section>
    <p class="results-caption">
      {{
        showExample
          ? "示例口径：报告降智与报告封号有重叠，不能相加作为异常总数。"
          : "“—”表示尚无可用统计，不代表样本数为零。"
      }}
    </p>

    <section class="result-panel status-panel">
      <div class="result-panel-heading">
        <div>
          <h2>账号情况分布</h2>
          <p>四种状态互斥，每份问卷归入一种。</p>
        </div>
        <span class="pill plain">{{ showExample ? "200 份示例问卷" : "等待真实统计" }}</span>
      </div>
      <template v-if="showExample">
        <div class="status-strip" aria-hidden="true">
          <span
            v-for="item in example.statuses"
            :key="item.label"
            :class="item.tone"
            :style="{ flexGrow: item.count }"
          ></span>
        </div>
        <ul class="status-legend">
          <li v-for="item in example.statuses" :key="item.label">
            <span class="status-dot" :class="item.tone"></span>
            <div>
              <span>{{ item.label }}</span
              ><strong
                >{{ item.count }}
                <small>份 · {{ percentage(item.count, example.total) }}%</small></strong
              >
            </div>
          </li>
        </ul>
      </template>
      <div v-else class="results-empty">
        <strong>统计将在数据接入后呈现</strong>
        <p>这里会展示正常、降智、封号，以及降智并封号的样本分布。</p>
        <button class="text-link" type="button" @click="showExample = true">先看示例图表 →</button>
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
        降智封号相关性
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
          每张卡片可独立切换降智 /
          封号。百分比表示该状态的有效回答中各选项所占比例，不能据此认定原因。
        </p>
      </div>
      <div class="result-distributions">
        <SurveyFactorCard
          v-for="distribution in surveyExampleFactors"
          :key="distribution.id"
          :distribution="distribution"
          :show-example="showExample"
        />
      </div>
    </section>

    <SurveyCorrelationPanel v-show="analysisView === 'correlation'" :show-example="showExample" />

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
