export const labels: Record<string, string> = {
  no_data: "等待第一份证据",
  uninformative: "贡献已收到 · 证据待积累",
  conditional: "联合证据已更新",
  sensitive: "对容量波动假设敏感",
  missing_snapshot: "缺少额度快照",
  capture_gap: "请求覆盖有缺口",
  missing_components: "缺少原始成本分项",
  unknown_control: "控制因素事实未知",
  invalid_fact: "原始事实不一致",
  reset_or_saturation: "重置或饱和观测",
  zero_progress: "零显示增量",
  external_usage_uncontrolled: "用量覆盖未确认",
  archived_source: "原始来源已归档",
  resource_limit: "等待资源处理",
};
export const n = (value: number | undefined) =>
  new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 }).format(value ?? 0);
