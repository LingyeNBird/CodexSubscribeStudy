<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import type { Study, Cause } from "./types";

const study = ref<Study | null>(null);
const route = ref(location.hash.slice(1) || "/");
const loading = ref(true);
const refreshing = ref(false);
const error = ref("");
const lastRefresh = ref("");
const controller = new AbortController();
let timer: ReturnType<typeof setInterval> | undefined;
const detail = computed(() => route.value === "/studies/gpt6-components");
const labels: Record<string, string> = {
  no_data: "等待第一份证据", uninformative: "贡献已收到 · 证据待积累",
  conditional: "联合证据已更新", sensitive: "对容量波动假设敏感",
  missing_snapshot: "缺少额度快照", capture_gap: "请求覆盖有缺口",
  missing_components: "缺少原始成本分项", unknown_control: "控制因素事实未知",
  invalid_fact: "原始事实不一致", reset_or_saturation: "重置或饱和观测",
  zero_progress: "零显示增量", external_usage_uncontrolled: "用量覆盖未确认",
  archived_source: "原始来源已归档", resource_limit: "等待资源处理",
};
const componentNames = ["普通输入", "缓存创建", "缓存读取", "输出"];
const componentLabel: Record<string, string> = { input: "普通输入", cache_creation: "缓存创建", cache_read: "缓存读取", output: "输出" };
const descriptions: Record<string, string> = {
  unchanged: "保留原始分项价格，不增加额外倍率。", global: "所有 GPT-6 分项使用同一个倍率。",
  cache_read: "只改变缓存读取成本，其他分项保持不变。", cache_creation: "只改变缓存创建成本。",
  output: "只改变输出成本。", input: "只改变普通输入成本。", mixed: "两项或更多分项采用不同倍率。",
};
const ordered = computed(() => [...(study.value?.causes ?? [])].sort((a, b) => (b.support ?? -1) - (a.support ?? -1)));
const leading = computed(() => ordered.value[0]);
const factorCause = ref("cache_read");
const selectedCause = computed(() => study.value?.causes.find(c => c.id === factorCause.value));
const qualityEntries = computed(() => Object.entries(study.value?.quality ?? {}).filter(([, v]) => v > 0));
const n = (value: number | undefined) => new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 }).format(value ?? 0);
const money = (value: number | undefined) => "$" + n(value);
const pct = (cause?: Cause) => cause?.support == null ? "尚无区分信息" : (cause.support * 100).toFixed(1) + "%";
async function load() {
  if (refreshing.value) return;
  refreshing.value = true;
  const request = new AbortController();
  const abort = () => request.abort();
  controller.signal.addEventListener("abort", abort, { once: true });
  const timeout = setTimeout(abort, 15000);
  try {
    const response = await fetch("/api/v2/studies/gpt6-components", { credentials: "omit", cache: "no-store", signal: request.signal });
    if (!response.ok) throw new Error("service");
    const data: Study = await response.json();
    if (controller.signal.aborted) return;
    study.value = data;
    error.value = "";
    lastRefresh.value = new Date().toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" });
  } catch {
    if (!controller.signal.aborted) error.value = "暂时无法读取科研统计。已有内容可能不是最新，请稍后重试。";
  } finally {
    clearTimeout(timeout);
    controller.signal.removeEventListener("abort", abort);
    loading.value = false;
    refreshing.value = false;
  }
}
function navigate() { route.value = location.hash.slice(1) || "/"; window.scrollTo({ top: 0, behavior: "instant" }); }
onMounted(() => {
  window.addEventListener("hashchange", navigate);
  void load();
  timer = setInterval(() => { if (document.visibilityState === "visible") void load(); }, 60000);
});
onBeforeUnmount(() => { controller.abort(); if (timer) clearInterval(timer); window.removeEventListener("hashchange", navigate); });
</script>

