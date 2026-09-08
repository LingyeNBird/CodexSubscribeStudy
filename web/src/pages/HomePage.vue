<script setup lang="ts">
import type { Study } from "../types";
import { labels, n } from "../studyPresentation";
defineProps<{ study: Study | null; loading: boolean }>();
</script>

<template>
  <section class="hero">
    <div class="hero-copy">
      <h1>让每一份用量，<br />成为一份<span class="highlight">证据。</span></h1>
      <div class="hero-actions">
        <a class="button primary" href="#/studies/gpt6-components">进入 GPT-6 研究 ↗</a
        ><a class="text-link" href="#/privacy">了解如何参与 →</a>
      </div>
    </div>
    <div class="hero-art" aria-hidden="true">
      <div class="art-note">先提出假设<br /><strong>再让数据说话</strong></div>
      <div class="art-stack mint"><span>01</span><strong>输入 / 缓存 / 输出</strong></div>
      <div class="art-stack violet"><span>02</span><strong>小片段也能互相补充</strong></div>
      <div class="art-stack peach"><span>03</span><strong>保留不确定性。</strong></div>
      <div class="art-plus">＋</div>
    </div>
  </section>
  <section class="section-head">
    <h2>正在进行的研究</h2>
    <span class="pill plain">02 个项目</span>
  </section>
  <a class="study-card" href="#/studies/gpt6-components"
    ><div class="project-mark" aria-hidden="true">6<span>↗</span></div>
    <div class="project-copy">
      <div class="tag-row">
        <span class="pill">{{ labels[study?.state ?? "no_data"] }}</span>
      </div>
      <h3>GPT-6 额度异常归因</h3>
      <p>整体倍率？缓存读取？还是多项混合？比较七类解释。</p>
      <div class="project-numbers">
        <span
          ><b>{{ loading ? "—" : n(study?.totals.requests) }}</b
          >贡献请求</span
        ><span
          ><b>{{ loading ? "—" : n(study?.totals.contributors) }}</b
          >贡献安装</span
        ><span><b>7</b>候选解释</span>
      </div>
    </div>
    <span class="project-arrow" aria-hidden="true">↗</span></a
  >
  <a class="study-card survey-study-card" href="#/studies/chatgpt-account-survey"
    ><div class="project-mark peach" aria-hidden="true">02<span>↗</span></div>
    <div class="project-copy">
      <div class="tag-row"><span class="pill plain">调查问卷</span></div>
      <h3>ChatGPT 套餐降智封号统计</h3>
      <p>从账号状态、套餐与连接方式，记录你的使用经历。</p>
      <div class="project-numbers"><span>自愿填写</span><span>查看统计结果</span></div>
    </div>
    <span class="project-arrow" aria-hidden="true">↗</span></a
  >
  <section class="principles">
    <article>
      <span class="mini-icon mint">↳</span>
      <h3>明细留在本地</h3>
      <p>不接收请求文本、Token 明细、账号身份或容量估值。</p>
    </article>
    <article>
      <span class="mini-icon violet">≈</span>
      <h3>小证据，共同研究</h3>
      <p>不要求每个人先得到结论；不同的少量信息也可以互相补充。</p>
    </article>
    <article>
      <span class="mini-icon peach">↻</span>
      <h3>更新去重，批次保留</h3>
      <p>同一批次更新替换，不同批次保留，不因停止更新而自动删除。</p>
    </article>
  </section>
</template>

<style scoped src="./HomePage.css"></style>
