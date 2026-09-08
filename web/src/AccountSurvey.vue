<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import type { Component } from "vue";
import ClaudeCodeIcon from "./components/icons/ClaudeCodeIcon.vue";
import OhMyPiIcon from "./components/icons/OhMyPiIcon.vue";
import OpenCodeIcon from "./components/icons/OpenCodeIcon.vue";
import CursorIcon from "./components/icons/CursorIcon.vue";
import ClineIcon from "./components/icons/ClineIcon.vue";
import RooCodeIcon from "./components/icons/RooCodeIcon.vue";
import AiderIcon from "./components/icons/AiderIcon.vue";

import CountrySelect from "./components/survey/CountrySelect.vue";
import EventTimeField from "./components/survey/EventTimeField.vue";
import ToolChoice from "./components/survey/ToolChoice.vue";
import "./styles/survey-controls.css";
const status = ref("");
const degraded = computed(() => status.value === "降智" || status.value === "降智并封号");
const banned = computed(() => status.value === "封号" || status.value === "降智并封号");
const discovery = ref<string[]>([]);
const discoveryOther = ref("");
const degradationModels = [
  "GPT-6 Astra",
  "GPT-5.6 Sora",
  "GPT-5.6 Terra",
  "GPT-5.6 Luna",
  "GPT-5.5",
];
const selectedDegradationModels = ref<string[]>([]);
const region = ref("");
const plan = ref("");
const activation = ref("");
const usage = ref<string[]>([]);
const proxy = ref("");
const proxyOther = ref("");
const exit = reactive({ network: "", quality: "", country: "", unknown: false });
const official = reactive([
  { name: "Web 网页", selected: false, mode: "", connection: false },
  { name: "Codex Desktop", selected: false, mode: "", connection: true },
  { name: "Codex CI", selected: false, mode: "", connection: true },
]);
const thirdParty = reactive(
  ["Claude Code", "Pi", "oh-my-pi", "OpenCode", "Cursor", "Cline", "Roo Code", "Aider", "其他"].map(
    (name) => ({ name, selected: false, mode: "" }),
  ),
);
const toolIcons: Record<string, Component> = {
  "Claude Code": ClaudeCodeIcon,
  "oh-my-pi": OhMyPiIcon,
  OpenCode: OpenCodeIcon,
  Cursor: CursorIcon,
  Cline: ClineIcon,
  "Roo Code": RooCodeIcon,
  Aider: AiderIcon,
};
const thirdPartyOther = ref("");
const duration = ref<number | string>("");
const durationUnit = ref("天");
const degradationTime = reactive({
  id: "degradation",
  label: "降智的时间",
  precision: "day",
  date: "",
  minute: "",
  unknown: false,
});
const banTime = reactive({
  id: "ban",
  label: "封号的时间",
  precision: "day",
  date: "",
  minute: "",
  unknown: false,
});
const events = computed(() =>
  [degraded.value ? degradationTime : null, banned.value ? banTime : null].filter(
    (event): event is typeof degradationTime => event !== null,
  ),
);
const shared = ref("");
const people = ref<number | string>(2);
const concurrency = ref<number | string>("");
const concurrencyUnknown = ref(false);
const warning = ref("");
const truncated = ref("");
</script>