<template>
  <a class="skip" href="#main">跳到正文</a>
  <div class="site-shell">
    <header class="topbar">
      <a class="brand" href="#/" aria-label="共研首页"><span class="brand-icon" aria-hidden="true"><i></i><i></i><i></i></span><span>共研<small>CODEX SUBSCRIBE STUDY</small></span></a>
      <nav aria-label="主导航"><a href="#/" :aria-current="route === '/' || detail ? 'page' : undefined">研究项目</a><a href="#/method" :aria-current="route === '/method' ? 'page' : undefined">研究方法</a><a href="#/privacy" :aria-current="route === '/privacy' ? 'page' : undefined">隐私与参与</a></nav>
      <a class="source-link" href="https://github.com/LingyeNBird/CodexSubscribeStudy" target="_blank" rel="noreferrer">开源代码 ↗</a>
    </header>
    <main id="main">
      <div v-if="error" class="banner error" role="alert"><span>{{ error }}</span><button @click="load" :disabled="refreshing">重新获取</button></div>
      <template v-if="route === '/'">
        <section class="hero">
          <div class="hero-copy"><span class="eyebrow"><span class="tiny-square"></span>开放方法 / 共同验证</span><h1>让每一份用量，<br />成为一份<span class="highlight">证据。</span></h1><p>一条请求也能贡献，不必先独立得到答案。<br />汇聚原始统计证据，共同研究 GPT-6 计费差异。</p><div class="hero-actions"><a class="button primary" href="#/studies/gpt6-components">进入 GPT-6 研究 ↗</a><a class="text-link" href="#/privacy">了解如何参与 →</a></div><span class="hero-foot">自愿加入 · 无样本量门槛 · 原始证据联合分析</span></div>
          <div class="hero-art" aria-hidden="true"><div class="art-note">先提出假设<br /><strong>再让数据说话</strong><span>OBSERVE → COMPARE</span></div><div class="art-stack mint"><span>01</span><strong>输入 / 缓存 / 输出</strong></div><div class="art-stack violet"><span>02</span><strong>小片段也能互相补充</strong></div><div class="art-stack peach"><span>03</span><strong>保留不确定性。</strong></div><div class="art-plus">＋</div></div>
        </section>
        <section class="section-head"><div><span class="eyebrow">LIVE RESEARCH</span><h2>正在进行的研究</h2></div><span class="pill plain">01 个项目</span></section>
        <a class="study-card" href="#/studies/gpt6-components"><div class="project-mark" aria-hidden="true">6<span>↗</span></div><div class="project-copy"><div class="tag-row"><span class="pill">{{ labels[study?.state ?? 'no_data'] }}</span><span class="subtle">协议 v2 · 原始事实</span></div><h3>GPT-6 额度异常归因</h3><p>整体倍率？缓存读取？还是多项混合？比较七类解释。</p><div class="project-numbers"><span><b>{{ loading ? '—' : n(study?.totals.requests) }}</b>贡献请求</span><span><b>{{ loading ? '—' : n(study?.totals.contributors) }}</b>贡献安装</span><span><b>7</b>候选解释</span></div></div><span class="project-arrow" aria-hidden="true">↗</span></a>
        <section class="principles"><article><span class="mini-icon mint">↳</span><h3>明细留在本地</h3><p>不接收请求文本、Token 明细、账号身份或容量估值。</p></article><article><span class="mini-icon violet">≈</span><h3>小证据，共同研究</h3><p>不要求每个人先得到结论；不同的少量信息也可以互相补充。</p></article><article><span class="mini-icon peach">↻</span><h3>更新去重，历史保留</h3><p>同一批次更新替换，不同历史批次保留，不因停止更新而自动删除。</p></article></section>
      </template>
      <template v-else-if="detail">
        <a class="back" href="#/">← 全部研究</a>
        <section class="detail-heading"><div><span class="eyebrow">STUDY 01 / OPEN INVESTIGATION</span><h1>GPT-6 额度异常归因</h1><p>固定其他计费因素，中心联合比较四个成本分项。</p></div><div class="refresh-block"><span class="pill">{{ labels[study?.state ?? 'no_data'] }}</span><button class="button small" :disabled="refreshing" @click="load">{{ refreshing ? '读取中…' : '刷新统计 ↻' }}</button><small v-if="lastRefresh">页面更新 {{ lastRefresh }}</small></div></section>
        <div v-if="loading" class="loading-panel" role="status">正在读取公共统计…</div>
        <template v-else-if="study">
          <section class="metrics" aria-label="科研数据总量">
            <article><span>去标识化贡献安装</span><strong>{{ n(study.totals.contributors) }}<small>个</small></strong><p>{{ n(study.totals.batches) }} 个持久批次，无人数准入门槛</p></article>
            <article class="mint"><span>贡献请求总量</span><strong>{{ n(study.totals.requests) }}<small>次</small></strong><p>GPT-6 {{ n(study.totals.gpt6_requests) }} / 其他 {{ n(study.totals.other_requests) }}</p></article>
            <article class="violet"><span>原始标准成本</span><strong>{{ money(study.totals.raw_usd) }}</strong><p>其中 GPT-6 {{ money(study.totals.gpt6_raw_usd) }}</p></article>
            <article class="peach"><span>观测额度消耗</span><strong>{{ n(study.totals.quota_points) }}<small>百分点</small></strong><p>{{ n(study.totals.intervals) }} 个原始区间 / {{ n(study.totals.contrasts) }} 个容量内对比</p></article>
          </section>
          <p class="metric-caption">不同批次保留，同批次以最新版本计数。安装不是已验证的自然人；跨账号百分点不代表同一个额度池。</p>
          <section v-if="study.state === 'no_data' || study.state === 'uninformative'" class="empty-evidence"><span class="empty-symbol" aria-hidden="true">…</span><div><h2>{{ study.state === 'no_data' ? '第一份证据，还在路上。' : '贡献已收到，等待互补信息。' }}</h2><p>一条请求也能贡献，无需满两个周期或 200 次请求。没有区分信息时不把先验偏好冒充发现。</p><a class="text-link" href="#/privacy">从 Sub2Pool 自愿参与 →</a></div></section>
          <div v-else class="banner" :class="study.state === 'sensitive' ? 'warning' : 'mint'"><strong>当前支持较多：{{ leading?.label }}（{{ pct(leading) }}）</strong><span>{{ study.confidence_meaning }}</span></div>
          <section class="analysis-grid">
            <article class="panel ranking"><div class="panel-head"><div><span class="eyebrow">COMPARE, NOT CONFIRM</span><h2>各个原因的支持度</h2></div><span class="pill plain">7 类假设</span></div><p class="panel-intro">共同倍率参数的证据先汇合，再加入一次先验；不是平均各客户端的猜测百分比。</p><div class="causes"><div v-for="(cause, index) in ordered" :key="cause.id" class="cause"><div class="cause-heading"><span class="rank">{{ String(index + 1).padStart(2, '0') }}</span><strong>{{ cause.label }}</strong><b>{{ pct(cause) }}</b></div><div class="bar-track" :class="{ 'bar-empty': cause.support === null }"><div :style="{ width: `${(cause.support ?? 0) * 100}%` }" :class="'bar-' + (index % 4)"></div></div><p>{{ descriptions[cause.id] }}</p></div></div><p class="fineprint">这是固定假设下的条件证据支持度，不是已经证明官方计费机制的概率，也不会自动修改计价。</p></article>
            <aside class="side-panels">
              <article class="panel lavender"><span class="eyebrow">EVIDENCE QUALITY</span><h2>证据能回答多少？</h2><p class="panel-intro">局部信息不足不妨碍贡献；中心检查信息是否互补。</p><div class="quality-row"><span>全局可区分方向</span><strong>{{ study.information_rank }} / 4</strong></div><div class="quality-row"><span>最大来源信息占比</span><strong>{{ (study.maximum_source_information_share * 100).toFixed(1) }}%</strong></div><div class="divider"></div><div v-for="[status, count] in qualityEntries" :key="status" class="quality-row"><span>{{ labels[status] ?? status }}</span><b>{{ count }}</b></div><p v-for="warning in study.warnings" :key="warning" class="fineprint">{{ warning }}</p></article>
              <article class="panel factor-panel"><span class="eyebrow">CONDITIONAL PARAMETERS</span><h2>候选倍率速览</h2><label for="cause-select" class="subtle">选择一个解释族</label><select id="cause-select" v-model="factorCause"><option v-for="cause in study.causes" :key="cause.id" :value="cause.id">{{ cause.label }}</option></select><div class="factor-grid"><div v-for="(value, index) in selectedCause?.factor_estimates" :key="index"><span>{{ componentNames[index] }}</span><strong>{{ selectedCause?.support == null ? '—' : `× ${value.toFixed(2)}` }}</strong></div></div><p class="fineprint">联合证据在该解释族内的倍率均值，不是当前网关价格。没有读取或使用 PF／平均恒定容量估值。</p></article>
            </aside>
          </section>
          <details class="panel score-details"><summary>查看倍率范围与原始事实敏感性</summary><p>90% 范围来自离散网格的条件证据分布，不保证真实机制的覆盖率。三种容量波动假设仅使用原始事实，不是引入容量估值。</p><div class="table-scroll"><table><thead><tr><th>分项</th><th>均值</th><th>90% 条件范围</th></tr></thead><tbody><tr v-for="p in study.parameters" :key="p.name"><td>{{ componentLabel[p.name] }}</td><td>{{ p.mean.toFixed(3) }}</td><td>{{ p.low }} ～ {{ p.high }}</td></tr></tbody></table><table v-if="study.drift_support.length"><thead><tr><th>解释</th><th>无漂移</th><th>常规漂移</th><th>较强漂移</th></tr></thead><tbody><tr v-for="(cause, i) in study.causes" :key="cause.id"><td>{{ cause.label }}</td><td v-for="(curve, j) in study.drift_support" :key="j">{{ (curve[i] * 100).toFixed(1) }}%</td></tr></tbody></table></div><p v-if="study.gpt6_quota">候选下 GPT-6 额度归属：{{ n(study.gpt6_quota.mean) }} 百分点（{{ n(study.gpt6_quota.low) }} ～ {{ n(study.gpt6_quota.high) }}）。这是归因结果，不能反过来当作真实扣减标签。</p></details>
          <div v-if="study.legacy_archive.contributors" class="banner"><strong>旧协议贡献已保留</strong><span>{{ n(study.legacy_archive.contributors) }} 个安装，{{ n(study.legacy_archive.requests) }} 次请求，{{ money(study.legacy_archive.raw_usd) }}。单独归档，不与 v2 重复累计。</span></div>
          <div class="method-strip"><div><strong>让结果可以被质疑，也可以被复现。</strong><p>方法 {{ study.method }} · 数据更新 {{ study.updated_at || '暂无' }}</p></div><a class="button" href="#/method">阅读方法说明 ↗</a></div>
        </template>
      </template>
      <section v-else-if="route === '/method'" class="prose-page"><span class="eyebrow">METHOD / RAW-ONLY VERSION 2</span><h1>小证据，也能<br />共同回答大问题。</h1><p class="lead">每个人不必先独立识别原因；客户端提交同一组候选倍率的统计证据，中心联合分析。</p><div class="method-steps"><article class="panel mint"><span class="step">01</span><h2>固定其他计费因素</h2><p>FAST 正确目标为 2，GPT-5.6／GPT-6 长上下文不额外翻倍，其他模型价格假定正确。本项目只估计 GPT-6 四分项倍率。</p></article><article class="panel violet"><span class="step">02</span><h2>只用原始事实</h2><p>按账号短片段处理未知容量尺度，考虑额度整数显示和波动。不发送、不存储、不借用粒子滤波或平均恒定容量估值。</p></article><article class="panel peach"><span class="step">03</span><h2>先汇聚，再比较</h2><p>1,311 组共同倍率参数，先累加证据，再按七类解释归一化。局部缺少某个方向的信息，可以由其他来源补充。</p></article></div><div class="panel"><h2>哪些数据进入研究？</h2><p>无本地请求数、区间数、周期数或置信度门槛。FAST、长上下文、混用其他模型不再触发整段拒收。缺少对应额度或成本分项时仍接收规模统计，但不会制造不存在的证据。</p><h2>支持度有什么含义？</h2><p>支持度是预先固定方法、价格前提与容量工作模型下的条件证据分布，不是官方真实机制的已校准概率。增加独立、互补的小样本能改善信息；所有来源共有的系统性偏差不会自动消失。</p><h2>如实保留失败情景</h2><p>历史模拟比较了容量估值代入等方案，最终工程已完全移除这些估值。容量突变系统性追随模型选择时，仍可能产生错误的高支持度；方法文档保留反例，不把软件测试通过说成真实计费研究成功。</p><a class="button" href="https://github.com/LingyeNBird/CodexSubscribeStudy/blob/main/docs/method-v2.md" target="_blank" rel="noreferrer">公式、实验与限制 ↗</a></div></section>
      <section v-else-if="route === '/privacy'" class="prose-page"><span class="eyebrow">PRIVACY / VOLUNTARY CONTRIBUTIONS</span><h1>参与是选择。<br />知情是前提。</h1><p class="lead">Sub2Pool 默认关闭科研，保存并明确授权后才分享。接收网站默认为 https://codex.nightunderfly.online/。</p><div class="privacy-grid"><article class="panel mint"><span class="pill plain">会发送</span><h2>原始事实的统计证据</h2><p>批次请求数、原始金额和额度统计、质量计数、共同参数证据曲线、信息矩阵及候选额度归属函数。随机安装公钥和批次编号用于去重、更新与撤回。</p></article><article class="panel violet"><span class="pill plain">不会发送</span><h2>请求与身份明细</h2><p>提示词、回答、Token 明细、调用时间线、账号／参与者名称、API Key、Sub2API 地址、IP 字段，以及粒子滤波／平均恒定容量估值或其辅助分析。</p></article></div><div class="panel"><h2>“匿名”有明确边界</h2><p>随机公钥具有同站点链接性，因此是去标识化，不是绝对匿名。无门槛意味着小贡献也可能有较少聚合保护。网络层和反代仍能看到出口 IP，应用不记录 IP 不代表网络设施不可见。</p><h2>保留历史，不自动撤回</h2><p>同批次更新只替换旧版本，不同历史批次长期保留，不因 120 天未更新自动删除。旧协议数据单独归档。关闭科研、导入数据库或身份失效不会自动向本站发起撤回。</p><p>只有明确确认的签名撤回才删除相应安装的远端贡献；本地原始事实不受影响。旧数据库页、备份、他人已下载汇总无法保证物理擦除。不同随机身份之间无法自动辨认重复的同一额度池。</p><h2>面向开发者的公共接口</h2><pre>GET  /api/v2/studies
GET  /api/v2/studies/gpt6-components
GET  /api/v2/protocol
POST /api/v2/reports
POST /api/v2/withdraw</pre><a class="button" href="https://github.com/LingyeNBird/CodexSubscribeStudy/blob/main/docs/api-v2.md" target="_blank" rel="noreferrer">提交协议与隐私清单 ↗</a></div></section>
      <section v-else class="empty-evidence"><h1>这个页面还不存在。</h1><a class="button" href="#/">回到研究首页</a></section>
    </main>
    <footer><a class="footer-brand" href="#/">共研 <span>Codex Subscribe Study</span></a><p>证据可以汇聚，不确定性不该被隐藏。</p><a href="#/privacy">去标识化 · 可撤回</a></footer>
  </div>
</template>
