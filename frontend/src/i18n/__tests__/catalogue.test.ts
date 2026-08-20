/**
 * @jest-environment node
 *
 * Drift guards for the message catalogues, mirroring what
 * `internal/i18n/coverage_test.go` does for the Go side.
 *
 * The Go catalogue has two tests behind it: one proves every user-facing
 * message has a registry entry, the other proves every entry has an English
 * source and a German translation. The frontend had no equivalent, and the gap
 * showed — three `t()` calls shipped referencing keys that do not resolve, and
 * next-intl's fallback is to render the key path, so the German form-error
 * summary listed a field literally named `common.gender`. Nothing failed, in
 * dev or in CI.
 *
 * These two suites close that: the first proves the catalogues agree with each
 * other, the second proves they agree with the code. They run in the node
 * environment because they read the source tree from disk.
 */

import fs from 'node:fs';
import path from 'node:path';
import ts from 'typescript';

import en from '../messages/en.json';
import de from '../messages/de.json';

type Catalogue = { [key: string]: string | Catalogue };

const SRC = path.join(__dirname, '..', '..');

/** Every leaf (string-valued) path in a catalogue, dot-joined. */
function leafPaths(node: Catalogue, prefix = ''): Map<string, string> {
  const out = new Map<string, string>();
  for (const [key, value] of Object.entries(node)) {
    const dotted = prefix ? `${prefix}.${key}` : key;
    if (typeof value === 'string') {
      out.set(dotted, value);
    } else {
      for (const [k, v] of leafPaths(value, dotted)) out.set(k, v);
    }
  }
  return out;
}

/** Every path in a catalogue, leaves and intermediate objects alike. */
function allPaths(node: Catalogue, prefix = ''): Set<string> {
  const out = new Set<string>();
  for (const [key, value] of Object.entries(node)) {
    const dotted = prefix ? `${prefix}.${key}` : key;
    out.add(dotted);
    if (typeof value !== 'string') {
      for (const p of allPaths(value, dotted)) out.add(p);
    }
  }
  return out;
}

/**
 * The variable names an ICU message expects to be given.
 *
 * This has to understand nesting rather than scan for `{word`, because a plural
 * branch body is delimited by braces too: in
 * `{count, plural, one {month} other {months}}` the only variable is `count` —
 * `month` and `months` are literal text. Reading them as variables makes every
 * translated plural look like a mismatch, since the German branch bodies say
 * `Monat` and `Monate`.
 *
 * Names are returned as a set. `{count}` alongside `{count, plural, …}` is one
 * variable mentioned twice, and a translation may mention it a different number
 * of times than the English does — word order is the translator's business.
 * What must not differ is *which* variables the string expects, because a
 * dropped one renders a sentence with a hole in it.
 */
function placeholders(message: string): Set<string> {
  const names = new Set<string>();
  let i = 0;

  const skipSpace = (): void => {
    while (i < message.length && /\s/.test(message[i])) i++;
  };

  const readWord = (): string => {
    const start = i;
    while (i < message.length && /\w/.test(message[i])) i++;
    return message.slice(start, i);
  };

  /** Advance past the argument we are inside, brace-counting from depth 1. */
  const skipToClose = (): void => {
    let depth = 1;
    while (i < message.length && depth > 0) {
      if (message[i] === '{') depth++;
      else if (message[i] === '}') depth--;
      i++;
    }
  };

  const readArgument = (): void => {
    i++; // past '{'
    skipSpace();
    const name = readWord();
    if (name) names.add(name);
    skipSpace();
    if (message[i] === '}') {
      i++;
      return;
    }
    if (message[i] !== ',') {
      skipToClose(); // malformed; step over it rather than looping
      return;
    }
    i++;
    skipSpace();
    const type = readWord();
    if (type !== 'plural' && type !== 'select' && type !== 'selectordinal') {
      skipToClose(); // date/number/time: a style, not more messages
      return;
    }
    // Selectors, each followed by a braced body that is itself a message and
    // may contain further arguments.
    while (i < message.length && message[i] !== '}') {
      if (message[i] === '{') {
        i++;
        readMessage();
        i++; // past the body's '}'
      } else {
        i++;
      }
    }
    i++; // past the argument's '}'
  };

  function readMessage(): void {
    while (i < message.length) {
      const ch = message[i];
      if (ch === '}') return; // end of the enclosing branch body
      // ICU quoting: an apostrophe only escapes when it precedes a brace.
      // Treating every apostrophe as a quote would swallow the `{count}` in
      // "you won't see {count} again".
      if (ch === "'" && (message[i + 1] === '{' || message[i + 1] === '}')) {
        i += 2;
        continue;
      }
      if (ch === '{') {
        readArgument();
        continue;
      }
      i++;
    }
  }

  readMessage();
  return names;
}

