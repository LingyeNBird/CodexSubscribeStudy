<template>
  <section class="prose-page">
    <span class="eyebrow">METHOD / RAW FACTS</span>
    <h1>小证据，也能<br />共同回答大问题。</h1>
    <p class="lead">每个人不必先独立识别原因；客户端提交同一组候选倍率的统计证据，中心联合分析。</p>
    <div class="method-steps">
      <article class="panel mint">
        <span class="step">01</span>
        <h2>固定其他计费因素</h2>
        <p>
          FAST 正确目标为 2，GPT-5.6／GPT-6 长上下文不额外翻倍，其他模型价格假定正确。本项目只估计
          GPT-6 四分项倍率。
        </p>
      </article>
      <article class="panel violet">
        <span class="step">02</span>
        <h2>只用原始事实</h2>
        <p>
          按账号短片段处理未知容量尺度，考虑额度整数显示和波动。不发送、不存储、不借用粒子滤波或平均恒定容量估值。
        </p>
      </article>
      <article class="panel peach">
        <span class="step">03</span>
        <h2>先汇聚，再比较</h2>
        <p>
          1,311
          组共同倍率参数，先累加证据，再按七类解释归一化。局部缺少某个方向的信息，可以由其他来源补充。
        </p>
      </article>
    </div>
    <div class="panel">
      <h2>哪些数据进入研究？</h2>
      <p>
        无本地请求数、区间数、周期数或置信度门槛。FAST、长上下文、混用其他模型不会触发整段拒收。缺少对应额度或成本分项时仍接收规模统计，但不会制造不存在的证据。
      </p>
      <h2>支持度有什么含义？</h2>
      <p>
        支持度是预先固定方法、价格前提与容量工作模型下的条件证据分布，不是官方真实机制的已校准概率。独立、互补的小样本能补充信息；所有来源共有的系统性偏差不会自动消失。
      </p>
      <h2>如实保留失败情景</h2>
      <p>
        容量突变系统性追随模型选择时，仍可能产生错误的高支持度；方法文档保留反例，不把软件测试通过说成真实计费研究成功。
      </p>
      <a
        class="button"
        href="https://github.com/LingyeNBird/CodexSubscribeStudy/blob/main/docs/method.md"
        target="_blank"
        rel="noreferrer"
        >公式、实验与限制 ↗</a
      >
    </div>
  </section>
</template>

<style scoped src="./MethodPage.css"></style>
