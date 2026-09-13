<script setup lang="ts">
import { computed, ref } from "vue";
import type { AdminRecord } from "../../data/adminApi";
import { trend } from "./adminModel";

const props = defineProps<{ records: AdminRecord[] }>();
const emit = defineEmits<{ drill: [from: string, to: string] }>();

const granularity = ref<"day" | "week" | "month">("day");
const metric = ref<"total" | "degraded" | "banned" | "limited">("total");
const metrics = [
  { id: "total", label: "问卷量" },
  { id: "degraded", label: "降智" },
  { id: "banned", label: "封号" },
  { id: "limited", label: "风控（限流）" },
] as const;

const buckets = computed(() => trend(props.records, granularity.value).buckets);
const unknown = computed(() => trend(props.records, granularity.value).unknown);
const peak = computed(() =>
  buckets.value.reduce((max, bucket) => Math.max(max, bucket[metric.value]), 0),
);

function value(bucket: (typeof buckets.value)[number]) {
  return bucket[metric.value];
}

/** The date range a bucket covers, so clicking it can filter on that period. */
function range(key: string) {
  if (granularity.value === "day") return { from: key, to: key };
  if (granularity.value === "week") {
    const end = new Date(`${key}T00:00:00Z`);
    end.setUTCDate(end.getUTCDate() + 6);
    return { from: key, to: end.toISOString().slice(0, 10) };
  }
  const [year, month] = key.split("-").map(Number);
  const end = new Date(Date.UTC(year, month, 0));
  return { from: `${key}-01`, to: end.toISOString().slice(0, 10) };
}

function drill(bucket: (typeof buckets.value)[number]) {
  const { from, to } = range(bucket.key);
  emit("drill", from, to);
}

function share(bucket: (typeof buckets.value)[number]) {
  return bucket.total
    ? `${((100 * value(bucket)) / bucket.total).toFixed(0)}%`
    : "—";
}
</script>

<template>
  <section class="trend">
    <div class="trend-controls">
      <label
        >粒度<select v-model="granularity">
          <option value="day">按天</option>
          <option value="week">按周</option>
          <option value="month">按月</option>
        </select></label
      >
      <div class="trend-metrics">
        <button
          v-for="item in metrics"
          :key="item.id"
          type="button"
          class="trend-metric"
          :class="{ 'is-on': metric === item.id }"
          @click="metric = item.id"
        >
          {{ item.label }}
        </button>
      </div>
      <p class="trend-note">点击柱子把该时间段加进筛选条件。</p>
    </div>

    <p v-if="unknown" class="trend-unknown">
      另有 {{ unknown.total }} 份问卷没有记录提交时间，未计入时间轴。
    </p>

    <div v-if="buckets.length" class="trend-scroll">
      <div class="trend-bars">
        <button
          v-for="bucket in buckets"
          :key="bucket.key"
          type="button"
          class="trend-bar"
          :title="`${bucket.key}：${value(bucket)} 份`"
          @click="drill(bucket)"
        >
          <span class="trend-value">{{ value(bucket) }}</span>
          <span
            class="trend-column"
            :style="{
              height: `${peak ? Math.max(3, (100 * value(bucket)) / peak) : 3}%`,
            }"
          ></span>
          <span class="trend-label"
            >{{ bucket.key }}<em>{{ share(bucket) }}</em></span
          >
        </button>
      </div>
    </div>
    <p v-else class="trend-empty">当前条件下没有带提交时间的问卷。</p>
  </section>
</template>

<style scoped src="./AdminTimeline.css"></style>
