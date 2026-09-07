<script setup lang="ts">
defineProps<{ name: string; group: string; modes: string[]; other?: boolean }>();
const selected = defineModel<boolean>("selected", { required: true });
const mode = defineModel<string>("mode", { required: true });
const otherName = defineModel<string>("otherName");
</script>

<template>
  <div class="tool-row" :class="{ selected }">
    <label class="tool-choice"><input v-model="selected" type="checkbox" />{{ name }}</label>
    <fieldset v-if="selected && modes.length" class="tool-modes">
      <legend>{{ name }} 连接方式</legend>
      <label v-for="option in modes" :key="option"
        ><input v-model="mode" type="radio" :name="group + '-' + name" :value="option" />{{
          option
        }}</label
      >
    </fieldset>
    <label v-if="selected && other" class="text-field other-tool"
      >其他工具名称<input v-model="otherName" type="text" placeholder="请输入工具名称"
    /></label>
  </div>
</template>

<style scoped src="./ToolChoice.css"></style>
