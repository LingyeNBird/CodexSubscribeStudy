import assert from "node:assert/strict";
import { test } from "node:test";
import type {
  StatusCounts,
  SurveyFactorSummary,
  SurveyOutcome,
  SurveySummary,
} from "../src/data/surveyApi";
import {
  calculateAssociation,
  calculateAssociations,
  calculateSurveyStatistics,
  effectiveOutcomeMask,
} from "../src/data/surveyStatistics.ts";

test("all outcome unions count overlapping submissions once and recompute phi", () => {
  const states: SurveyOutcome[][] = [
    [],
    ["degraded"],
    ["banned"],
    ["degraded", "banned"],
    ["limited"],
    ["degraded", "limited"],
    ["banned", "limited"],
    ["degraded", "banned", "limited"],
  ];
  const applicable: StatusCounts = [1, 1, 1, 1, 1, 1, 1, 1];
  const selected: StatusCounts = [0, 1, 0, 1, 0, 1, 0, 1];
  for (const wanted of [
    ["degraded"],
    ["banned"],
    ["limited"],
    ["degraded", "banned"],
    ["degraded", "limited"],
    ["banned", "limited"],
    ["degraded", "banned", "limited"],
  ] as SurveyOutcome[][]) {
    const mask = effectiveOutcomeMask(wanted, "degraded");
    const table = [0, 0, 0, 0];
    states.forEach((tags, index) => {
      const slot = (selected[index] ? 0 : 2) + (tags.some((tag) => wanted.includes(tag)) ? 0 : 1);
      table[slot]!++;
    });
    const [a, b, c, d] = table as [number, number, number, number];
    const actual = calculateAssociation(selected, applicable, mask);
    assert.deepEqual(actual.selected, { events: a, total: a + b });
    assert.deepEqual(actual.unselected, { events: c, total: c + d });
    const expected = (a * d - b * c) / Math.sqrt((a + b) * (c + d) * (a + c) * (b + d));
    assert(Math.abs(actual.phi! - expected) < 1e-12);
  }
  assert.equal(calculateAssociation(selected, applicable, 3).selected.events, 4);
  assert.equal(effectiveOutcomeMask([], "banned"), 2);
  assert.equal(effectiveOutcomeMask([], "limited"), 4);
});

test("missing answers stay out of comparisons and degenerate tables are not zero-risk estimates", () => {
  const zero: StatusCounts = [0, 0, 0, 0, 0, 0, 0, 0];
  const answered: StatusCounts = [2, 0, 0, 1, 0, 0, 0, 0];
  const option: StatusCounts = [0, 0, 0, 1, 0, 0, 0, 0];
  assert.equal(calculateAssociation(zero, answered, 3).phi, null);
  assert.equal(calculateAssociation(answered, answered, 3).phi, null);
  assert.equal(calculateAssociation(zero, zero, 3).phi, null);
  const factor: SurveyFactorSummary = {
    id: "tools",
    title: "工具",
    description: "",
    multiple: true,
    distributionScope: "answered",
    associationScope: "answered",
    comparable: true,
    applicable: answered,
    options: [
      { label: "A", counts: option },
      { label: "B", counts: option },
    ],
  };
  const summary: SurveySummary = {
    version: 2,
    catalogDigest: "test",
    statuses: [20, 0, 0, 1, 0, 0, 0, 0],
    factors: [factor, { ...factor, id: "discovery", comparable: false }],
    usagePattern: { total: 2, sums: Array.from({ length: 24 }, (_, hour) => hour) },
    range: {
      firstSubmissionId: 1,
      lastSubmissionId: 21,
      firstSubmittedAt: null,
      lastSubmittedAt: null,
      unknownTimeCount: 21,
      computedAt: "2026-09-09T00:00:00Z",
    },
  };
  const associations = calculateAssociations(summary.factors, 3);
  assert.deepEqual(
    associations.map((group) => group.id),
    ["tools"],
  );
  assert.equal(associations[0]!.total, 3);
  assert.deepEqual(associations[0]!.rows[0]!.association, {
    selected: { events: 1, total: 1 },
    unselected: { events: 0, total: 2 },
    phi: 1,
  });
  const statistics = calculateSurveyStatistics(summary);
  assert.equal(statistics.total, 21);
  assert.equal(statistics.degraded, 1);
  assert.equal(statistics.banned, 1);
  assert.equal(statistics.factors[0]!.groups.degraded.total, 1);
  assert.equal(
    statistics.factors[0]!.groups.degraded.rows.reduce((sum, row) => sum + row.count, 0),
    2,
  );
  assert.equal(statistics.usagePattern.levels[23], 11.5);
  summary.usagePattern.total = 0;
  summary.usagePattern.sums.fill(0);
  assert.deepEqual(calculateSurveyStatistics(summary).usagePattern.levels, Array(24).fill(0));
});
