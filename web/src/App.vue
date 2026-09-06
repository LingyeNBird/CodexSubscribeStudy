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
const labels: Record<string, string> = { no_data: "等待第一份证据", collecting: "持续收集样本", exploratory: "探索性结果", heterogeneous: "不同来源存在分歧", insufficient_data: "样本仍不足", unidentifiable: "组成相关 / 难以识别", model_mismatch: "候选解释均不适配", drift_sensitive: "额度波动假设敏感", external_usage_uncontrolled: "用量覆盖未确认" };
const componentNames = ["普通输入", "缓存创建", "缓存读取", "输出"];
const descriptions: Record<string, string> = { unchanged: "保留原始分项价格，不增加额外倍率。", global: "所有 GPT-6 分项使用同一个倍率。", cache_read: "只改变缓存读取成本，其他分项保持不变。", cache_creation: "只改变缓存创建成本，其他分项保持不变。", output: "只改变输出成本，其他分项保持不变。", input: "只改变普通输入成本，其他分项保持不变。", mixed: "两项或更多分项采用不同倍率的组合。" };
const ordered = computed(() => [...(study.value?.causes ?? [])].sort((a, b) => (b.support ?? -1) - (a.support ?? -1)));
const leading = computed(() => ordered.value[0]);
const factorCause = ref("cache_read");
const selectedCause = computed(() => study.value?.causes.find(c => c.id === factorCause.value));
const qualityEntries = computed(() => Object.entries(study.value?.quality ?? {}).sort((a, b) => b[1] - a[1]));
const n = (value: number | undefined) => new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 1 }).format(value ?? 0);
const money = (value: number | undefined) => "$" + new Intl.NumberFormat("en-US", { maximumFractionDigits: 0 }).format(value ?? 0);
const pct = (cause?: Cause) => cause?.support == null ? "尚无结论" : (cause.support * 100).toFixed(1) + "%";
async function load() {
  if (refreshing.value) return;
  refreshing.value = true;
  const request = new AbortController();
  const abort = () => request.abort();
  controller.signal.addEventListener("abort", abort, { once: true });
  const timeout = setTimeout(abort, 15000);
  try {
    const response = await fetch("/api/v1/studies/gpt6-components", { credentials: "omit", cache: "no-store", signal: request.signal });
    if (!response.ok) throw new Error("service");
    study.value = await response.json();
    error.value = "";
    lastRefresh.value = new Date().toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" });
  } catch (cause) {
    if (!controller.signal.aborted) error.value = "暂时无法读取科研统计。已有内容可能不是最新，请稍后重试。";
  } finally { clearTimeout(timeout); controller.signal.removeEventListener("abort", abort); loading.value = false; refreshing.value = false; }
}
function navigate() { route.value = location.hash.slice(1) || "/"; window.scrollTo({ top: 0, behavior: "instant" }); }
onMounted(() => { window.addEventListener("hashchange", navigate); void load(); timer = setInterval(() => { if (document.visibilityState === "visible") void load(); }, 60000); });
onBeforeUnmount(() => { controller.abort(); if (timer) clearInterval(timer); window.removeEventListener("hashchange", navigate); });
</script>

