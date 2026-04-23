/**
 * downloadAsText triggers a browser download of plaintext content.
 * Creates a Blob, hooks it up to an invisible anchor, clicks it, and
 * revokes the URL. Used by the backup-codes dialog to save the
 * one-shot recovery codes as a .txt file the user can stash in a
 * password manager.
 */
export function downloadAsText(content: string, filename: string): void {
  if (typeof document === 'undefined' || typeof URL === 'undefined') return;
  const blob = new Blob([content], { type: 'text/plain;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}
