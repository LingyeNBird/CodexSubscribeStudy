<script setup lang="ts">
import { computed, onActivated, reactive, ref } from "vue";
import { surveyRequest, type SurveySubmission } from "./data/surveyApi";
import { toolIcons } from "./components/icons/toolIcons";

import CountrySelect from "./components/survey/CountrySelect.vue";
import EventTimeField from "./components/survey/EventTimeField.vue";
import ToolChoice from "./components/survey/ToolChoice.vue";
import UsagePattern from "./components/survey/UsagePattern.vue";
import IPRiskSlider from "./components/survey/IPRiskSlider.vue";
import "./styles/survey-controls.css";
const accountState = ref("");
const selectedIssues = ref<string[]>([]);
const status = computed(() =>
  accountState.value === "正常"
    ? ["正常"]
    : accountState.value === "存在异常"
      ? [...selectedIssues.value]
      : [],
);
const degraded = computed(() => status.value.includes("降智"));
const banned = computed(() => status.value.includes("封号"));
const limited = computed(() => status.value.includes("风控（限流）"));
const limitedDiscovery = ref<string[]>([]);
const limitedDiscoveryOther = ref("");
const usagePattern = ref<number[]>();
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
const exit = reactive<{
  network: string;
  stability: string;
  risk: number | undefined;
  country: string;
  unknown: boolean;
}>({ network: "", stability: "", risk: undefined, country: "", unknown: false });
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
const submittedBefore = ref(false);
const submitting = ref(false);
const submitError = ref("");
const submittedStorageKey = "chatgpt-account-survey:submitted";
onActivated(() => {
  try {
    submittedBefore.value = window.localStorage.getItem(submittedStorageKey) === "1";
  } catch {
    // Storage restrictions must not prevent filling or submitting the questionnaire.
  }
});
async function submitSurvey() {
  if (submitting.value) return;
  submitError.value = "";
  if (!status.value.length || !plan.value) {
    submitError.value = "请选择目前的账号情况和账号套餐级别。";
    return;
  }
  const payload: SurveySubmission = { status: status.value, answers: {}, details: {} };
  if (usagePattern.value) payload.usagePattern = [...usagePattern.value];
  const answer = (key: string, value: string | string[]) => {
    const values = Array.isArray(value) ? value : value ? [value] : [];
    if (values.length) payload.answers[key] = values;
  };
  const detail = (key: string, value: string | number) => {
    const text = String(value).trim();
    if (text) payload.details[key] = text;
  };
  answer("plans", plan.value);
  detail("country", region.value);
  if (plan.value !== "Free 免费") answer("activation", activation.value);
  if (degraded.value) {
    answer("models", selectedDegradationModels.value);
    answer("discovery", discovery.value);
    if (discovery.value.includes("其他")) detail("discoveryOther", discoveryOther.value);
  }
  if (limited.value) {
    answer("limitedDiscovery", limitedDiscovery.value);
    if (limitedDiscovery.value.includes("其他"))
      detail("limitedDiscoveryOther", limitedDiscoveryOther.value);
  }
  answer("usage", usage.value);
  if (usage.value.includes("反代")) {
    answer("proxy", proxy.value);
    if (proxy.value === "其他") detail("proxyOther", proxyOther.value);
  }
  if (usage.value.length) {
    answer("network", exit.network);
    answer("ipStability", exit.stability);
    if (exit.risk !== undefined) payload.ipRisk = exit.risk;
    detail("exitCountry", exit.unknown ? "unknown" : exit.country);
  }
  answer(
    "official",
    official.filter((tool) => tool.selected).map((tool) => tool.name),
  );
  for (const tool of official) {
    if (tool.selected && tool.connection)
      answer(tool.name === "Codex Desktop" ? "desktopMode" : "ciMode", tool.mode);
  }
  answer(
    "tools",
    thirdParty.filter((tool) => tool.selected).map((tool) => tool.name),
  );
  for (const tool of thirdParty) {
    if (!tool.selected) continue;
    if (tool.name !== "Claude Code") detail(`toolMode:${tool.name}`, tool.mode);
    if (tool.name === "其他") detail("thirdPartyOther", thirdPartyOther.value);
  }
  if (duration.value !== "") {
    detail("duration", duration.value);
    detail("durationUnit", durationUnit.value);
  }
  for (const event of events.value) {
    detail(
      event.id === "degradation" ? "degradationTime" : "banTime",
      event.unknown ? "unknown" : event.precision === "day" ? event.date : event.minute,
    );
  }
  answer("shared", shared.value);
  if (shared.value === "是") detail("people", people.value);
  detail("concurrency", concurrencyUnknown.value ? "unknown" : concurrency.value);
  answer("warning", warning.value);
  answer("truncated", truncated.value);
  submitting.value = true;
  try {
    const result = await surveyRequest<{ accepted: boolean }>("submissions", payload);
    if (!result.accepted) throw new Error("提交未完成，请稍后重试。");
    submittedBefore.value = true;
    try {
      window.localStorage.setItem(submittedStorageKey, "1");
    } catch {
      // The submission succeeded even when browser storage is unavailable.
    }
    window.location.hash = "/studies/chatgpt-account-survey/results";
  } catch (error) {
    submitError.value = error instanceof Error ? error.message : "提交失败，请稍后重试。";
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <div class="survey-page">
    <a class="back" href="#/">← 全部研究</a>
    <header class="survey-heading">
      <span class="pill plain">研究 02 · 调查问卷</span>
      <h1>ChatGPT 套餐<br />降智封号统计</h1>
      <p>记录账号情况、使用方式与发生时间，帮助比较不同使用情形。</p>
      <div class="survey-notice" role="note">
        “降智”为填写者的观察判断，不代表已确认的模型能力变化。账号情况和套餐级别为必填，其余问题可按实际情况填写。
      </div>
      <div class="survey-results-link">
        <span>想先看看大家的情况？</span>
        <a
          class="button"
          :class="{ primary: submittedBefore }"
          href="#/studies/chatgpt-account-survey/results"
          >{{ submittedBefore ? "你貌似已提交过，直接看统计 →" : "不填问卷，直接看统计 →" }}</a
        >
      </div>
    </header>

    <form class="survey-form" @submit.prevent="submitSurvey" :aria-busy="submitting">
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
            <label v-for="item in ['正常', '存在异常']" :key="item" class="choice"
              ><input v-model="accountState" type="radio" name="account-state" :value="item" />{{
                item
              }}</label
            >
          </div>
        </fieldset>
        <fieldset v-if="accountState === '存在异常'" class="follow-up">
          <legend><span class="multiple-badge">多选</span>出现了哪些情况？</legend>
          <div class="choices">
            <label v-for="item in ['降智', '封号', '风控（限流）']" :key="item" class="choice">
              <input
                v-model="selectedIssues"
                type="checkbox"
                name="account-issues"
                :value="item"
              />{{ item }}
            </label>
          </div>
        </fieldset>
        <fieldset v-if="degraded" class="follow-up">
          <legend><span class="multiple-badge">多选</span>降智的模型</legend>
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
          <legend><span class="multiple-badge">多选</span>你是如何发现降智的？</legend>
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
        <fieldset v-if="limited" class="follow-up">
          <legend><span class="multiple-badge">多选</span>如何识别出风控的？</legend>
          <div class="choices">
            <label
              v-for="item in ['容量达到上限', '服务不可用', '周限额度明显骤降', '其他']"
              :key="item"
              class="choice"
            >
              <input
                v-model="limitedDiscovery"
                type="checkbox"
                name="limited-discovery"
                :value="item"
              />{{ item }}
            </label>
          </div>
          <p class="field-note">
            因为使用 6 Astra 导致的周限额度下降，不要选择“周限额度明显骤降”。
          </p>
          <label v-if="limitedDiscovery.includes('其他')" class="text-field">
            其他识别方式<input
              v-model="limitedDiscoveryOther"
              name="limited-discovery-other"
              type="text"
              placeholder="请描述你观察到的情况"
            />
          </label>
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
          <legend><span class="multiple-badge">多选</span>账号使用方式</legend>
          <p id="usage-multiple-hint" class="multiple-hint">
            如果同时使用直登和反代，请<strong>两项都选上</strong>。
          </p>
          <div class="choices">
            <label v-for="item in ['直登', '反代']" :key="item" class="choice"
              ><input
                v-model="usage"
                type="checkbox"
                :value="item"
                aria-describedby="usage-multiple-hint"
              />{{ item }}</label
            >
          </div>
        </fieldset>
        <fieldset v-if="usage.includes('反代')" class="follow-up">
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
        <fieldset v-if="usage.length" class="follow-up">
          <legend>反代出口 IP 或者直登 IP</legend>
          <fieldset>
            <legend class="sublegend">网络类型</legend>
            <div class="choices">
              <label v-for="item in ['家宽', '机房', '机场']" :key="item" class="choice"
                ><input v-model="exit.network" type="radio" name="exit-network" :value="item" />{{
                  item
                }}</label
              >
            </div>
          </fieldset>
          <fieldset>
            <legend class="sublegend">IP 是否固定</legend>
            <div class="choices">
              <label v-for="item in ['固定 IP', 'IP 乱飞']" :key="item" class="choice">
                <input v-model="exit.stability" type="radio" name="ip-stability" :value="item" />{{
                  item
                }}
              </label>
            </div>
          </fieldset>
          <fieldset>
            <legend class="sublegend">IP 风险程度</legend>
            <IPRiskSlider v-model="exit.risk" />
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
            “我不知道”仅针对 IP 所在国家或地区；选中后忽略该项，取消后恢复填写。
          </p>
        </fieldset>
        <fieldset>
          <legend><span class="multiple-badge">多选</span>使用哪些官方工具？</legend>
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
          <legend><span class="multiple-badge">多选</span>使用哪些第三方工具？</legend>
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
            <h2>账号时长与事件</h2>
            <p>时间可以只精确到天，不确定时无需猜测。</p>
          </div>
        </div>
        <fieldset>
          <legend>你平时的使用规律</legend>
          <UsagePattern v-model="usagePattern" />
        </fieldset>
        <fieldset>
          <legend>账号存活时长</legend>
          <div class="field-row duration-row">
            <label class="text-field"
              >数值<input
                v-model="duration"
                type="number"
                min="0"
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
          <strong>提交问卷</strong>
          <p v-if="submitError" role="alert">{{ submitError }}</p>
          <p v-else>提交后即可查看统计结果。</p>
        </div>
        <button class="button primary" type="submit" :disabled="submitting">
          {{ submitting ? "正在提交…" : "提交并查看统计 →" }}
        </button>
      </div>
    </form>
  </div>
</template>

<style scoped src="./AccountSurvey.css"></style>