const enLeaves = leafPaths(en as Catalogue);
const deLeaves = leafPaths(de as Catalogue);

describe('message catalogue parity', () => {
  it('has the same set of keys in English and German', () => {
    const missingFromDe = [...enLeaves.keys()].filter((k) => !deLeaves.has(k)).sort();
    const missingFromEn = [...deLeaves.keys()].filter((k) => !enLeaves.has(k)).sort();

    expect({ missingFromDe, missingFromEn }).toEqual({ missingFromDe: [], missingFromEn: [] });
  });

  it('has the same nesting shape in both catalogues', () => {
    // A key that is an object in one catalogue and a string in the other breaks
    // more quietly than a missing key: `t()` on the object side renders the key
    // path instead of a message, and only in that one language.
    const enShape = allPaths(en as Catalogue);
    const deShape = allPaths(de as Catalogue);
    const onlyEn = [...enShape].filter((p) => !deShape.has(p)).sort();
    const onlyDe = [...deShape].filter((p) => !enShape.has(p)).sort();

    expect({ onlyEn, onlyDe }).toEqual({ onlyEn: [], onlyDe: [] });
  });

  it('expects the same ICU variables in both languages', () => {
    const mismatched: string[] = [];
    for (const [key, enMessage] of enLeaves) {
      const deMessage = deLeaves.get(key);
      if (deMessage === undefined) continue; // reported by the parity test above
      const wanted = [...placeholders(enMessage)].sort().join(',');
      const got = [...placeholders(deMessage)].sort().join(',');
      if (wanted !== got) mismatched.push(`${key}: en={${wanted}} de={${got}}`);
    }

    expect(mismatched).toEqual([]);
  });

  it('has no blank German translations', () => {
    const blank = [...deLeaves.entries()].filter(([, v]) => v.trim() === '').map(([k]) => k);

    expect(blank).toEqual([]);
  });
});

/** One translation lookup found in the source, with the namespace it was bound to. */
interface TranslationCall {
  key: string;
  scope: string | null;
  file: string;
  line: number;
}

/**
 * The literal keys a call argument can resolve to, or null when it is computed.
 *
 * A conditional between two literals — `t(x ? 'a.b' : 'a.c')` — is as checkable
 * as either literal alone, so both branches are returned.
 */
function literalKeys(arg: ts.Expression): string[] | null {
  if (ts.isStringLiteralLike(arg)) return [arg.text];
  if (ts.isConditionalExpression(arg)) {
    const whenTrue = literalKeys(arg.whenTrue);
    const whenFalse = literalKeys(arg.whenFalse);
    if (whenTrue && whenFalse) return [...whenTrue, ...whenFalse];
  }
  return null;
}

/**
 * Collect every translation lookup in a file.
 *
 * Parsed rather than pattern-matched, for the same reason the Go coverage test
 * walks an AST: the translator is not always called `t` — there are bindings
 * named `tCommon`, `tLabels`, `tParent`, `tGender` and more — and each carries
 * its own namespace. A regex that ignored which binding a call belongs to would
 * happily accept `tCommon('children.title')`, a lookup that resolves to
 * nothing.
 */
