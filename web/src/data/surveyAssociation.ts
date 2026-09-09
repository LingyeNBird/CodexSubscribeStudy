export interface BinaryAssociation {
  selected: { events: number; total: number };
  unselected: { events: number; total: number };
  phi: number | null;
}

export const tone = (phi: number | null) => {
  if (phi === null) return "unknown";
  const magnitude = Math.abs(phi);
  if (magnitude < 0.1) return "neutral";
  return `${phi > 0 ? "more" : "less"}-${magnitude >= 0.5 ? "strong" : magnitude >= 0.3 ? "moderate" : "weak"}`;
};
export const direction = (phi: number | null) =>
  phi === null
    ? "暂无有效比较"
    : Math.abs(phi) < 0.1
      ? "关联较弱"
      : phi > 0
        ? "↑ 异常更多"
        : "↓ 异常更少";
export const coefficient = (phi: number | null) =>
  phi === null ? "—" : `${phi > 0 ? "+" : ""}${phi.toFixed(3)}`;
export const percent = (value: BinaryAssociation["selected"]) =>
  value.total ? `${((100 * value.events) / value.total).toFixed(1)}%` : "—";
export const explanation = (value: BinaryAssociation) => {
  if (value.phi === null) return "缺少可比较样本，或变量没有变化，无法判断关联。";
  const difference =
    100 *
    (value.selected.events / value.selected.total -
      value.unselected.events / value.unselected.total);
  return difference === 0
    ? "两组报告该异常的比例相同。"
    : `选择该项的样本，异常比例${difference > 0 ? "高" : "低"} ${Math.abs(difference).toFixed(1)} 个百分点。`;
};
