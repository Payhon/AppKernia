const allowed = new Map<string, Set<string>>([
  ["p", new Set()],
  ["br", new Set()],
  ["strong", new Set()],
  ["em", new Set()],
  ["u", new Set()],
  ["s", new Set()],
  ["blockquote", new Set()],
  ["code", new Set()],
  ["pre", new Set()],
  ["ul", new Set()],
  ["ol", new Set()],
  ["li", new Set()],
  ["h1", new Set()],
  ["h2", new Set()],
  ["h3", new Set()],
  ["h4", new Set()],
  ["h5", new Set()],
  ["h6", new Set()],
  ["table", new Set()],
  ["thead", new Set()],
  ["tbody", new Set()],
  ["tr", new Set()],
  ["th", new Set()],
  ["td", new Set()],
  ["hr", new Set()],
  ["a", new Set(["href", "title"])],
]);
const drop = new Set([
  "script",
  "style",
  "iframe",
  "object",
  "embed",
  "form",
  "input",
  "button",
  "svg",
  "math",
]);

function safeUrl(value: string) {
  try {
    const parsed = new URL(value, "https://appkernia.invalid");
    return ["http:", "https:", "mailto:"].includes(parsed.protocol);
  } catch {
    return false;
  }
}

export function sanitizeNotificationHtml(raw: string): string {
  const document = new DOMParser().parseFromString(raw, "text/html");
  const clean = (node: Node): Node[] => {
    if (node.nodeType === Node.COMMENT_NODE) return [];
    if (node.nodeType === Node.TEXT_NODE)
      return [document.createTextNode(node.textContent ?? "")];
    if (!(node instanceof Element)) return [];
    const name = node.tagName.toLowerCase();
    if (drop.has(name)) return [];
    const children = [...node.childNodes].flatMap(clean);
    const attributes = allowed.get(name);
    if (!attributes) return children;
    const next = document.createElement(name);
    for (const attribute of [...node.attributes]) {
      const key = attribute.name.toLowerCase();
      const value = attribute.value.trim();
      if (!attributes.has(key) || (key === "href" && !safeUrl(value))) continue;
      next.setAttribute(key, value);
    }
    next.append(...children);
    return [next];
  };
  const output = document.createElement("div");
  output.append(...[...document.body.childNodes].flatMap(clean));
  return output.innerHTML.trim();
}
