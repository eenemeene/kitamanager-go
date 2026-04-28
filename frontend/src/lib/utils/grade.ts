// Natural-order comparison for pay-plan grade strings like "S8a".
//
// Mirrors CompareGrade in internal/models/payplan.go. The API returns entries
// in natural order already, but components that de-duplicate across periods
// (the salary chart, for example) may still need to re-sort the merged set.

export function compareGrade(a: string, b: string): number {
  const [pa, na, sa] = splitGrade(a);
  const [pb, nb, sb] = splitGrade(b);
  if (pa !== pb) return pa < pb ? -1 : 1;
  if (na !== nb) return na - nb;
  if (sa !== sb) return sa < sb ? -1 : 1;
  return 0;
}

function splitGrade(g: string): [string, number, string] {
  const m = g.match(/(\d+)/);
  if (!m || m.index === undefined) {
    return [g, 0, ''];
  }
  return [g.slice(0, m.index), parseInt(m[1], 10), g.slice(m.index + m[1].length)];
}
