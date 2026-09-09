import QRCode from "qrcode";
import type { SurveySummary } from "./surveyApi";
import {
  calculateAssociations,
  calculateSurveyStatistics,
  countStatuses,
  surveyOutcomes,
} from "./surveyStatistics";
import { direction, tone } from "./surveyAssociation";

export const surveyShareUrl =
  "https://codex.nightunderfly.online/#/studies/chatgpt-account-survey/results";
export const posterWidth = 1200;
export interface PosterSelection {
  overview: boolean;
  factorIds: string[];
  outcomeMask: number;
}
export interface SurveyPoster {
  svg: string;
  width: number;
  height: number;
}
export type PosterMeasure = (text: string, size: number, weight: number) => number;

const colors = {
  ink: "#26352e",
  paper: "#fffef9",
  background: "#f5f4ec",
  mint: "#d2efdf",
  violet: "#e4def9",
  peach: "#f9decf",
  sun: "#f3e7a9",
};
const toneColors: Record<string, string> = {
  "more-strong": "#edb5a0",
  "more-moderate": "#f5d0b8",
  "more-weak": "#fae7d6",
  "less-strong": "#b2d8c0",
  "less-moderate": "#d0e8d6",
  "less-weak": "#e1efe3",
  neutral: "#eeeee3",
  unknown: colors.paper,
};
const qr = QRCode.create(surveyShareUrl, { errorCorrectionLevel: "M" }).modules;
const fontFamily = "Inter, 'Noto Sans SC', 'Microsoft YaHei', sans-serif";

