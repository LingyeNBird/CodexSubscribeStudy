<script setup lang="ts">
import { computed } from "vue";
const model = defineModel<number | undefined>();
const riskLabel = computed(() => {
  const value = model.value;
  if (value === undefined) return "未填写";
  return value < 15
    ? "低风险"
    : value < 25
      ? "较低风险"
      : value < 40
        ? "中等风险"
        : value < 50
          ? "偏高风险"
          : value < 70
            ? "高风险"
            : "极高风险";
});
const update = (event: Event) => {
  model.value = Number((event.target as HTMLInputElement).value);
};
function select(event: PointerEvent) {
  if (event.button === 0) update(event);
}
</script>

<template>
  <div class="ip-risk">
    <div class="risk-heading">
      <output for="ip-risk-range" :class="{ 'risk-unset': model === undefined }">{{
        model === undefined ? "未填写" : `${model}% · ${riskLabel}`
      }}</output>
      <button
        v-if="model !== undefined"
        type="button"
        class="risk-reset"
        @click="model = undefined"
      >
        清空
      </button>
    </div>
    <input
      id="ip-risk-range"
      name="ip-risk"
      type="range"
      min="0"
      max="100"
      step="1"
      :value="model ?? 50"
      :aria-valuetext="model === undefined ? '未填写' : `${model}% ${riskLabel}`"
      aria-label="IP 风险程度"
      @input="update"
      @change="update"
      @pointerdown="select"
    />
    <div class="risk-ticks" aria-hidden="true">
      <span>0%</span><span>50%</span><span>100%</span>
    </div>
    <div class="risk-ends"><span>低风险</span><span>高风险</span></div>
    <p class="risk-sources">
      可以从以下网站查询：
      <a href="https://ippure.com/" target="_blank" rel="noopener noreferrer">IPPure</a>、
      <a href="https://ip.net.coffee/ip/" target="_blank" rel="noopener noreferrer">IP Net Coffee</a>、
      <a href="https://ipsuper.com/" target="_blank" rel="noopener noreferrer">IPSuper</a>、
      <a href="https://cleanip.io/" target="_blank" rel="noopener noreferrer">CleanIP</a>、
      <a href="https://www.iptrait.com/" target="_blank" rel="noopener noreferrer">IPTrait</a>。
    </p>
  </div>
</template>

<style scoped src="./IPRiskSlider.css"></style>
