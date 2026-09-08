export interface BinaryAssociation {
  selected: { events: number; total: number };
  unselected: { events: number; total: number };
  phi: number | null;
}

/** Phi for a 2×2 table: a/b selected, c/d unselected; a/c have the outcome. */
export function binaryAssociation(a: number, b: number, c: number, d: number): BinaryAssociation {
  const denominator = Math.sqrt((a + b) * (c + d) * (a + c) * (b + d));
  return {
    selected: { events: a, total: a + b },
    unselected: { events: c, total: c + d },
    phi: denominator === 0 ? null : (a * d - b * c) / denominator,
  };
}
