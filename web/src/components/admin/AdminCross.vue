<script setup lang="ts">
import { computed, ref, watch } from "vue";
import type { AdminCatalog, AdminRecord } from "../../data/adminApi";
import {
  STATUS_DIMENSION,
  crossTab,
  displayOf,
  listDimensions,
} from "./adminModel";

const props = defineProps<{
  records: AdminRecord[];
  catalog: AdminCatalog | null;
}>();

const emit = defineEmits<{ drill: [dimension: string, value: string] }>();

const dimensions = computed(() => listDimensions(props.catalog));
const rowDimension = ref("question:plans");
const colDimension = ref(STATUS_DIMENSION);
const percentModes = [
  { id: "count", label: "份数" },
  { id: "row", label: "占行" },
  { id: "col", label: "占列" },
] as const;
const percent = ref<"count" | "row" | "col">("count");

// Fall back to the first real question when a saved state names a dimension
// that no longer exists.
watch(dimensions, (list) => {
  const ids = list.map((dimension) => dimension.id);
  const fallback =
    ids.find((id) => id.startsWith("question:")) ?? STATUS_DIMENSION;
  if (!ids.includes(rowDimension.value)) rowDimension.value = fallback;
  if (!ids.includes(colDimension.value)) colDimension.value = STATUS_DIMENSION;
});

const table = computed(() =>
  crossTab(
    props.records,
    rowDimension.value,
    colDimension.value,
    props.catalog,
  ),
);

function cellText(count: number, rowIndex: number, colIndex: number) {
  if (percent.value === "count") return String(count);
  const base =
    percent.value === "row"
      ? table.value.rowTotals[rowIndex]
      : table.value.colTotals[colIndex];
  return base ? `${((100 * count) / base).toFixed(1)}%` : "—";
}

function title(dimension: string) {
  return (
    dimensions.value.find((item) => item.id === dimension)?.title ?? dimension
  );
}

function label(dimension: string, value: string) {
  return displayOf(dimension, value);
}

function drillRow(value: string) {
  emit("drill", rowDimension.value, value);
}

function drillCol(value: string) {
  emit("drill", colDimension.value, value);
}
</script>

<template>
  <section class="cross">
    <div class="cross-controls">
      <label
        >行<select v-model="rowDimension">
          <option
            v-for="dimension in dimensions"
            :key="dimension.id"
            :value="dimension.id"
          >
            {{ dimension.title }}
          </option>
        </select></label
      >
      <label
        >列<select v-model="colDimension">
          <option
            v-for="dimension in dimensions"
            :key="dimension.id"
            :value="dimension.id"
          >
            {{ dimension.title }}
          </option>
        </select></label
      >
      <div class="cross-modes">
        <button
          v-for="mode in percentModes"
          :key="mode.id"
          type="button"
          class="cross-mode"
          :class="{ 'is-on': percent === mode.id }"
          @click="percent = mode.id"
        >
          {{ mode.label }}
        </button>
      </div>
      <p class="cross-note">
        {{ table.total }} 份进入统计，点击表头或单元格可加进筛选条件。
      </p>
    </div>

    <p v-if="table.repeats" class="cross-warning">
      所选维度是多选，同一份问卷会落进多个单元格，所以行合计、列合计可能大于
      {{ table.total }}。
    </p>

    <div
      v-if="table.rowValues.length && table.colValues.length"
      class="cross-scroll"
    >
      <table class="cross-table">
        <thead>
          <tr>
            <th class="cross-corner">
              {{ title(rowDimension) }} \ {{ title(colDimension) }}
            </th>
            <th v-for="(col, colIndex) in table.colValues" :key="col">
              <button type="button" class="cross-head" @click="drillCol(col)">
                {{ label(colDimension, col) }}
                <em>{{ table.colTotals[colIndex] }}</em>
              </button>
            </th>
            <th class="cross-total-head">合计</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(row, rowIndex) in table.rowValues" :key="row">
            <th>
              <button type="button" class="cross-head" @click="drillRow(row)">
                {{ label(rowDimension, row) }}
                <em>{{ table.rowTotals[rowIndex] }}</em>
              </button>
            </th>
            <td
              v-for="(col, colIndex) in table.colValues"
              :key="col"
              :class="{ 'is-zero': !table.counts[rowIndex][colIndex] }"
            >
              {{
                cellText(
                  table.counts[rowIndex][colIndex] ?? 0,
                  rowIndex,
                  colIndex,
                )
              }}
            </td>
            <td class="cross-total">{{ table.rowTotals[rowIndex] }}</td>
          </tr>
        </tbody>
        <tfoot>
          <tr>
            <th>合计</th>
            <td
              v-for="(col, colIndex) in table.colValues"
              :key="col"
              class="cross-total"
            >
              {{ table.colTotals[colIndex] }}
            </td>
            <td class="cross-total">{{ table.grand }}</td>
          </tr>
        </tfoot>
      </table>
    </div>
    <p v-else class="cross-empty">当前条件下没有可交叉的样本。</p>
  </section>
</template>

<style scoped src="./AdminCross.css"></style>
