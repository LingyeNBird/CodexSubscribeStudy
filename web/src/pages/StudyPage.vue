<script setup lang="ts">
import { computed } from "vue";
import type { Study, Cause } from "../types";
import { labels, n } from "../studyPresentation";
const props = defineProps<{
  study: Study | null;
  loading: boolean;
  refreshing: boolean;
  lastRefresh: string;
}>();
const emit = defineEmits<{ refresh: [] }>();
const componentNames = ["普通输入", "缓存创建", "缓存读取", "输出"];
const componentLabel: Record<string, string> = {
  input: "普通输入",
  cache_creation: "缓存创建",
  cache_read: "缓存读取",
  output: "输出",
};
const descriptions: Record<string, string> = {
  unchanged: "保留原始分项价格，不增加额外倍率。",
  global: "所有 GPT-6 分项使用同一个倍率。",
  cache_read: "只改变缓存读取成本，其他分项保持不变。",
  cache_creation: "只改变缓存创建成本。",
  output: "只改变输出成本。",
  input: "只改变普通输入成本。",
  mixed: "两项或更多分项采用不同倍率。",
};
const ordered = computed(() =>
  [...(props.study?.causes ?? [])].sort((a, b) => (b.support ?? -1) - (a.support ?? -1)),
);
const leading = computed(() => ordered.value[0]);
const factorCause = defineModel<string>("factorCause", { required: true });
const selectedCause = computed(() => props.study?.causes.find((c) => c.id === factorCause.value));
const qualityEntries = computed(() =>
  Object.entries(props.study?.quality ?? {}).filter(([, v]) => v > 0),
);
const money = (value: number | undefined) => "$" + n(value);
const pct = (cause?: Cause) =>
  cause?.support == null ? "尚无区分信息" : (cause.support * 100).toFixed(1) + "%";
</script>

