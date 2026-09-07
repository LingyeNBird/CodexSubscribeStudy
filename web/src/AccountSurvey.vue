<script setup lang="ts">
import { computed, reactive, ref } from "vue";

// ISO 3166-1 alpha-2 countries and territories; names are localized by the browser.
const countryCodes =
  "AD AE AF AG AI AL AM AO AQ AR AS AT AU AW AX AZ BA BB BD BE BF BG BH BI BJ BL BM BN BO BQ BR BS BT BV BW BY BZ CA CC CD CF CG CH CI CK CL CM CN CO CR CU CV CW CX CY CZ DE DJ DK DM DO DZ EC EE EG EH ER ES ET FI FJ FK FM FO FR GA GB GD GE GF GG GH GI GL GM GN GP GQ GR GS GT GU GW GY HK HM HN HR HT HU ID IE IL IM IN IO IQ IR IS IT JE JM JO JP KE KG KH KI KM KN KP KR KW KY KZ LA LB LC LI LK LR LS LT LU LV LY MA MC MD ME MF MG MH MK ML MM MN MO MP MQ MR MS MT MU MV MW MX MY MZ NA NC NE NF NG NI NL NO NP NR NU NZ OM PA PE PF PG PH PK PL PM PN PR PS PT PW PY QA RE RO RS RU RW SA SB SC SD SE SG SH SI SJ SK SL SM SN SO SR SS ST SV SX SY SZ TC TD TF TG TH TJ TK TL TM TN TO TR TT TV TW TZ UA UG UM US UY UZ VA VC VE VG VI VN VU WF WS YE YT ZA ZM ZW".split(
    " ",
  );
const regionNames = new Intl.DisplayNames(["zh-CN"], { type: "region" });
const countries = countryCodes
  .map((code) => ({ code, name: regionNames.of(code) ?? code }))
  .sort((a, b) => a.name.localeCompare(b.name, "zh-CN"));
const commonCountries = ["PH", "JP", "US", "BO"];
const status = ref("");
const degraded = computed(() => status.value === "降智" || status.value === "降智并封号");
const banned = computed(() => status.value === "封号" || status.value === "降智并封号");
const discovery = ref<string[]>([]);
const discoveryOther = ref("");
const region = ref("");
const plan = ref("");
const activation = ref("");
const usage = ref<string[]>([]);
const proxy = ref("");
const proxyOther = ref("");
const exit = reactive({ network: "", quality: "", address: "", country: "", unknown: false });
const official = reactive([
  { name: "Web 网页", selected: false, mode: "", connection: false },
  { name: "Codex Desktop", selected: false, mode: "", connection: true },
  { name: "Codex CI", selected: false, mode: "", connection: true },
]);
const thirdParty = reactive(
  [
    "Claude Code",
    "CI",
    "Pi",
    "oh-my-pi",
    "OpenCode",
    "Cursor",
    "Cline",
    "Roo Code",
    "Aider",
    "其他",
  ].map((name) => ({ name, selected: false, mode: "" })),
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
const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
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
        当前仅供前端体验，不会上传或持久保存答案；刷新或离开问卷后填写内容会清空。“降智”为填写者的观察判断，不代表已确认的模型能力变化。
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
          <label class="text-field" for="account-country"
            >国家或地区<select id="account-country" v-model="region">
              <option value="" disabled>请选择国家或地区</option>
              <option v-for="country in countries" :key="country.code" :value="country.code">
                {{ country.name }}
              </option>
            </select></label
          >
          <div class="country-shortcuts">
            <span>常用</span
            ><button
              v-for="code in commonCountries"
              :key="code"
              type="button"
              :aria-pressed="region === code"
              @click="region = code"
            >
              {{ regionNames.of(code) }}
            </button>
          </div>
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
              <label class="text-field"
                >IP 地址<input
                  v-model="exit.address"
                  type="text"
                  :disabled="exit.unknown"
                  placeholder="IPv4 或 IPv6 地址"
                  spellcheck="false" /></label
              ><label class="text-field"
                >IP 所在国家或地区<select v-model="exit.country" :disabled="exit.unknown">
                  <option value="" disabled>请选择国家或地区</option>
                  <option v-for="country in countries" :key="country.code" :value="country.code">
                    {{ country.name }}
                  </option>
                </select></label
              ><button
                class="unknown-button"
                type="button"
                :aria-pressed="exit.unknown"
                @click="exit.unknown = !exit.unknown"
              >
                我不知道
              </button>
            </div>
            <p class="field-note">
              “我不知道”仅针对 IP 地址与所在地区；选中后忽略这两项，取消后恢复填写。
            </p>
          </fieldset>
        </template>
        <fieldset>
          <legend>使用哪些官方工具？<span class="hint">可多选</span></legend>
          <div class="tool-list">
            <div
              v-for="tool in official"
              :key="tool.name"
              class="tool-row"
              :class="{ selected: tool.selected }"
            >
              <label class="tool-choice"
                ><input v-model="tool.selected" type="checkbox" />{{ tool.name }}</label
              >
              <fieldset v-if="tool.selected && tool.connection" class="tool-modes">
                <legend>{{ tool.name }} 连接方式</legend>
                <label v-for="mode in ['反代', '直登']" :key="mode"
                  ><input
                    v-model="tool.mode"
                    type="radio"
                    :name="`official-${tool.name}`"
                    :value="mode"
                  />{{ mode }}</label
                >
              </fieldset>
            </div>
          </div>
        </fieldset>
        <fieldset>
          <legend>使用哪些第三方工具？<span class="hint">可多选</span></legend>
          <div class="tool-list">
            <div
              v-for="tool in thirdParty"
              :key="tool.name"
              class="tool-row"
              :class="{ selected: tool.selected }"
            >
              <label class="tool-choice"
                ><input v-model="tool.selected" type="checkbox" />{{ tool.name }}</label
              >
              <fieldset v-if="tool.selected && tool.name !== 'Claude Code'" class="tool-modes">
                <legend>{{ tool.name }} 连接方式</legend>
                <label v-for="mode in ['直登（OAuth）', '反代']" :key="mode"
                  ><input
                    v-model="tool.mode"
                    type="radio"
                    :name="`third-party-${tool.name}`"
                    :value="mode"
                  />{{ mode }}</label
                >
              </fieldset>
              <label v-if="tool.selected && tool.name === '其他'" class="text-field other-tool"
                >其他工具名称<input
                  v-model="thirdPartyOther"
                  type="text"
                  placeholder="请输入工具名称"
              /></label>
            </div>
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
        <fieldset v-for="event in events" :key="event.id" class="follow-up">
          <legend>{{ event.label }}</legend>
          <div class="event-controls">
            <div class="choices">
              <label
                v-for="precision in [
                  { value: 'day', label: '只知道日期' },
                  { value: 'minute', label: '精确到分钟' },
                ]"
                :key="precision.value"
                class="choice"
                ><input
                  v-model="event.precision"
                  type="radio"
                  :name="`${event.id}-precision`"
                  :value="precision.value"
                  :disabled="event.unknown"
                />{{ precision.label }}</label
              >
            </div>
            <button
              type="button"
              class="unknown-button"
              :aria-pressed="event.unknown"
              @click="event.unknown = !event.unknown"
            >
              我不清楚具体时间
            </button>
          </div>
          <label class="text-field"
            >{{ event.label }}（{{ event.precision === "day" ? "日期" : "日期与时间" }}）<input
              v-if="event.precision === 'day'"
              v-model="event.date"
              type="date"
              :disabled="event.unknown" /><input
              v-else
              v-model="event.minute"
              type="datetime-local"
              step="60"
              :disabled="event.unknown"
          /></label>
          <p class="field-note">
            {{
              event.unknown
                ? "已选择时间未知，忽略日期与时间输入；再次点击按钮可取消。"
                : `时间按你的本地时区 ${timezone} 理解。`
            }}
          </p>
        </fieldset>
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
        <button class="button" type="button" disabled>暂未开放提交</button>
      </div>
    </form>
  </div>
