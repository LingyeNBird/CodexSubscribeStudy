import { binaryAssociation } from "./surveyAssociation";
export type SurveyOutcome = "degraded" | "banned";
export interface FactorDistribution {
  id: string;
  title: string;
  description: string;
  multiple: boolean;
  eligibility: string;
  groups: Record<SurveyOutcome, { total: number; rows: { label: string; count: number }[] }>;
}

// Entirely synthetic questionnaire records for layout preview, never collected answers.
// Every card and denominator is aggregated from this same set, including overlapping outcomes.
const choices = {
  models: ["GPT-6 Astra", "GPT-5.6 Sora", "GPT-5.6 Terra", "GPT-5.6 Luna", "GPT-5.5"],
  plans: ["Free 免费", "Go", "Plus", "Business", "Pro 5X", "Pro 20X"],
  tools: [
    "Claude Code",
    "Pi",
    "oh-my-pi",
    "OpenCode",
    "Cursor",
    "Cline",
    "Roo Code",
    "Aider",
    "其他",
  ],
  country: ["菲律宾", "日本", "美国", "玻利维亚", "其他国家或地区"],
  activation: ["代充", "试用", "学生优惠", "Google Play 应用内购", "Apple 礼品卡", "网页付款"],
  usage: ["直登", "反代"],
  proxy: ["sub2API", "CPA", "其他"],
  network: ["宽带", "机房"],
  quality: ["优秀", "良好", "中", "差"],
  exitCountry: ["菲律宾", "日本", "美国", "玻利维亚", "其他国家或地区", "我不知道"],
  official: ["Web 网页", "Codex Desktop", "Codex CI"],
  desktopMode: ["反代", "直登"],
  ciMode: ["反代", "直登"],
  thirdMode: ["直登（OAuth）", "反代"],
  duration: ["不足 1 天", "1–6 天", "7–29 天", "30–89 天", "90–364 天", "365 天及以上"],
  shared: ["是", "否"],
  people: ["2 人", "3–5 人", "6–10 人", "11–20 人", "21 人及以上"],
  concurrency: ["0", "1", "2–3", "4–5", "6–10", "11 及以上", "我不知道"],
  warning: ["是", "否"],
  truncated: ["是", "否", "我不知道"],
  discovery: ["画鹈鹕", "juice值", "通过回答风格判断", "专业项目", "其他"],
  eventTime: ["只知道日期", "精确到分钟", "我不清楚具体时间"],
} as const;
type FactorKey = keyof typeof choices;
interface ExampleRecord {
  degraded: boolean;
  banned: boolean;
  answers: Partial<Record<FactorKey, string[]>>;
}
function pick(key: FactorKey, index: number, multiple = false): string[] {
  const options = choices[key];
  const first = index % options.length;
  return multiple && index % 3 === 0
    ? [options[first]!, options[(first + 1) % options.length]!]
    : [options[first]!];
}
const records: ExampleRecord[] = Array.from({ length: 200 }, (_, i) => {
  const degraded = (i >= 112 && i < 152) || i >= 182;
  const banned = i >= 152;
  const answers: ExampleRecord["answers"] = {};
  for (const [offset, key] of (Object.keys(choices) as FactorKey[]).entries()) {
    answers[key] = pick(
      key,
      i * 7 + offset * 11 + Math.floor(i / 9),
      ["models", "tools", "usage", "official", "thirdMode", "discovery"].includes(key),
    );
  }
  if (!degraded) {
    delete answers.models;
    delete answers.discovery;
  }
  if (answers.plans![0] === "Free 免费") delete answers.activation;
  if (!answers.usage!.includes("反代")) {
    for (const key of ["proxy", "network", "quality", "exitCountry"] as const) delete answers[key];
  }
  if (!answers.official!.includes("Codex Desktop")) delete answers.desktopMode;
  if (!answers.official!.includes("Codex CI")) delete answers.ciMode;
  const connectedTools = answers.tools!.filter((tool) => tool !== "Claude Code");
  answers.thirdMode = [
    ...new Set(
      connectedTools.map((_, index) => choices.thirdMode[(i + index) % choices.thirdMode.length]!),
    ),
  ];
  if (!answers.thirdMode.length) delete answers.thirdMode;
  if (answers.shared![0] !== "是") delete answers.people;
  return { degraded, banned, answers };
});

