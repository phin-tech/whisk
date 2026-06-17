export function externalAttachmentURL(attachment: { url?: string }) {
  return String(attachment.url ?? "").trim();
}

export async function openExternalURL(
  url: string,
  nativeOpen: (url: string) => Promise<void>,
  fallbackOpen: (url: string) => void = (target) => {
    window.open(target, "_blank", "noopener");
  },
) {
  const target = url.trim();
  if (!target) return;

  try {
    await nativeOpen(target);
  } catch {
    fallbackOpen(target);
  }
}
