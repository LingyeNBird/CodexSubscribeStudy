<template>
  <section class="prose-page">
    <span class="eyebrow">PRIVACY / VOLUNTARY CONTRIBUTIONS</span>
    <h1>参与是选择。<br />知情是前提。</h1>
    <p class="lead">
      Sub2Pool 默认关闭科研，保存并明确授权后才分享。接收网站默认为
      https://codex.nightunderfly.online/。
    </p>
    <div class="privacy-grid">
      <article class="panel mint">
        <span class="pill plain">会发送</span>
        <h2>原始事实的统计证据</h2>
        <p>
          批次请求数、原始金额和额度统计、质量计数、共同参数证据曲线、信息矩阵及候选额度归属函数。随机安装公钥和批次编号用于去重和更新。
        </p>
      </article>
      <article class="panel violet">
        <span class="pill plain">不会发送</span>
        <h2>请求与身份明细</h2>
        <p>
          提示词、回答、Token 明细、调用时间线、账号／参与者名称、API Key、Sub2API 地址、IP
          字段，以及粒子滤波／平均恒定容量估值或其辅助分析。
        </p>
      </article>
    </div>
    <div class="panel">
      <h2>“匿名”有明确边界</h2>
      <p>
        随机公钥具有同站点链接性，因此是去标识化，不是绝对匿名。无门槛意味着小贡献也可能有较少聚合保护。网络层和反代仍能看到出口
        IP，应用不记录 IP 不代表网络设施不可见。
      </p>
      <h2>贡献会长期保留</h2>
      <p>
        同批次更新替换，不同批次长期保留，不因 120
        天未更新、停止参与、关闭科研、导入数据库或身份失效而删除。
      </p>
      <p>
        本站不提供删除已提交贡献的接口。参与前请确认接受长期保留规则。不同随机身份之间无法自动辨认重复的同一额度池。
      </p>
      <h2>面向开发者的公共接口</h2>
      <pre>
GET  /api/studies
GET  /api/studies/gpt6-components
GET  /api/protocol
POST /api/reports</pre>
      <a
        class="button"
        href="https://github.com/LingyeNBird/CodexSubscribeStudy/blob/main/docs/api.md"
        target="_blank"
        rel="noreferrer"
        >提交协议与隐私清单 ↗</a
      >
    </div>
  </section>
</template>

<style scoped src="./PrivacyPage.css"></style>
