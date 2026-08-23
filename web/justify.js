/**
 * Justified-rows layout algorithm.
 *
 * Derives each frame's shape from the photo inside it (Flickr / Google Photos
 * algorithm). Items are partitioned into contiguous rows via linear-partition
 * DP that minimises row-height variance, then scaled to fill the canvas.
 * When a uniform scale would push any displayed aspect outside [0.35, 3.2],
 * falls back to per-row scaling with iterative clamping.
 *
 * @param {Array<{id: *, aspect: number}>} items
 * @param {{width: number, height: number, gutter: number}} opts
 * @returns {{
 *   rows: Array<{height: number, items: Array<{id: *, x: number, y: number, w: number, h: number}>}>,
 *   totalHeight: number
 * }}
 */
export function justify(items, opts) {
  const { width, height, gutter } = opts;
  const n = items.length;
  if (n === 0) return { rows: [], totalHeight: 0 };

  const a = items.map(it => Math.max(0.35, Math.min(3.2, it.aspect)));

  function rh(l, r) {
    let s = 0;
    for (let i = l; i < r; i++) s += a[i];
    return (width - (r - l - 1) * gutter) / s;
  }

  function partition(R) {
    if (R === 1) return [0, n];
    if (R >= n) return Array.from({ length: n + 1 }, (_, i) => i);

    let best = null;
    let bestV = Infinity;
    const d = [];

    (function go(from, rem) {
      if (rem === 0) {
        const sp = [0, ...d, n];
        const hs = new Array(R);
        for (let i = 0; i < R; i++) hs[i] = rh(sp[i], sp[i + 1]);
        let m = 0;
        for (let i = 0; i < R; i++) m += hs[i];
        m /= R;
        let v = 0;
        for (let i = 0; i < R; i++) v += (hs[i] - m) ** 2;
        if (v < bestV) { bestV = v; best = sp.slice(); }
        return;
      }
      for (let i = from; i <= n - rem; i++) {
        d.push(i);
        go(i + 1, rem - 1);
        d.pop();
      }
    })(1, R - 1);

    return best;
  }

  // Compute per-row scale factors that keep all displayed aspects in
  // [0.35, 3.2] while summing to the target height. Falls back to uniform
  // when all rows are compatible.
  function solveScales(R, sp) {
    const availH = height - (R - 1) * gutter;
    const natHs = new Array(R);
    const bounds = new Array(R);
    let sumNat = 0;

    for (let ri = 0; ri < R; ri++) {
      const l = sp[ri], r = sp[ri + 1];
      const h = rh(l, r);
      natHs[ri] = h;
      sumNat += h;
      let lo = 0, hi = Infinity;
      for (let i = l; i < r; i++) {
        lo = Math.max(lo, a[i] / 3.2);
        hi = Math.min(hi, a[i] / 0.35);
      }
      bounds[ri] = { lo, hi };
    }

    const scales = new Array(R).fill(availH / sumNat);
    const fixed = new Set();

    for (let iter = 0; iter <= R; iter++) {
      let changed = false;
      let fixedH = 0;
      let freeNat = 0;

      for (let i = 0; i < R; i++) {
        if (fixed.has(i)) {
          fixedH += natHs[i] * scales[i];
          continue;
        }
        if (scales[i] < bounds[i].lo) {
          scales[i] = bounds[i].lo;
          fixed.add(i);
          fixedH += natHs[i] * scales[i];
          changed = true;
        } else if (scales[i] > bounds[i].hi) {
          scales[i] = bounds[i].hi;
          fixed.add(i);
          fixedH += natHs[i] * scales[i];
          changed = true;
        } else {
          freeNat += natHs[i];
        }
      }

      if (!changed || freeNat === 0) break;

      const remH = availH - fixedH;
      const sc = remH / freeNat;
      for (let i = 0; i < R; i++) {
        if (!fixed.has(i)) scales[i] = sc;
      }
    }

    let worst = 0;
    for (let ri = 0; ri < R; ri++) {
      const l = sp[ri], r = sp[ri + 1];
      for (let i = l; i < r; i++) {
        const d = a[i] / scales[ri];
        if (d < 0.35) worst = Math.max(worst, 0.35 - d);
        else if (d > 3.2) worst = Math.max(worst, d - 3.2);
      }
    }

    let actualH = (R - 1) * gutter;
    for (let i = 0; i < R; i++) actualH += natHs[i] * scales[i];

    return { scales, natHs, worst, actualH };
  }

  const maxR = Math.min(n, 5);
  const cands = [];

  for (let R = 1; R <= maxR; R++) {
    const sp = partition(R);
    const sol = solveScales(R, sp);
    cands.push({ R, sp, ...sol });
  }

  const TOL = 0.01;
  cands.sort((a, b) => {
    const aOk = a.worst < TOL;
    const bOk = b.worst < TOL;
    if (aOk !== bOk) return aOk ? -1 : 1;
    if (aOk) return Math.abs(a.actualH - height) - Math.abs(b.actualH - height);
    return a.worst - b.worst;
  });

  const { R, sp, scales } = cands[0];

  const rows = [];
  let y = 0;

  for (let ri = 0; ri < R; ri++) {
    const l = sp[ri];
    const r = sp[ri + 1];
    const h = rh(l, r) * scales[ri];
    const k = r - l;
    const avail = width - (k - 1) * gutter;
    let sa = 0;
    for (let i = l; i < r; i++) sa += a[i];

    const rowItems = [];
    let x = 0;
    let used = 0;

    for (let i = l; i < r; i++) {
      let w;
      if (i === r - 1) {
        w = avail - used;
      } else {
        w = avail * a[i] / sa;
        used += w;
      }
      rowItems.push({ id: items[i].id, x, y, w, h });
      x += w + (i < r - 1 ? gutter : 0);
    }

    rows.push({ height: h, items: rowItems });
    y += h + (ri < R - 1 ? gutter : 0);
  }

  return { rows, totalHeight: y };
}