<template>
  <div class="survey-page">
    <a class="back" href="#/">← 全部研究</a>
    <header class="survey-heading">
      <span class="pill plain">研究 02 · 调查问卷</span>
      <h1>ChatGPT 套餐<br />降智封号统计</h1>
      <p>记录账号情况、使用方式与发生时间，帮助比较不同使用情形。</p>
      <div class="survey-notice" role="note">
        当前仅供前端体验，不会上传或持久保存答案；查看统计再返回时保留填写内容，刷新或离开本研究后清空。“降智”为填写者的观察判断，不代表已确认的模型能力变化。
      </div>
      <div class="survey-results-link">
        <span>想先看看大家的情况？</span>
        <a class="button" href="#/studies/chatgpt-account-survey/results">不填问卷，直接看统计 →</a>
      </div>
    </header>

    <form class="survey-form" @submit.prevent>
      <section class="survey-section">
        <div class="survey-section-title">
          <span class="mini-icon mint">01</span>
          <div>
            <h2>账号情况</h2>
            <p>先选择状态，相关问题会自动展开。</p>
          </div>
        </div>
        <fieldset>
          <legend>目前的账号情况</legend>
          <div class="choices">
            <label v-for="item in ['降智', '封号', '降智并封号', '正常']" :key="item" class="choice"
              ><input v-model="status" type="radio" name="status" :value="item" />{{ item }}</label
            >
          </div>
        </fieldset>
        <fieldset v-if="degraded" class="follow-up">
          <legend>降智的模型<span class="hint">可多选</span></legend>
          <div class="choices">
            <label v-for="model in degradationModels" :key="model" class="choice">
              <input v-model="selectedDegradationModels" type="checkbox" :value="model" />
              {{ model }}
            </label>
            <button
              type="button"
              class="unknown-button"
              @click="selectedDegradationModels = [...degradationModels]"
            >
              全部
            </button>
          </div>
        </fieldset>
        <fieldset v-if="degraded" class="follow-up">
          <legend>你是如何发现降智的？<span class="hint">可多选</span></legend>
          <div class="choices">
            <label
              v-for="item in ['画鹈鹕', 'juice值', '通过回答风格判断', '专业项目', '其他']"
              :key="item"
              class="choice"
              ><input v-model="discovery" type="checkbox" :value="item" />{{ item }}</label
            >
          </div>
          <label v-if="discovery.includes('其他')" class="text-field"
            >其他判断方式<input
              v-model="discoveryOther"
              type="text"
              placeholder="请描述你观察到的现象"
          /></label>
        </fieldset>
        <fieldset>
          <legend>账号地区</legend>
          <CountrySelect id="account-country" v-model="region" shortcuts />
        </fieldset>
        <fieldset>
          <legend>账号套餐级别</legend>
          <div class="choices">
            <label
              v-for="item in ['Free 免费', 'Go', 'Plus', 'Business', 'Pro 5X', 'Pro 20X']"
              :key="item"
              class="choice"
              ><input v-model="plan" type="radio" name="plan" :value="item" />{{ item }}</label
            >
          </div>
        </fieldset>
        <fieldset v-if="plan && plan !== 'Free 免费'" class="follow-up">
          <legend>账号开通方式</legend>
          <div class="choices">
            <label
              v-for="item in [
                '代充',
                '试用',
                '学生优惠',
                'Google Play 应用内购',
                'Apple 礼品卡',
                '网页付款',
              ]"
              :key="item"
              class="choice"
              ><input v-model="activation" type="radio" name="activation" :value="item" />{{
                item
              }}</label
            >
          </div>
        </fieldset>
      </section>

      <section class="survey-section">
        <div class="survey-section-title">
          <span class="mini-icon violet">02</span>
          <div>
            <h2>连接与工具</h2>
            <p>可同时选择多种使用方式和工具。</p>
          </div>
        </div>
        <fieldset>
          <legend>账号使用方式<span class="hint">可多选</span></legend>
          <div class="choices">
            <label v-for="item in ['直登', '反代']" :key="item" class="choice"
              ><input v-model="usage" type="checkbox" :value="item" />{{ item }}</label
            >
          </div>
        </fieldset>
        <template v-if="usage.includes('反代')">
          <fieldset class="follow-up">
            <legend>反代工具</legend>
            <div class="choices">
              <label v-for="item in ['sub2API', 'CPA', '其他']" :key="item" class="choice"
                ><input v-model="proxy" type="radio" name="proxy" :value="item" />{{ item }}</label
              >
            </div>
            <label v-if="proxy === '其他'" class="text-field"
              >其他反代工具<input v-model="proxyOther" type="text" placeholder="请输入工具名称"
            /></label>
          </fieldset>
          <fieldset class="follow-up">
            <legend>反代出口 IP</legend>
            <fieldset>
              <legend class="sublegend">网络类型</legend>
              <div class="choices">
                <label v-for="item in ['宽带', '机房']" :key="item" class="choice"
                  ><input v-model="exit.network" type="radio" name="exit-network" :value="item" />{{
                    item
                  }}</label
                >
              </div>
            </fieldset>
            <fieldset>
              <legend class="sublegend">IP 质量</legend>
              <div class="choices">
                <label v-for="item in ['优秀', '良好', '中', '差']" :key="item" class="choice"
                  ><input v-model="exit.quality" type="radio" name="exit-quality" :value="item" />{{
                    item
                  }}</label
                >
              </div>
            </fieldset>
            <div class="field-row">
              <CountrySelect
                v-model="exit.country"
                label="IP 所在国家或地区"
                :disabled="exit.unknown"
                shortcuts
              /><button
                class="unknown-button"
                type="button"
                :aria-pressed="exit.unknown"
                @click="exit.unknown = !exit.unknown"
              >
                我不知道
              </button>
            </div>
            <p class="field-note">
              “我不知道”仅针对 IP 所在国家或地区；选中后忽略该项，取消后恢复填写。不收集 IP 地址。
            </p>
          </fieldset>
        </template>
        <fieldset>
          <legend>使用哪些官方工具？<span class="hint">可多选</span></legend>
          <div class="tool-list">
            <ToolChoice
              v-for="tool in official"
              :key="tool.name"
              :name="tool.name"
              group="official"
              :modes="tool.connection ? ['反代', '直登'] : []"
              v-model:selected="tool.selected"
              v-model:mode="tool.mode"
            />
          </div>
        </fieldset>
        <fieldset>
          <legend>使用哪些第三方工具？<span class="hint">可多选</span></legend>
          <div class="tool-list">
            <ToolChoice
              v-for="tool in thirdParty"
              :key="tool.name"
              :name="tool.name"
              :icon="toolIcons[tool.name]"
              group="third-party"
              :modes="tool.name === 'Claude Code' ? [] : ['直登（OAuth）', '反代']"
              :other="tool.name === '其他'"
              v-model:selected="tool.selected"
              v-model:mode="tool.mode"
              v-model:other-name="thirdPartyOther"
            />
          </div>
          <p class="field-note">
            连接方式按你的实际使用情况填写，不代表每款工具都原生支持这两种方式。Claude Code
            不追加连接方式问题。
          </p>
        </fieldset>
      </section>

      <section class="survey-section">
        <div class="survey-section-title">
          <span class="mini-icon peach">03</span>
          <div>
            <h2>使用时长与事件</h2>
            <p>时间可以只精确到天，不确定时无需猜测。</p>
          </div>
        </div>
        <fieldset>
          <legend>总使用时长</legend>
          <div class="field-row duration-row">
            <label class="text-field"
              >数值<input
                v-model="duration"
                type="number"
                step="any"
                placeholder="请输入数字，可填写小数" /></label
            ><label class="text-field"
              >单位<select v-model="durationUnit">
                <option v-for="unit in ['天', '小时', '星期', '月', '年']" :key="unit">
                  {{ unit }}
                </option>
              </select></label
            >
          </div>
        </fieldset>
        <EventTimeField
          v-for="event in events"
          :key="event.id"
          :id="event.id"
          :label="event.label"
          v-model:precision="event.precision"
          v-model:date="event.date"
          v-model:minute="event.minute"
          v-model:unknown="event.unknown"
        />
      </section>

      <section class="survey-section">
        <div class="survey-section-title">
          <span class="mini-icon mint">04</span>
          <div>
            <h2>共享与使用经历</h2>
            <p>记录使用规模，以及是否遇到警告或内容截断。</p>
          </div>
        </div>
        <fieldset>
          <legend>是否分发？</legend>
          <div class="choices">
            <label v-for="item in ['是', '否']" :key="item" class="choice"
              ><input v-model="shared" type="radio" name="shared" :value="item" />{{ item }}</label
            >
          </div>
          <label v-if="shared === '是'" class="text-field follow-up"
            >分发人数<input v-model="people" type="number" min="2" step="1"
          /></label>
        </fieldset>
        <fieldset>
          <legend>最高账号并发是多少？</legend>
          <div class="field-row">
            <label class="text-field"
              >最高并发数<input
                v-model="concurrency"
                type="number"
                min="0"
                step="1"
                :disabled="concurrencyUnknown"
                placeholder="请输入最高并发数" /></label
            ><button
              class="unknown-button"
              type="button"
              :aria-pressed="concurrencyUnknown"
              @click="concurrencyUnknown = !concurrencyUnknown"
            >
              我不知道
            </button>
          </div>
          <p v-if="concurrencyUnknown" class="field-note">
            已选择并发未知，忽略输入数值；再次点击可取消。
          </p>
        </fieldset>
        <fieldset>
          <legend>是否收到过邮箱警告？</legend>
          <div class="choices">
            <label v-for="item in ['是', '否']" :key="item" class="choice"
              ><input v-model="warning" type="radio" name="warning" :value="item" />{{
                item
              }}</label
            >
          </div>
        </fieldset>
        <fieldset>
          <legend>在调用的过程中，是否因输入内容或输出内容被截断过？</legend>
          <div class="choices">
            <label v-for="item in ['是', '否', '我不知道']" :key="item" class="choice"
              ><input v-model="truncated" type="radio" name="truncated" :value="item" />{{
                item
              }}</label
            >
          </div>
        </fieldset>
      </section>
      <div class="survey-end">
        <div>
          <strong>问卷前端预览</strong>
          <p>暂未开放提交，当前填写内容不会发送到服务器。</p>
        </div>
        <a class="button primary" href="#/studies/chatgpt-account-survey/results">查看统计结果 →</a>
      </div>
    </form>
  </div>
</template>

<style scoped src="./AccountSurvey.css"></style>
