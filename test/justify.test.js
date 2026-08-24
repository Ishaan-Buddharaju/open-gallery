import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { justify } from '../web/justify.js';

// ── helpers ──────────────────────────────────────────────────────────────

function make(aspects) {
  return aspects.map((a, i) => ({ id: `img-${i}`, aspect: a }));
}

function allRects(result) {
  return result.rows.flatMap(r => r.items);
}

/** Assert every hard invariant from the spec. */
function check(result, items, opts) {
  const rects = allRects(result);
  const { width, height } = opts;

  // 1. Every item appears exactly once.
  const ids = rects.map(r => r.id).sort();
  const exp = items.map(it => it.id).sort();
  assert.deepStrictEqual(ids, exp, 'items mismatch');

  // 2. No two rects overlap (≤ 0.5 px tolerance).
  for (let i = 0; i < rects.length; i++) {
    for (let j = i + 1; j < rects.length; j++) {
      const a = rects[i], b = rects[j];
      const ox = Math.min(a.x + a.w, b.x + b.w) - Math.max(a.x, b.x);
      const oy = Math.min(a.y + a.h, b.y + b.h) - Math.max(a.y, b.y);
      assert.ok(
        !(ox > 0.5 && oy > 0.5),
        `overlap: ${a.id} ∩ ${b.id} = ${ox.toFixed(1)}×${oy.toFixed(1)}`
      );
    }
  }

  // 3. Every row spans the full width to within 1 px.
  for (const row of result.rows) {
    if (!row.items.length) continue;
    const first = row.items[0];
    const last = row.items[row.items.length - 1];
    assert.ok(
      Math.abs(first.x) < 1,
      `left edge off by ${first.x.toFixed(2)} px`
    );
    assert.ok(
      Math.abs(last.x + last.w - width) < 1,
      `right edge off by ${Math.abs(last.x + last.w - width).toFixed(2)} px`
    );
  }

  // 4. Union covers canvas to within 1 px on all four edges.
  const minX = Math.min(...rects.map(r => r.x));
  const maxX = Math.max(...rects.map(r => r.x + r.w));
  const minY = Math.min(...rects.map(r => r.y));
  const maxY = Math.max(...rects.map(r => r.y + r.h));
  assert.ok(minX < 1, `left gap: ${minX.toFixed(2)} px`);
  assert.ok(Math.abs(maxX - width) < 1, `right gap: ${(width - maxX).toFixed(2)} px`);
  assert.ok(minY < 1, `top gap: ${minY.toFixed(2)} px`);
  assert.ok(
    Math.abs(maxY - height) < 1,
    `bottom gap: ${Math.abs(maxY - height).toFixed(2)} px`
  );

  // 5. No rect has aspect ratio outside [0.35, 3.2].
  for (const r of rects) {
    const ar = r.w / r.h;
    assert.ok(
      ar >= 0.34 && ar <= 3.21,
      `${r.id} aspect ${ar.toFixed(4)} outside [0.35, 3.2]`
    );
  }

  // 6. Deterministic: same input → same output, every time.
  const again = justify(items, opts);
  assert.deepStrictEqual(result, again, 'non-deterministic');
}

// ── canvases ─────────────────────────────────────────────────────────────

const C16_9 = { width: 1920, height: 1080, gutter: 8 };
const C16_10 = { width: 1920, height: 1200, gutter: 8 };

// ── count sweep n = 1..12 ────────────────────────────────────────────────

describe('n = 1..12, uniform aspects', () => {
  for (let n = 1; n <= 12; n++) {
    it(`${n} landscape (1.333) on 16:9`, () => {
      const its = make(Array(n).fill(1.333));
      check(justify(its, C16_9), its, C16_9);
    });
    it(`${n} portrait (0.75) on 16:9`, () => {
      const its = make(Array(n).fill(0.75));
      check(justify(its, C16_9), its, C16_9);
    });
    it(`${n} square (1.0) on 16:9`, () => {
      const its = make(Array(n).fill(1.0));
      check(justify(its, C16_9), its, C16_9);
    });
  }
});

// ── 16:10 count sweep ────────────────────────────────────────────────────

describe('n = 1..12, 16:10', () => {
  for (let n = 1; n <= 12; n++) {
    it(`${n} landscape on 16:10`, () => {
      const its = make(Array(n).fill(1.333));
      check(justify(its, C16_10), its, C16_10);
    });
    it(`${n} portrait on 16:10`, () => {
      const its = make(Array(n).fill(0.75));
      check(justify(its, C16_10), its, C16_10);
    });
    it(`${n} square on 16:10`, () => {
      const its = make(Array(n).fill(1.0));
      check(justify(its, C16_10), its, C16_10);
    });
  }
});

// ── realistic mix ────────────────────────────────────────────────────────

describe('mixed aspects', () => {
  const MIX = [0.75, 1.333, 0.75, 1.0, 1.777, 0.75, 1.333, 0.5625];

  it('realistic mix on 16:9', () => {
    const its = make(MIX);
    check(justify(its, C16_9), its, C16_9);
  });

  it('realistic mix on 16:10', () => {
    const its = make(MIX);
    check(justify(its, C16_10), its, C16_10);
  });
});

// ── pathological: panorama among portraits ───────────────────────────────

describe('pathological', () => {
  it('3.5 panorama among 7 portraits, 16:9', () => {
    const its = make([0.75, 0.75, 0.75, 3.5, 0.75, 0.75, 0.75, 0.75]);
    check(justify(its, C16_9), its, C16_9);
  });

  it('3.5 panorama among 7 portraits, 16:10', () => {
    const its = make([0.75, 0.75, 0.75, 3.5, 0.75, 0.75, 0.75, 0.75]);
    check(justify(its, C16_10), its, C16_10);
  });

  it('hostile trio: panorama + screenshot + square', () => {
    const its = make([3.5, 0.5625, 1.0]);
    check(justify(its, C16_9), its, C16_9);
  });

  it('hostile trio on 16:10', () => {
    const its = make([3.5, 0.5625, 1.0]);
    check(justify(its, C16_10), its, C16_10);
  });
});

// ── empty input ──────────────────────────────────────────────────────────

describe('edge cases', () => {
  it('empty items', () => {
    const result = justify([], C16_9);
    assert.deepStrictEqual(result, { rows: [], totalHeight: 0 });
  });
});
