<script setup lang="ts">
import { ref } from "vue";
const model = defineModel<number[] | undefined>();
const props = withDefaults(defineProps<{ readonly?: boolean }>(), { readonly: false });
const grid = ref<HTMLElement>();
const hours = Array.from({ length: 24 }, (_, hour) => hour);
const level = (hour: number) => model.value?.[hour] ?? 0;
const percentage = (hour: number) => `${((level(hour) / 24) * 100).toFixed(1)}%`;
let pointer: number | null = null;
let previous: { x: number; y: number } | null = null;
function end(event?: PointerEvent) {
  if (event && event.pointerId !== pointer) return;
  const id = pointer;
  pointer = null;
  previous = null;
  if (id !== null && grid.value?.hasPointerCapture(id)) grid.value.releasePointerCapture(id);
}
function paint(event: PointerEvent) {
  const box = grid.value!.getBoundingClientRect();
  if (!box.width || !box.height) return;
  const point = {
    x: (24 * (event.clientX - box.left)) / box.width,
    y: (24 * (event.clientY - box.top)) / box.height,
  };
  if (point.x < 0 || point.x >= 24 || point.y < 0 || point.y >= 24) {
    previous = null;
    return;
  }
  const values = model.value ? [...model.value] : Array<number>(24).fill(0);
  const from = previous ?? point;
  const steps = Math.max(
    1,
    Math.ceil(Math.max(Math.abs(point.x - from.x), Math.abs(point.y - from.y)) * 2),
  );
  for (let step = 1; step <= steps; step++) {
    const x = from.x + ((point.x - from.x) * step) / steps;
    const y = from.y + ((point.y - from.y) * step) / steps;
    values[Math.min(23, Math.floor(y))] = Math.min(24, Math.floor(x) + 1);
  }
  previous = point;
  model.value = values;
}
function start(event: PointerEvent) {
  if (
    props.readonly ||
    (event.type === "pointerdown" && event.button !== 0) ||
    !(event.buttons & 1) ||
    event.isPrimary === false ||
    pointer !== null
  )
    return;
  event.preventDefault();
  pointer = event.pointerId;
  grid.value!.setPointerCapture(pointer);
  paint(event);
}
function move(event: PointerEvent) {
  if (props.readonly || event.isPrimary === false) return;
  if (!(event.buttons & 1)) {
    end();
    return;
  }
  if (pointer === null) {
    start(event);
    return;
  }
  if (pointer === event.pointerId) paint(event);
}
function keydown(event: KeyboardEvent, hour: number) {
  if (props.readonly) return;
  const actions: Record<string, number> = {
    ArrowRight: level(hour) + 1,
    ArrowLeft: level(hour) - 1,
    Home: 0,
    End: 24,
    Delete: 0,
    Backspace: 0,
  };
  const value = actions[event.key];
  if (value === undefined) return;
  event.preventDefault();
  const values = model.value ? [...model.value] : Array<number>(24).fill(0);
  values[hour] = Math.max(0, Math.min(24, value));
  model.value = values;
}
function clear() {
  end();
  model.value = undefined;
}
function clearHour(hour: number) {
  const values = model.value ? [...model.value] : Array<number>(24).fill(0);
  values[hour] = 0;
  model.value = values;
}
</script>

<template>
  <div class="usage-pattern" :class="{ 'is-readonly': readonly }">
    <p v-if="!readonly" class="pattern-meaning">
      按当地时间填写，50% 表示该小时通常约有30分钟在使用。
    </p>
    <div class="pattern-toolbar" v-if="!readonly">
      <p>点击或按住左键拖动绘制；点击左侧时间可清空该行。</p>
      <button class="unknown-button" type="button" @click="clear">清空</button>
    </div>
    <div class="pattern-axis" aria-hidden="true">
      <span>0%</span><span>50%</span><span>100%</span>
    </div>
    <div class="pattern-board">
      <div class="pattern-hours">
        <template v-for="hour in hours" :key="hour">
          <span v-if="readonly">{{ hour }} 点</span>
          <button v-else type="button" :aria-label="`清空 ${hour} 点`" @click="clearHour(hour)">
            {{ hour }} 点
          </button>
        </template>
      </div>
      <div
        ref="grid"
        class="pattern-grid"
        @pointerdown="start"
        @pointermove="move"
        @pointerup="end"
        @pointercancel="end"
        @lostpointercapture="end"
        @dragstart.prevent
      >
        <div
          v-for="hour in hours"
          :key="hour"
          class="pattern-row"
          :role="readonly ? 'img' : 'slider'"
          :tabindex="readonly ? undefined : 0"
          :aria-label="`${hour} 点使用程度${readonly ? '：' + percentage(hour) : ''}`"
          :aria-valuemin="readonly ? undefined : 0"
          :aria-valuemax="readonly ? undefined : 24"
          :aria-valuenow="readonly ? undefined : level(hour)"
          :aria-valuetext="readonly ? undefined : percentage(hour)"
          :style="{ '--level': level(hour) }"
          @keydown="keydown($event, hour)"
        >
          <span
            v-for="cell in 24"
            :key="cell"
            class="pattern-cell"
            :style="{ '--column': cell - 1 }"
            aria-hidden="true"
          ></span>
        </div>
      </div>
      <div class="pattern-values" aria-hidden="true">
        <span v-for="hour in hours" :key="hour">{{ percentage(hour) }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped src="./UsagePattern.css"></style>
