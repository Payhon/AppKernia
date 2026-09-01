const appDocument = /^\/h5\/apps\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\/(?:download|(?:articles|pages)\/[^/]+)\/?$/i;
const uuid = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

// Accept only public document links returned by our API, never arbitrary embeds.
export function publicPageURL(raw: string | undefined, locale: string): string | null {
  if (!raw) return null;
  try {
    const url = new URL(raw);
    const local = ["localhost", "127.0.0.1", "[::1]"].includes(url.hostname);
    if (url.protocol !== "https:" && !(url.protocol === "http:" && local)) return null;
    if (url.username || url.password) return null;
    const legacy = /^\/s\/[^/]+$/.test(url.pathname) && uuid.test(url.searchParams.get("app_id") ?? "");
    if (!appDocument.test(url.pathname) && !legacy) return null;
    if (url.searchParams.has("format")) return null;
    url.searchParams.set("lang", locale === "en-US" ? "en-US" : "zh-CN");
    return url.toString();
  } catch { return null; }
}

export const previewChannel = "ak.public-web.preview.v1";
export type PreviewMessageType = "ready" | "unavailable" | "close";

export function previewMessage(event: MessageEvent<unknown>, frame: Window | null, origin: string, loadId: string): PreviewMessageType | null {
  if (!frame || event.source !== frame || event.origin !== origin || !loadId) return null;
  const data = event.data;
  if (!data || typeof data !== "object" || !("channel" in data) || data.channel !== previewChannel || !("loadId" in data) || data.loadId !== loadId || !("type" in data)) return null;
  return data.type === "ready" || data.type === "unavailable" || data.type === "close" ? data.type : null;
}