<template>
  <a class="back" href="#/">← 全部研究</a>
  <section class="detail-heading">
    <div>
      <span class="eyebrow">STUDY 01 / OPEN INVESTIGATION</span>
      <h1>GPT-6 额度异常归因</h1>
      <p>固定其他计费因素，中心联合比较四个成本分项。</p>
    </div>
    <div class="refresh-block">
      <span class="pill">{{ labels[study?.state ?? "no_data"] }}</span
      ><button class="button small" :disabled="refreshing" @click="emit('refresh')">
        {{ refreshing ? "读取中…" : "刷新统计 ↻" }}</button
      ><small v-if="lastRefresh">页面更新 {{ lastRefresh }}</small>
    </div>
  </section>
  <div v-if="loading" class="loading-panel" role="status">正在读取公共统计…</div>
  <template v-else-if="study">
    <section class="metrics" aria-label="科研数据总量">
      <article>
        <span>去标识化贡献安装</span
        ><strong>{{ n(study.totals.contributors) }}<small>个</small></strong>
        <p>{{ n(study.totals.batches) }} 个持久批次，无人数准入门槛</p>
      </article>
      <article class="mint">
        <span>贡献请求总量</span><strong>{{ n(study.totals.requests) }}<small>次</small></strong>
        <p>
          GPT-6 {{ n(study.totals.gpt6_requests) }} / 其他
          {{ n(study.totals.other_requests) }}
        </p>
      </article>
      <article class="violet">
        <span>原始标准成本</span><strong>{{ money(study.totals.raw_usd) }}</strong>
        <p>其中 GPT-6 {{ money(study.totals.gpt6_raw_usd) }}</p>
      </article>
      <article class="peach">
        <span>观测额度消耗</span
        ><strong>{{ n(study.totals.quota_points) }}<small>百分点</small></strong>
        <p>
          {{ n(study.totals.intervals) }} 个原始区间 / {{ n(study.totals.contrasts) }} 个容量内对比
        </p>
      </article>
    </section>
    <p class="metric-caption">
      不同批次分别保留，同批次只计已接收报告。安装不是已验证的自然人；跨账号百分点不代表同一个额度池。
    </p>
    <section
      v-if="study.state === 'no_data' || study.state === 'uninformative'"
      class="empty-evidence"
    >
      <span class="empty-symbol" aria-hidden="true">…</span>
      <div>
        <h2>
          {{ study.state === "no_data" ? "第一份证据，还在路上。" : "贡献已收到，等待互补信息。" }}
        </h2>
        <p>一条请求也能贡献，无需满两个周期或 200 次请求。没有区分信息时不把先验偏好冒充发现。</p>
        <a class="text-link" href="#/privacy">从 Sub2Pool 自愿参与 →</a>
      </div>
    </section>
    <div v-else class="banner" :class="study.state === 'sensitive' ? 'warning' : 'mint'">
      <strong>当前支持较多：{{ leading?.label }}（{{ pct(leading) }}）</strong
      ><span>{{ study.confidence_meaning }}</span>
    </div>
    <section class="analysis-grid">
      <article class="panel ranking">
        <div class="panel-head">
          <div>
            <span class="eyebrow">COMPARE, NOT CONFIRM</span>
            <h2>各个原因的支持度</h2>
          </div>
          <span class="pill plain">7 类假设</span>
        </div>
        <p class="panel-intro">
          共同倍率参数的证据先汇合，再加入一次先验；不是平均各客户端的猜测百分比。
        </p>
        <div class="causes">
          <div v-for="(cause, index) in ordered" :key="cause.id" class="cause">
            <div class="cause-heading">
              <span class="rank">{{ String(index + 1).padStart(2, "0") }}</span
              ><strong>{{ cause.label }}</strong
              ><b>{{ pct(cause) }}</b>
            </div>
            <div class="bar-track" :class="{ 'bar-empty': cause.support === null }">
              <div
                :style="{ width: `${(cause.support ?? 0) * 100}%` }"
                :class="'bar-' + (index % 4)"
              ></div>
            </div>
            <p>{{ descriptions[cause.id] }}</p>
          </div>
        </div>
        <p class="fineprint">
          这是固定假设下的条件证据支持度，不是已经证明官方计费机制的概率，也不会自动修改计价。
        </p>
      </article>
      <aside class="side-panels">
        <article class="panel lavender">
          <span class="eyebrow">EVIDENCE QUALITY</span>
          <h2>证据能回答多少？</h2>
          <p class="panel-intro">局部信息不足不妨碍贡献；中心检查信息是否互补。</p>
          <div class="quality-row">
            <span>全局可区分方向</span><strong>{{ study.information_rank }} / 4</strong>
          </div>
          <div class="quality-row">
            <span>最大来源信息占比</span
            ><strong>{{ (study.maximum_source_information_share * 100).toFixed(1) }}%</strong>
          </div>
          <div class="divider"></div>
          <div v-for="[status, count] in qualityEntries" :key="status" class="quality-row">
            <span>{{ labels[status] ?? status }}</span
            ><b>{{ count }}</b>
          </div>
          <p v-for="warning in study.warnings" :key="warning" class="fineprint">
            {{ warning }}
          </p>
        </article>
        <article class="panel factor-panel">
          <span class="eyebrow">CONDITIONAL PARAMETERS</span>
          <h2>候选倍率速览</h2>
          <label for="cause-select" class="subtle">选择一个解释族</label
          ><select id="cause-select" v-model="factorCause">
            <option v-for="cause in study.causes" :key="cause.id" :value="cause.id">
              {{ cause.label }}
            </option>
          </select>
          <div class="factor-grid">
            <div v-for="(value, index) in selectedCause?.factor_estimates" :key="index">
              <span>{{ componentNames[index] }}</span
              ><strong>{{ selectedCause?.support == null ? "—" : `× ${value.toFixed(2)}` }}</strong>
            </div>
          </div>
          <p class="fineprint">
            联合证据在该解释族内的倍率均值，不是当前网关价格。没有读取或使用 PF／平均恒定容量估值。
          </p>
        </article>
      </aside>
    </section>
    <details class="panel score-details">
      <summary>查看倍率范围与原始事实敏感性</summary>
      <p>
        90%
        范围来自离散网格的条件证据分布，不保证真实机制的覆盖率。三种容量波动假设仅使用原始事实，不是引入容量估值。
      </p>
      <div class="table-scroll">
        <table>
          <thead>
            <tr>
              <th>分项</th>
              <th>均值</th>
              <th>90% 条件范围</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in study.parameters" :key="p.name">
              <td>{{ componentLabel[p.name] }}</td>
              <td>{{ p.mean.toFixed(3) }}</td>
              <td>{{ p.low }} ～ {{ p.high }}</td>
            </tr>
          </tbody>
        </table>
        <table v-if="study.drift_support.length">
          <thead>
            <tr>
              <th>解释</th>
              <th>无漂移</th>
              <th>常规漂移</th>
              <th>较强漂移</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(cause, i) in study.causes" :key="cause.id">
              <td>{{ cause.label }}</td>
              <td v-for="(curve, j) in study.drift_support" :key="j">
                {{ (curve[i] * 100).toFixed(1) }}%
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <p v-if="study.gpt6_quota">
        候选下 GPT-6 额度归属：{{ n(study.gpt6_quota.mean) }} 百分点（{{
          n(study.gpt6_quota.low)
        }}
        ～ {{ n(study.gpt6_quota.high) }}）。这是归因结果，不能反过来当作真实扣减标签。
      </p>
    </details>
    <div class="method-strip">
      <div>
        <strong>让结果可以被质疑，也可以被复现。</strong>
        <p>方法 {{ study.method }} · 数据更新 {{ study.updated_at || "暂无" }}</p>
      </div>
      <a class="button" href="#/method">阅读方法说明 ↗</a>
    </div>
  </template>
</template>

<style scoped src="./StudyPage.css"></style>