export const surveyExampleOverview = {
  total: records.length,
  statuses: [
    { label: "正常", count: records.filter((r) => !r.degraded && !r.banned).length, tone: "mint" },
    { label: "降智", count: records.filter((r) => r.degraded && !r.banned).length, tone: "violet" },
    { label: "封号", count: records.filter((r) => !r.degraded && r.banned).length, tone: "peach" },
    {
      label: "降智并封号",
      count: records.filter((r) => r.degraded && r.banned).length,
      tone: "sun",
    },
  ],
};
const definitions: {
  key: FactorKey;
  title: string;
  description: string;
  multiple?: boolean;
  eligibility?: string;
}[] = [
  {
    key: "models",
    title: "降智的模型",
    description: "报告降智时选择的模型。",
    multiple: true,
    eligibility: "仅含同时报告降智并回答此题的问卷；封号视图不代表封号发生在哪个模型上。",
  },
  { key: "plans", title: "套餐级别", description: "所选异常样本使用的套餐，不是各套餐的异常率。" },
  {
    key: "tools",
    title: "第三方工具",
    description: "所选异常样本使用的第三方工具。",
    multiple: true,
  },
  { key: "country", title: "账号地区", description: "四个常用国家单列，其余国家与地区合并展示。" },
  {
    key: "activation",
    title: "账号开通方式",
    description: "当前套餐的开通途径。",
    eligibility: "仅含非 Free 且回答开通方式的问卷。",
  },
  { key: "usage", title: "账号使用方式", description: "直登与反代可以同时使用。", multiple: true },
  {
    key: "proxy",
    title: "反代工具",
    description: "选择反代时使用的工具。",
    eligibility: "仅含选择反代且回答此题的问卷。",
  },
  {
    key: "network",
    title: "反代出口网络",
    description: "出口为宽带还是机房网络。",
    eligibility: "仅含选择反代且回答此题的问卷。",
  },
  {
    key: "quality",
    title: "反代 IP 质量",
    description: "填写者自评的质量等级，并非统一检测结果。",
    eligibility: "仅含选择反代且回答此题的问卷。",
  },
  {
    key: "exitCountry",
    title: "反代出口地区",
    description: "仅统计国家与地区，不收集 IP 地址。",
    eligibility: "仅含选择反代的有效回答；明确选择“不知道”单列，不等于漏答。",
  },
  {
    key: "official",
    title: "官方工具",
    description: "网页、桌面与 CI 工具的使用情况。",
    multiple: true,
  },
  {
    key: "desktopMode",
    title: "Codex Desktop 连接方式",
    description: "使用该官方工具时的连接方式。",
    eligibility: "仅含选择 Codex Desktop 且回答连接方式的问卷。",
  },
  {
    key: "ciMode",
    title: "Codex CI 连接方式",
    description: "使用该官方工具时的连接方式。",
    eligibility: "仅含选择 Codex CI 且回答连接方式的问卷。",
  },
  {
    key: "thirdMode",
    title: "第三方工具连接方式",
    description: "按问卷去重：不同工具可使用不同连接方式。",
    multiple: true,
    eligibility: "仅含选择非 Claude Code 工具并回答连接方式的问卷；Claude Code 不询问此项。",
  },
  {
    key: "duration",
    title: "总使用时长",
    description: "按天分组：小时÷24、星期×7、月×30、年×365；月年为统计换算约定。",
  },
  { key: "shared", title: "是否分发", description: "所选异常样本是否向他人分发。" },
  {
    key: "people",
    title: "分发人数",
    description: "分发范围按人数区间展示。",
    eligibility: "仅含选择分发且提供人数的问卷。",
  },
  { key: "concurrency", title: "最高账号并发", description: "按最高并发分组，“不知道”单独列出。" },
  {
    key: "warning",
    title: "邮箱警告",
    description: "是否收到过邮箱警告，不推断其与异常的先后顺序。",
  },
  { key: "truncated", title: "输入或输出截断", description: "填写者是否报告过内容截断。" },
  {
    key: "discovery",
    title: "降智发现方式",
    description: "填写者依据哪些方式判断降智。",
    multiple: true,
    eligibility: "仅含同时报告降智并回答此题的问卷；封号视图只包含两种状态重叠的样本。",
  },
  {
    key: "eventTime",
    title: "事件时间记录",
    description: "降智视图对应降智时间，封号视图对应封号时间；仅比较记录精度，不推断事件原因。",
  },
];
export const surveyExampleFactors: FactorDistribution[] = definitions.map((definition) => {
  const group = (outcome: SurveyOutcome) => {
    const applicable = records.filter(
      (record) => record[outcome] && record.answers[definition.key]?.length,
    );
    return {
      total: applicable.length,
      rows: choices[definition.key].map((label) => ({
        label,
        count: applicable.filter((record) => record.answers[definition.key]!.includes(label))
          .length,
      })),
    };
  };
  return {
    id: definition.key,
    title: definition.title,
    description: definition.description,
    multiple: definition.multiple ?? false,
    eligibility: definition.eligibility ?? "分母为所选异常状态中回答此题的问卷；漏答不计入。",
    groups: { degraded: group("degraded"), banned: group("banned") },
  };
});

// Outcome-gated questions have no comparable non-outcome group.
export const surveyExampleAssociations = definitions
  .filter((definition) => !["models", "discovery", "eventTime"].includes(definition.key))
  .map((definition) => {
    const applicable = records.filter((record) => record.answers[definition.key]?.length);
    return {
      id: definition.key,
      title: definition.title,
      scope: definition.eligibility ?? "所有回答此题的问卷，包括正常、降智、封号和两者兼有。",
      total: applicable.length,
      rows: choices[definition.key].map((label) => {
        const selected = applicable.filter((record) =>
          record.answers[definition.key]!.includes(label),
        );
        const unselected = applicable.filter(
          (record) => !record.answers[definition.key]!.includes(label),
        );
        const measure = (outcome: SurveyOutcome) => {
          const a = selected.filter((record) => record[outcome]).length;
          const c = unselected.filter((record) => record[outcome]).length;
          return binaryAssociation(a, selected.length - a, c, unselected.length - c);
        };
        return { label, degraded: measure("degraded"), banned: measure("banned") };
      }),
    };
  });

export const surveyExampleOutcomeAssociation = binaryAssociation(
  records.filter((record) => record.degraded && record.banned).length,
  records.filter((record) => record.degraded && !record.banned).length,
  records.filter((record) => !record.degraded && record.banned).length,
  records.filter((record) => !record.degraded && !record.banned).length,
);