function xml(value: string): string {
  return value.replace(
    /[&<>"']/g,
    (character) =>
      ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&apos;" })[character]!,
  );
}
function wrap(
  text: string,
  width: number,
  size: number,
  weight: number,
  measure: PosterMeasure,
): string[] {
  const lines: string[] = [];
  let current = "";
  for (const character of text) {
    if (current && measure(current + character, size, weight) > width) {
      lines.push(current.trim());
      current = character;
    } else current += character;
  }
  if (current) lines.push(current.trim());
  return lines.length ? lines : [""];
}

export function buildSurveyPoster(
  summary: SurveySummary,
  selection: PosterSelection,
  icons: Record<string, string> = {},
  measure: PosterMeasure = (text, size) =>
    [...text].reduce(
      (width, character) =>
        width + size * (/[^\x00-\x7F]/.test(character) ? 1 : /[MW@]/.test(character) ? 0.95 : 0.65),
      0,
    ),
): SurveyPoster {
  const factors = summary.factors.filter((factor) => selection.factorIds.includes(factor.id));
  if (!selection.overview && !factors.length) throw new Error("请至少选择一项分享内容。");
  if (
    !Number.isInteger(selection.outcomeMask) ||
    selection.outcomeMask < 1 ||
    selection.outcomeMask > 7
  )
    throw new Error("请选择有效的异常统计范围。");
  const statistics = calculateSurveyStatistics(summary);
  const associations = new Map(
    calculateAssociations(factors, selection.outcomeMask).map((group) => [group.id, group]),
  );
  const outcomeLabel = surveyOutcomes
    .filter((item) => item.bit & selection.outcomeMask)
    .map((item) => item.label)
    .join("或");
  const parts: string[] = [];
  const rect = (
    x: number,
    y: number,
    width: number,
    height: number,
    fill: string,
    radius = 18,
    stroke = true,
  ) =>
    parts.push(
      `<rect x="${x}" y="${y}" width="${width}" height="${height}" rx="${radius}" fill="${fill}"${stroke ? ` stroke="${colors.ink}" stroke-width="3"` : ""}/>`,
    );
  const panel = (x: number, y: number, width: number, height: number, fill: string) => {
    rect(x + 7, y + 8, width, height, colors.ink, 18, false);
    rect(x, y, width, height, fill);
  };
  const text = (
    value: string,
    x: number,
    y: number,
    size: number,
    weight = 700,
    fill = colors.ink,
    anchor = "start",
  ) =>
    parts.push(
      `<text x="${x}" y="${y}" font-size="${size}" font-weight="${weight}" fill="${fill}" text-anchor="${anchor}">${xml(value)}</text>`,
    );

  rect(60, 54, 54, 54, colors.mint, 15);
  for (const [index, height] of [15, 28, 21].entries())
    rect(73 + index * 11, 93 - height, 7, height, colors.ink, 3, false);
  text("共研", 132, 94, 38, 800);
  text("ChatGPT 套餐", 60, 203, 64, 800);
  text("降智封号统计", 60, 292, 72, 800);
  text("社区问卷 · 一起看见真实使用情况", 64, 343, 25, 600);

  panel(874, 54, 266, 308, colors.paper);
  const cell = Math.floor(242 / (qr.size + 8));
  const qrSize = (qr.size + 8) * cell;
  const qrX = 874 + (266 - qrSize) / 2;
  const qrY = 70;
  rect(qrX, qrY, qrSize, qrSize, "#ffffff", 0, false);
  let qrPath = "";
  for (let row = 0; row < qr.size; row++)
    for (let column = 0; column < qr.size; column++) {
      if (qr.data[row * qr.size + column])
        qrPath += `M${qrX + (column + 4) * cell} ${qrY + (row + 4) * cell}h${cell}v${cell}h-${cell}z`;
    }
  parts.push(`<path d="${qrPath}" fill="${colors.ink}" shape-rendering="crispEdges"/>`);
  text("扫码查看 · 参与问卷", 1007, 340, 21, 700, colors.ink, "middle");
  let y = 421;
  if (selection.overview) {
    text("问卷统计", 60, y, 38, 800);
    y += 30;
    panel(60, y, 344, 264, colors.ink);
    text("问卷样本", 89, y + 48, 28, 700, colors.paper);
    text(String(statistics.total), 88, y + 182, 112, 800, colors.mint);
    text("份真实填写", 91, y + 230, 25, 600, colors.paper);
    const metrics = [
      { label: "报告降智", value: statistics.degraded, color: colors.violet },
      { label: "报告封号", value: statistics.banned, color: colors.peach },
      { label: "风控（限流）", value: statistics.limited, color: colors.sun },
      { label: "报告正常", value: statistics.normal, color: colors.mint },
    ];
    metrics.forEach((metric, index) => {
      const x = 428 + (index % 2) * 368;
      const top = y + Math.floor(index / 2) * 144;
      panel(x, top, 344, 120, metric.color);
      text(metric.label, x + 23, top + 39, 25);
      text(String(metric.value), x + 23, top + 96, 52, 800);
      text("份", x + 310, top + 93, 24, 600, colors.ink, "end");
    });
    y += 342;
  }

  if (factors.length) {
    const lines = wrap(`关联视角：${outcomeLabel}`, 1080, 32, 700, measure);
    for (const line of lines) {
      text(line, 60, y, 32, 700);
      y += 43;
    }
    y += 19;
  }
  for (const [factorIndex, factor] of factors.entries()) {
    rect(
      60,
      y,
      48,
      48,
      [colors.mint, colors.violet, colors.peach, colors.sun][factorIndex % 4]!,
      10,
    );
    text(String(factorIndex + 1).padStart(2, "0"), 84, y + 33, 23, 800, colors.ink, "middle");
    const headings = wrap(factor.title, 995, 36, 800, measure);
    headings.forEach((heading, index) => text(heading, 127, y + 36 + index * 45, 36, 800));
    y += Math.max(48, headings.length * 45) + 22;
    parts.push(`<path d="M60 ${y}H1140" stroke="${colors.ink}" stroke-width="4"/>`);
    y += 26;
    const columns = factor.options.length <= 2 ? 2 : 3;
    const width = (1080 - (columns - 1) * 24) / columns;
    const group = associations.get(factor.id);
    const isTool = factor.id === "tools" || factor.id === "official";
    for (let offset = 0; offset < factor.options.length; offset += columns) {
      const options = factor.options.slice(offset, offset + columns);
      const names = options.map((option) => wrap(option.label, width - 44, 30, 700, measure));
      const rowHeight = Math.max(
        ...names.map((lines) => (isTool ? 90 : 26) + lines.length * 39 + 86),
      );
      options.forEach((option, column) => {
        const x = 60 + column * (width + 24);
        const association = group?.rows[offset + column]?.association;
        const fill = association ? toneColors[tone(association.phi)]! : colors.violet;
        panel(x, y, width, rowHeight, fill);
        if (isTool) {
          const source = icons[option.label]?.match(/<svg\b[\s\S]*?<\/svg>/)?.[0];
          if (source)
            parts.push(
              source.replace(/<svg\b[^>]*>/, (tag) =>
                tag
                  .replace(/\s(?:width|height|x|y)="[^"]*"/g, "")
                  .replace(
                    "<svg",
                    `<svg x="${x + width / 2 - 26}" y="${y + 23}" width="52" height="52"`,
                  ),
              ),
            );
          else text(option.label.slice(0, 2), x + width / 2, y + 61, 35, 800, colors.ink, "middle");
        }
        const nameX = isTool ? x + width / 2 : x + 22;
        names[column]!.forEach((line, index) =>
          text(
            line,
            nameX,
            y + (isTool ? 119 : 55) + index * 39,
            30,
            700,
            colors.ink,
            isTool ? "middle" : "start",
          ),
        );
        let message: string;
        if (association) message = direction(association.phi);
        else {
          const total = countStatuses(factor.applicable, selection.outcomeMask);
          const count = countStatuses(option.counts, selection.outcomeMask);
          message = total ? `${count} 份 · ${((100 * count) / total).toFixed(1)}%` : "暂无样本";
        }
        text(message, x + width / 2, y + rowHeight - 31, 28, 800, colors.ink, "middle");
      });
      y += rowHeight + 24;
    }
    y += 35;
  }
  if (!factors.length) y += 5;
  rect(60, y, 1080, 156, colors.mint, 20);
  text("你的使用情况，也很重要。", 88, y + 62, 43, 800);
  text("扫码查看统计 · 参与问卷", 90, y + 113, 29, 700);
  text("↗", 1096, y + 108, 70, 700, colors.ink, "end");
  y += 209;
  const date = summary.range.computedAt.slice(0, 10).replaceAll("-", ".");
  text(`基于 ${statistics.total} 份问卷 · ${date}`, 60, y, 23, 600);
  text("样本关联 ≠ 因果", 1140, y, 23, 600, colors.ink, "end");
  text("codex.nightunderfly.online", 60, y + 40, 25, 700);
  const height = Math.ceil(y + 91);
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${posterWidth}" height="${height}" viewBox="0 0 ${posterWidth} ${height}" role="img" aria-label="共研问卷统计分享海报"><rect width="100%" height="100%" fill="${colors.background}"/><g font-family="${fontFamily}" color="${colors.ink}">${parts.join("")}</g></svg>`;
  return { svg, width: posterWidth, height };
}

export async function posterToPng(poster: SurveyPoster): Promise<Blob> {
  const url = URL.createObjectURL(new Blob([poster.svg], { type: "image/svg+xml;charset=utf-8" }));
  try {
    const image = new Image();
    await new Promise<void>((resolve, reject) => {
      image.onload = () => resolve();
      image.onerror = () => reject(new Error("海报图片加载失败，请重试。"));
      image.src = url;
    });
    const canvas = document.createElement("canvas");
    canvas.width = poster.width;
    canvas.height = poster.height;
    const context = canvas.getContext("2d");
    if (!context) throw new Error("当前浏览器无法生成图片。");
    context.drawImage(image, 0, 0);
    return await new Promise<Blob>((resolve, reject) =>
      canvas.toBlob(
        (blob) =>
          blob ? resolve(blob) : reject(new Error("图片导出失败，请减少分享内容后重试。")),
        "image/png",
      ),
    );
  } finally {
    URL.revokeObjectURL(url);
  }
}
