/**
 * copyToClipboard writes `text` to the user's clipboard and returns
 * whether the copy succeeded. Silent fail (never throws) so callers
 * can surface a single toast with either outcome without a try/catch.
 *
 * Falls back to a textarea + execCommand path for the rare browsers
 * (or contexts — e.g. insecure origin) where navigator.clipboard is
 * unavailable. If both fail the returned promise resolves to false.
 */
export async function copyToClipboard(text: string): Promise<boolean> {
  if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      // Fall through to the legacy path — common on http:// origins
      // where Clipboard API is blocked by the browser.
    }
  }
  if (typeof document === 'undefined') return false;
  try {
    const ta = document.createElement('textarea');
    ta.value = text;
    ta.setAttribute('readonly', '');
    ta.style.position = 'absolute';
    ta.style.left = '-9999px';
    document.body.appendChild(ta);
    ta.select();
    const ok = document.execCommand('copy');
    document.body.removeChild(ta);
    return ok;
  } catch {
    return false;
  }
}