function collectCalls(file: string): { calls: TranslationCall[]; dynamic: number } {
  const source = ts.createSourceFile(
    file,
    fs.readFileSync(file, 'utf8'),
    ts.ScriptTarget.Latest,
    true,
    ts.ScriptKind.TSX
  );

  // Translator bindings in this file: variable name -> namespace (null = root).
  const scopes = new Map<string, string | null>();
  const calls: TranslationCall[] = [];
  let dynamic = 0;

  /** `useTranslations(…)`, `getTranslations(…)`, and awaited forms of either. */
  const translatorFactory = (expr: ts.Expression): ts.CallExpression | null => {
    const inner = ts.isAwaitExpression(expr) ? expr.expression : expr;
    if (!ts.isCallExpression(inner)) return null;
    const callee = inner.expression;
    const name = ts.isIdentifier(callee)
      ? callee.text
      : ts.isPropertyAccessExpression(callee)
        ? callee.name.text
        : '';
    return name === 'useTranslations' || name === 'getTranslations' ? inner : null;
  };

  // Two passes: bindings first, because a translator may be declared below a
  // closure that uses it.
  const findBindings = (node: ts.Node): void => {
    if (ts.isVariableDeclaration(node) && ts.isIdentifier(node.name) && node.initializer) {
      const factory = translatorFactory(node.initializer);
      if (factory) {
        const arg = factory.arguments[0];
        scopes.set(node.name.text, arg && ts.isStringLiteralLike(arg) ? arg.text : null);
      }
    }
    ts.forEachChild(node, findBindings);
  };

  const findCalls = (node: ts.Node): void => {
    if (ts.isCallExpression(node)) {
      const callee = node.expression;
      let binding = '';
      let method = '';
      if (ts.isIdentifier(callee)) {
        binding = callee.text;
      } else if (ts.isPropertyAccessExpression(callee) && ts.isIdentifier(callee.expression)) {
        binding = callee.expression.text; // t.rich('key', …), t.has('key')
        method = callee.name.text;
      }
      const arg = node.arguments[0];
      if (scopes.has(binding) && arg) {
        const keys = literalKeys(arg);
        if (keys) {
          for (const key of keys) {
            calls.push({
              key,
              scope: scopes.get(binding) ?? null,
              file,
              line: source.getLineAndCharacterOfPosition(node.getStart()).line + 1,
            });
          }
        } else if (method !== 'has') {
          // `t.has()` asks whether a key exists; a computed argument there is
          // the guarded pattern, not an unchecked lookup.
          dynamic++;
        }
      }
    }
    ts.forEachChild(node, findCalls);
  };

  findBindings(source);
  findCalls(source);
  return { calls, dynamic };
}

function sourceFiles(dir: string): string[] {
  const out: string[] = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      if (entry.name === '__tests__' || entry.name === 'node_modules') continue;
      out.push(...sourceFiles(full));
    } else if (/\.tsx?$/.test(entry.name) && !/\.test\.tsx?$/.test(entry.name)) {
      out.push(full);
    }
  }
  return out;
}

describe('translation keys used in components', () => {
  const files = sourceFiles(SRC);
  const calls: TranslationCall[] = [];
  let dynamicCalls = 0;
  for (const file of files) {
    const result = collectCalls(file);
    calls.push(...result.calls);
    dynamicCalls += result.dynamic;
  }

  const enShape = allPaths(en as Catalogue);

  /** The catalogue path a lookup resolves to, or null when it resolves to nothing. */
  const resolve = (call: TranslationCall): string | null => {
    const scoped = call.scope ? `${call.scope}.${call.key}` : call.key;
    if (enLeaves.has(scoped)) return scoped;
    // Some call sites pass a fully-qualified key to a scoped translator. It
    // works, so report genuine breakage rather than style.
    if (enLeaves.has(call.key)) return call.key;
    return null;
  };

  it('finds the translation call sites', () => {
    // Guard against a moved layout making every assertion below vacuous. The
    // floors sit well under the real counts (255 files, 1499 lookups at the
    // time of writing) — a tripwire for a broken walk, not a census.
    expect(files.length).toBeGreaterThan(150);
    expect(calls.length).toBeGreaterThan(1200);
  });

  it('resolves every literal key to a message', () => {
    const unresolved = calls
      .filter((c) => resolve(c) === null)
      .map((c) => {
        const shown = c.scope ? `${c.scope}.${c.key}` : c.key;
        const why = enShape.has(shown) ? 'is a namespace, not a message' : 'does not exist';
        return `${path.relative(SRC, c.file)}:${c.line}  ${shown} — ${why}`;
      })
      .sort();

    // next-intl renders the key path both when a key is missing and when it
    // names a namespace rather than a message, so either way the failure
    // reaches the screen as developer text with nothing raised.
    expect([...new Set(unresolved)]).toEqual([]);
  });

  it('resolves every literal key in German too', () => {
    // Separate from the parity suite above: parity proves the catalogues agree
    // with each other, this proves the key the code asks for is one a German
    // reader actually gets a message for.
    const missing = calls
      .map(resolve)
      .filter((p): p is string => p !== null)
      .filter((p) => !deLeaves.has(p))
      .sort();

    expect([...new Set(missing)]).toEqual([]);
  });

  it('keeps computed keys a small enough minority to review by hand', () => {
    // `t(`states.${org.state}`)` and `t(item.name)` cannot be checked
    // statically, so they are the blind spot in the two tests above. That is
    // acceptable while they stay rare — 76 of 1575 lookups, 4.8%, at the time
    // of writing. The bound is a ratchet on that blind spot: drifting toward
    // computed keys should be a decision someone makes, not coverage quietly
    // draining away.
    const ratio = dynamicCalls / (dynamicCalls + calls.length);

    expect(ratio).toBeLessThan(0.1);
  });
});