<template>
  <a class="skip" href="#main">跳到正文</a>
  <div class="site-shell">
    <header class="topbar">
      <a class="brand" href="#/" aria-label="共研首页"><span class="brand-icon" aria-hidden="true"><i></i><i></i><i></i></span><span>共研<small>CODEX SUBSCRIBE STUDY</small></span></a>
      <nav aria-label="主导航"><a href="#/" :aria-current="route === '/' || detail ? 'page' : undefined">研究项目</a><a href="#/method" :aria-current="route === '/method' ? 'page' : undefined">研究方法</a><a href="#/privacy" :aria-current="route === '/privacy' ? 'page' : undefined">隐私与参与</a></nav>
      <a class="source-link" href="https://github.com/LingyeNBird/CodexSubscribeStudy" target="_blank" rel="noreferrer">开源代码 <span aria-hidden="true">↗</span></a>
    </header>
    <main id="main">
      <div v-if="error" class="banner error" role="alert"><span>{{ error }}</span><button @click="load" :disabled="refreshing">重新获取</button></div>
      <template v-if="route === '/'">
        <section class="hero">
          <div class="hero-copy"><span class="eyebrow"><span class="tiny-square"></span>开放方法 / 共同验证</span><h1>让每一份用量，<br />成为一份<span class="highlight">证据。</span></h1><p>不是先选一个倍率，再寻找答案。<br />用社区的去标识化统计，一起理解订阅额度的真实差异。</p><div class="hero-actions"><a class="button primary" href="#/studies/gpt6-components">进入 GPT-6 研究 <span aria-hidden="true">↗</span></a><a class="text-link" href="#/privacy">了解如何参与 →</a></div><span class="hero-foot">自愿加入 · 本地分析 · 只分享汇总</span></div>
          <div class="hero-art" aria-hidden="true"><div class="art-note">先提出假设<br /><strong>再让数据说话</strong><span>OBSERVE → COMPARE</span></div><div class="art-stack mint"><span>01</span><strong>输入 / 缓存 / 输出</strong></div><div class="art-stack violet"><span>02</span><strong>额度波动 ≠ 计费异常</strong></div><div class="art-stack peach"><span>03</span><strong>保留不确定性。</strong></div><div class="art-plus">＋</div></div>
        </section>
        <section class="section-head"><div><span class="eyebrow">LIVE RESEARCH</span><h2>正在进行的研究</h2></div><span class="pill plain">01 个项目</span></section>
        <a class="study-card" href="#/studies/gpt6-components">
          <div class="project-mark" aria-hidden="true">6<span>↗</span></div><div class="project-copy"><div class="tag-row"><span class="pill">{{ labels[study?.state ?? 'no_data'] }}</span><span class="subtle">协议 v1 · 观察性研究</span></div><h3>GPT-6 额度异常归因</h3><p>整体倍率？缓存读取？还是多项混合？<br class="desktop-break" />比较七类解释，隔离订阅额度随时间变化的干扰。</p><div class="project-numbers"><span><b>{{ loading ? '—' : n(study?.totals.requests) }}</b>合格请求</span><span><b>{{ loading ? '—' : n(study?.totals.contributors) }}</b>贡献安装</span><span><b>7</b>候选解释</span></div></div><span class="project-arrow" aria-hidden="true">↗</span>
        </a>
        <section class="principles"><article><span class="mini-icon mint">↳</span><h3>明细留在本地</h3><p>不接收提示词、回答、Token 明细、账号名称或请求时间序列。</p></article><article><span class="mini-icon violet">≈</span><h3>不把相关当作因果</h3><p>公开候选、假设和局限。数据不够，结果就保持“尚无结论”。</p></article><article><span class="mini-icon peach">↻</span><h3>持续更新，不重复计票</h3><p>每个安装只保留最新滚动统计，重复发送不会累计成新贡献。</p></article></section>
      </template>
      <template v-else-if="detail">
        <a class="back" href="#/">← 全部研究</a>
        <section class="detail-heading"><div><span class="eyebrow">STUDY 01 / OPEN INVESTIGATION</span><h1>GPT-6 额度异常归因</h1><p>以 GPT-5.6 为对照，比较分项计费与额度变化的关系。</p></div><div class="refresh-block"><span class="pill">{{ labels[study?.state ?? 'no_data'] }}</span><button class="button small" :disabled="refreshing" @click="load">{{ refreshing ? '读取中…' : '刷新统计 ↻' }}</button><small v-if="lastRefresh">页面更新 {{ lastRefresh }}</small></div></section>
        <div v-if="loading" class="loading-panel" role="status">正在读取公共统计…</div>
        <template v-else-if="study">
          <section class="metrics" aria-label="科研数据总量"><article><span>匿名贡献安装</span><strong>{{ n(study.totals.contributors) }}<small>个</small></strong><p>{{ n(study.totals.eligible_contributors) }} 个通过归因门槛</p></article><article class="mint"><span>研究请求总量</span><strong>{{ n(study.totals.requests) }}<small>次</small></strong><p>GPT-6 {{ n(study.totals.gpt6_requests) }} / GPT-5.6 {{ n(study.totals.baseline_requests) }}</p></article><article class="violet"><span>原始标准成本</span><strong>{{ money(study.totals.raw_usd) }}</strong><p>其中 GPT-6 {{ money(study.totals.gpt6_raw_usd) }}</p></article><article class="peach"><span>观测额度消耗</span><strong>{{ n(study.totals.quota_points) }}<small>百分点</small></strong><p>{{ n(study.totals.cycles) }} 个账号周期 / {{ n(study.totals.blocks) }} 个区间</p></article></section>
          <p class="metric-caption">各安装最新 {{ study.window_days }} 天窗口的合计，不是永久累计或真人数量；跨账号百分点相加不代表同一个额度池。超过 120 天未更新的安装不纳入统计。</p>
          <section v-if="study.state === 'no_data' || study.state === 'collecting'" class="empty-evidence"><span class="empty-symbol" aria-hidden="true">…</span><div><h2>{{ study.state === 'no_data' ? '第一份证据，还在路上。' : '样本在积累，结论不抢跑。' }}</h2><p>至少 {{ study.minimum_contributors }} 个安装通过数据完整性与可识别性门槛后，才展示公共支持度。当前 {{ study.totals.eligible_contributors }} / {{ study.minimum_contributors }}。这里的空白不是 0% 支持，也不是证明没有异常。</p><a class="text-link" href="#/privacy">从 Sub2Pool 自愿参与 →</a></div></section>
          <div v-else class="banner" :class="study.state === 'heterogeneous' ? 'warning' : 'mint'"><strong>{{ study.state === 'heterogeneous' ? '来源之间有分歧，暂不建议使用单一解释。' : `当前支持较多：${leading?.label}（${pct(leading)}）` }}</strong><span>{{ study.confidence_meaning }} 不会自动调整 Sub2Pool 的计费配置。</span></div>
          <section class="analysis-grid"><article class="panel ranking"><div class="panel-head"><div><span class="eyebrow">COMPARE, NOT CONFIRM</span><h2>各个原因的支持度</h2></div><span class="pill plain">7 类假设</span></div><p class="panel-intro">按整个账号周期留出验证，再在安装与周期两个层面重抽样；不是把用户上传的百分比简单平均。</p><div class="causes"><div v-for="(cause, index) in ordered" :key="cause.id" class="cause"><div class="cause-heading"><span class="rank">{{ String(index + 1).padStart(2, '0') }}</span><strong>{{ cause.label }}</strong><b>{{ pct(cause) }}</b></div><div class="bar-track" :class="{ 'bar-empty': cause.support === null }"><div :style="{ width: `${(cause.support ?? 0) * 100}%` }" :class="'bar-' + (index % 4)"></div></div><p>{{ descriptions[cause.id] }}</p></div></div><p class="fineprint">支持度表示重抽样中该解释族的预测表现排第一的比例；候选都可能不是真相，数值也不是可直接采用的定价。</p></article>
            <aside class="side-panels"><article class="panel lavender"><span class="eyebrow">EVIDENCE QUALITY</span><h2>证据能回答多少？</h2><p class="panel-intro">只有用量覆盖已确认，且分项构成充分变化的样本才能参与原因排名。</p><div class="quality-row" v-for="(count, index) in study.identifiable_sites" :key="index"><span>{{ componentNames[index] }}</span><strong>{{ count }} 个可区分安装</strong></div><div class="divider"></div><div v-for="[status, count] in qualityEntries" :key="status" class="quality-row"><span>{{ labels[status] ?? status }}</span><b>{{ count }}</b></div><p v-if="!qualityEntries.length" class="subtle">尚无有效提交</p></article>
              <article class="panel factor-panel"><span class="eyebrow">EXPLORATORY PARAMETERS</span><h2>候选倍率速览</h2><label for="cause-select" class="subtle">选择一个解释族</label><select id="cause-select" v-model="factorCause"><option v-for="cause in study.causes" :key="cause.id" :value="cause.id">{{ cause.label }}</option></select><div class="factor-grid"><div v-for="(value, index) in selectedCause?.factor_estimates" :key="index"><span>{{ componentNames[index] }}</span><strong>{{ selectedCause?.support == null ? '—' : `× ${value.toFixed(2)}` }}</strong></div></div><p class="fineprint">这是离散网格下各安装参数后验均值的汇总，不是公共联合后验、可信区间或计费建议。网格范围为 0.5–3 倍；未知分项不意味着倍率为 1。</p></article>
            </aside></section>
          <details class="panel score-details"><summary>查看预测评分与不确定性</summary><p>相对“无需额外倍率”的每个对比点预测增益，正数较好。90% 重抽样区间反映来源与周期变异，不是因果置信区间。</p><div class="table-scroll"><table><thead><tr><th>解释</th><th>平均增益</th><th>90% 区间</th></tr></thead><tbody><tr v-for="cause in study.causes" :key="cause.id"><td>{{ cause.label }}</td><td>{{ cause.score_mean?.toFixed(3) ?? '—' }}</td><td>{{ cause.score_low?.toFixed(3) ?? '—' }} ～ {{ cause.score_high?.toFixed(3) ?? '—' }}</td></tr></tbody></table></div></details>
          <div class="method-strip"><div><strong>让结果可以被质疑，也可以被复现。</strong><p>方法 {{ study.method }} · 数据更新 {{ study.updated_at || '暂无' }} · 普通档位 / 非长上下文</p></div><a class="button" href="#/method">阅读方法说明 ↗</a></div>
        </template>
      </template>
      <section v-else-if="route === '/method'" class="prose-page"><span class="eyebrow">METHOD / VERSION 1</span><h1>先定义“正常”，<br />再比较异常。</h1><p class="lead">我们比较的不是两种模型每次请求的价格是否相等，而是不同计费解释能否使同一账号的单位额度成本随时间更一致。</p><div class="method-steps"><article class="panel mint"><span class="step">01</span><h2>从原始事实出发</h2><p>对齐真实额度快照时间与原始成本分项。不使用已被 1.8 倍率修正的结果，也不把 Sub2Pool 的粒子滤波估计当作真值，避免循环论证。</p></article><article class="panel violet"><span class="step">02</span><h2>允许额度本身变化</h2><p>每个账号周期有自己的未知容量。相邻区间的对数单位额度成本差消除初始容量差异，随机游走描述期间波动，同时考虑整数百分比的共享端点误差。</p></article><article class="panel peach"><span class="step">03</span><h2>用没参与拟合的周期验证</h2><p>七类解释、1,311 个预先声明的参数组合，按整个账号周期留出预测。复杂的混合解释不能仅凭“试过更多参数”赢得样本内比赛。</p></article></div><div class="panel"><h2>哪些数据进入研究？</h2><p>只使用原始请求覆盖完整、普通档位、非长上下文、仅含 GPT-5.6 和 GPT-6 的区间；每段至少消耗 3 个百分点，不超过 6 小时。每周期至少 8 段，包含两种模型；本地至少 2 周期、24 段、200 次请求、各模型族至少 50 次才尝试比较。</p><p>参与者还需要确认同一额度池没有未采集的网页端或其他网关用量。否则只展示样本量，不参与原因排名。这个确认是自报信息，不是技术上已验证的事实。</p><h2>“置信度”具体是什么？</h2><p>本地按账号周期重抽样；公共平台按安装重抽样，并用周期评分协方差加入有限样本不确定性。页面百分比是预测排名的胜率，不是某个真实价格机制有该概率成立。贡献安装很少时区间仍可能不稳定，因此至少三个合格安装才展示排名。</p><p>分项构成高度相关、只有一次模型切换、所有候选都拟合很差、不同容量波动先验产生不同赢家时，会显示证据不足或敏感性警告，而不是硬凑出百分比。</p><h2>这些结论不能说明什么？</h2><p>观察性数据无法彻底分开恰好伴随模型切换的额度策略变化、账号等级差异、网关自定义价卡或漏记用量。容量随机游走是假设而非平台承诺；相似的混合因素可能仍不可识别。普通短上下文上的结果不能外推到 FAST 或长上下文请求。</p><p>匿名提交也不能保证贡献来自不同真人或没有伪造。平台限制单个安装权重并校验格式与签名，但无法杜绝多身份灌入或多安装使用相同账号。不据此自动改价格。</p><a class="button" href="https://github.com/LingyeNBird/CodexSubscribeStudy/blob/main/docs/method.md" target="_blank" rel="noreferrer">完整公式、模拟实验与限制 ↗</a></div></section>
      <section v-else-if="route === '/privacy'" class="prose-page"><span class="eyebrow">PRIVACY / VOLUNTARY CONTRIBUTIONS</span><h1>参与是选择。<br />知情是前提。</h1><p class="lead">在 Sub2Pool 的「系统设置 → 科研共创」中自愿开启。默认关闭；你能看到发送内容、网站地址和当前本地分析，随时停止或撤回。</p><div class="privacy-grid"><article class="panel mint"><span class="pill plain">会发送</span><h2>只有统计摘要</h2><p>滚动 90 天的总请求数、两种模型族次数、取整后的标准成本、额度百分点、有效周期与区间数、排除原因数量、可识别性标记。</p><p>七个解释族的预测评分均值、协方差、重抽样支持度、代表倍率及固定方法版本。另有随机生成的网站隔离公钥、版本号和签名，用于去重与撤回。</p></article><article class="panel violet"><span class="pill plain">不会发送</span><h2>你的请求与身份明细</h2><p>提示词、回答、每条 Token 用量、逐请求/逐区间记录、时间序列、账号和参与者名称、API Key、Sub2API 地址。</p><p>提交正文不含 IP 字段；应用不读取 IP 或转发地址，也不写访问日志。没有第三方统计脚本、远程字体或追踪 Cookie。</p></article></div><div class="panel"><h2>“匿名”有明确边界</h2><p>随机公钥会把同一个安装的更新关联起来，因此是去标识化，不是不可关联。直接 HTTP 连接中，接收网站及其代理/托管商仍能看到你的出口 IP。应用不记录 IP，不等于网络层不可见；运营者必须同步关闭或缩短代理/CDN 日志。</p><h2>不重复计票，也允许撤回</h2><p>每个安装只保留最新版本的滚动摘要，旧摘要被替换；120 天未更新则过期，不再展示或计入。公钥不是用户账号，也不是独立真人证明。</p><p>关闭科研会停止新的发送任务，但已发出的请求可能完成。点击“停止并撤回已提交统计”会发送签名删除请求，服务保留无统计内容的键哈希和版本墓碑防止旧请求复活。备份和第三方缓存可能延后删除，公开聚合截图无法追回。</p><h2>面向开发者的公共接口</h2><p>使用 Ed25519 对完整请求体、HTTP 方法和路径签名，接收端严格验证固定字段，不接收任意明细。无需注册，不提供单个贡献者的公开查询接口。</p><pre>GET  /api/v1/studies
GET  /api/v1/studies/gpt6-components
GET  /api/v1/protocol
POST /api/v1/reports
POST /api/v1/withdraw</pre><a class="button" href="https://github.com/LingyeNBird/CodexSubscribeStudy/blob/main/docs/api.md" target="_blank" rel="noreferrer">查看提交协议与部署隐私清单 ↗</a></div></section>
      <section v-else class="empty-evidence"><h1>这个页面还不存在。</h1><a class="button" href="#/">回到研究首页</a></section>
    </main>
    <footer><a class="footer-brand" href="#/">共研 <span>Codex Subscribe Study</span></a><p>证据可以汇聚，不确定性不该被隐藏。</p><a href="#/privacy">去标识化 · 可撤回</a></footer>
  </div>
</template>