</template>

<style scoped>
.survey-page {
  max-width: 970px;
  margin: auto;
  padding-bottom: 55px;
}
.survey-heading {
  margin: 4px 0 32px;
}
.survey-heading h1 {
  font-size: clamp(2rem, 4vw, 3.2rem);
  margin: 18px 0;
  line-height: 1.35;
}
.survey-heading > p {
  color: var(--muted);
  max-width: 650px;
}
.survey-notice {
  background: var(--sun);
  border: 1px solid #a49a64;
  border-radius: 12px;
  padding: 14px 18px;
  font-size: 0.82rem;
  line-height: 1.85;
  max-width: 800px;
}
.survey-form {
  display: grid;
  gap: 25px;
}
.survey-section {
  padding: 30px 34px;
  background: var(--paper);
  border: var(--line);
  border-radius: 22px;
  box-shadow: 4px 4px 0 var(--ink);
}
.survey-section-title {
  display: flex;
  align-items: center;
  gap: 16px;
  padding-bottom: 22px;
  margin-bottom: 24px;
  border-bottom: 1px solid #d9dfd4;
}
.survey-section-title .mini-icon {
  font-size: 0.8rem;
  font-weight: 750;
  flex-shrink: 0;
}
.survey-section-title h2 {
  font-size: 1.45rem;
}
.survey-section-title p {
  font-size: 0.83rem;
  color: var(--muted);
  margin: 5px 0 0;
}
.survey-form fieldset {
  border: 0;
  padding: 0;
  margin: 0 0 27px;
  min-width: 0;
}
.survey-form fieldset:last-child {
  margin-bottom: 0;
}
.survey-form legend {
  font-weight: 700;
  font-size: 0.96rem;
  padding: 0;
  margin-bottom: 12px;
  max-width: 100%;
}
.hint {
  font-size: 0.72rem;
  font-weight: 400;
  color: var(--muted);
  display: inline-block;
  margin-left: 12px;
}
.choices {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}
.choice {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  border: 1px solid #abb9ad;
  border-radius: 10px;
  padding: 9px 14px;
  cursor: pointer;
  font-size: 0.86rem;
  background: #fafaf4;
  min-height: 43px;
}
.choice:has(input:checked) {
  background: var(--mint);
  border-color: var(--ink);
}
.choice:hover {
  border-color: var(--ink);
}
.survey-form input[type="radio"],
.survey-form input[type="checkbox"] {
  accent-color: #35654b;
  width: 16px;
  height: 16px;
  margin: 0;
  flex-shrink: 0;
}
.text-field {
  display: flex;
  flex-direction: column;
  gap: 7px;
  font-size: 0.8rem;
  color: var(--muted);
  max-width: 100%;
  flex: 1;
  min-width: 0;
}
.choices + .text-field {
  margin-top: 14px;
}
.survey-form input:not([type="checkbox"]):not([type="radio"]),
.survey-form select {
  background: #fffefb;
  border: 1.5px solid #9eafa1;
  border-radius: 10px;
  padding: 10px 12px;
  font-size: 0.9rem;
  min-height: 45px;
  min-width: 0;
  width: 100%;
  box-sizing: border-box;
}
.survey-form input:focus-visible,
.survey-form select:focus-visible {
  outline: 3px solid #79a68c;
  outline-offset: 2px;
}
.survey-form input:disabled,
.survey-form select:disabled {
  opacity: 0.5;
  background: #eaece5;
}
.choice:has(input:disabled) {
  opacity: 0.5;
  cursor: default;
}
.survey-form .follow-up {
  border-left: 3px solid #afd2ba;
  padding-left: 19px;
  margin-top: 19px;
}
.country-shortcuts {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
  font-size: 0.75rem;
  color: var(--muted);
}
.country-shortcuts button,
.unknown-button {
  border: 1px solid #9eafa1;
  border-radius: 9px;
  background: transparent;
  padding: 8px 12px;
  font-size: 0.8rem;
  cursor: pointer;
  min-height: 39px;
}
.country-shortcuts button[aria-pressed="true"],
.unknown-button[aria-pressed="true"] {
  background: var(--mint);
  border-color: var(--ink);
  font-weight: 700;
}
.field-row {
  display: flex;
  align-items: end;
  gap: 12px;
  flex-wrap: wrap;
}
.field-row .text-field {
  flex: 1 1 180px;
}
.field-note {
  font-size: 0.74rem;
  color: var(--muted);
  line-height: 1.8;
  margin: 10px 0 0;
}
.survey-form .sublegend {
  font-size: 0.82rem;
  color: var(--muted);
}
.tool-list {
  display: grid;
  gap: 8px;
}
.tool-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  border: 1px solid #d3dbd0;
  border-radius: 11px;
  padding: 12px 15px;
  min-height: 49px;
}
.tool-row.selected {
  border-color: #71987e;
  background: #f0f6ed;
}
.tool-choice {
  display: flex;
  align-items: center;
  gap: 9px;
  font-weight: 600;
  font-size: 0.87rem;
  min-width: 165px;
  cursor: pointer;
  min-height: 27px;
}
.survey-form .tool-modes {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  margin: 0 0 0 auto;
}
.tool-modes legend {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip-path: inset(50%);
}
.tool-modes label {
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: 0.8rem;
  cursor: pointer;
  min-height: 30px;
}
.other-tool {
  flex-basis: 100%;
  margin-top: 6px;
}
.duration-row {
  max-width: 490px;
}
.duration-row .text-field:last-child {
  flex: 0 1 120px;
}
.event-controls {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 14px;
}
.event-controls + .text-field {
  max-width: 420px;
}
.survey-end {
  display: flex;
  justify-content: space-between;
  gap: 20px;
  align-items: center;
  padding: 10px 3px;
}
.survey-end p {
  color: var(--muted);
  font-size: 0.8rem;
  margin: 5px 0 0;
}
.survey-end button:disabled {
  cursor: not-allowed;
}
@media (max-width: 720px) {
  .survey-section {
    padding: 23px 19px;
    border-radius: 17px;
  }
  .survey-form {
    gap: 20px;
  }
  .survey-section-title {
    align-items: start;
    gap: 12px;
  }
  .survey-section-title h2 {
    font-size: 1.2rem;
  }
  .survey-form .follow-up {
    padding-left: 12px;
  }
  .choice {
    padding: 8px 11px;
  }
  .tool-row {
    padding: 10px 12px;
  }
  .survey-form .tool-modes {
    margin-left: 25px;
    flex-basis: 100%;
  }
  .field-row {
    align-items: stretch;
  }
  .unknown-button {
    align-self: start;
  }
  .survey-end {
    align-items: start;
    flex-direction: column;
  }
  .survey-heading {
    margin-bottom: 25px;
  }
  .survey-notice {
    padding: 12px 14px;
  }
  .survey-heading h1 {
    font-size: 2rem;
  }
  .event-controls {
    align-items: start;
    flex-direction: column;
  }
}
</style>
