// Preview is an enhancement for the configured Admin parent, never an auth mode.
const channel = "ak.public-web.preview.v1";

export function documentApp(url) {
  const match = /^\/h5\/apps\/([0-9a-f-]{36})\/(?:download|(?:articles|pages)\/[^/]+)\/?$/i.exec(url.pathname);
  if (match) return match[1].toLowerCase();
  if (/^\/s\/[^/]+$/.test(url.pathname) && /^[0-9a-f-]{36}$/i.test(url.searchParams.get("app_id") || "")) return url.searchParams.get("app_id").toLowerCase();
  return null;
}

export function keepInPreview(target, current) {
  return target.origin === current.origin && documentApp(current) !== null && documentApp(target) === documentApp(current) && !target.searchParams.has("format");
}

export function installPreview(win, doc) {
  const parentOrigin = doc.body.dataset.previewOrigin;
  if (!parentOrigin || win.parent === win) return () => {};
  let loadId = null;
  const send = (type) => win.parent.postMessage({ channel, type, loadId }, parentOrigin);
  const receive = (event) => {
    const data = event.data;
    if (event.origin !== parentOrigin || event.source !== win.parent || !data || data.channel !== channel || data.type !== "init" || typeof data.loadId !== "string" || !/^[a-zA-Z0-9-]{1,80}$/.test(data.loadId)) return;
    loadId = data.loadId;
    send(doc.body.dataset.previewKind === "error" ? "unavailable" : "ready");
  };
  const click = (event) => {
    if (!loadId) return;
    const anchor = event.target.closest?.("a[href]");
    if (!anchor) return;
    const target = new URL(anchor.href, win.location.href);
    if (keepInPreview(target, new URL(win.location.href))) {
      anchor.removeAttribute("target");
    } else {
      // Native anchor activation preserves user activation and popup policy.
      anchor.target = "_blank";
      anchor.rel = "noopener noreferrer";
    }
  };
  const keydown = (event) => {
    if (loadId && event.key === "Escape" && !event.defaultPrevented) {
      event.preventDefault();
      send("close");
    }
  };
  win.addEventListener("message", receive);
  doc.addEventListener("click", click, true);
  doc.addEventListener("auxclick", click, true);
  doc.addEventListener("keydown", keydown);
  return () => {
    win.removeEventListener("message", receive);
    doc.removeEventListener("click", click, true);
    doc.removeEventListener("auxclick", click, true);
    doc.removeEventListener("keydown", keydown);
  };
}

if (typeof window !== "undefined") installPreview(window, document);
