<script setup lang="ts">
defineProps<{ id: string; label: string }>();
const precision = defineModel<string>("precision", { required: true });
const date = defineModel<string>("date", { required: true });
const minute = defineModel<string>("minute", { required: true });
const unknown = defineModel<boolean>("unknown", { required: true });
const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
</script>

<template>
  <fieldset class="follow-up">
    <legend>{{ label }}</legend>
    <div class="event-controls">
      <div class="choices">
        <label
          v-for="option in [
            { value: 'day', label: '只知道日期' },
            { value: 'minute', label: '精确到分钟' },
          ]"
          :key="option.value"
          class="choice"
          ><input
            v-model="precision"
            type="radio"
            :name="`${id}-precision`"
            :value="option.value"
            :disabled="unknown"
          />{{ option.label }}</label
        >
      </div>
      <button
        type="button"
        class="unknown-button"
        :aria-pressed="unknown"
        @click="unknown = !unknown"
      >
        我不清楚具体时间
      </button>
    </div>
    <label class="text-field"
      >{{ label }}（{{ precision === "day" ? "日期" : "日期与时间" }}）<input
        v-if="precision === 'day'"
        v-model="date"
        type="date"
        :disabled="unknown" /><input
        v-else
        v-model="minute"
        type="datetime-local"
        step="60"
        :disabled="unknown"
    /></label>
    <p class="field-note">
      {{
        unknown
          ? "已选择时间未知，忽略日期与时间输入；再次点击按钮可取消。"
          : `时间按你的本地时区 ${timezone} 理解。`
      }}
    </p>
  </fieldset>
</template>

<style scoped src="./EventTimeField.css"></style>
