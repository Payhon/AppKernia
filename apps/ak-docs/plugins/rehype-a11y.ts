interface HastNode {
  type?: string;
  tagName?: string;
  properties?: Record<string, unknown>;
  children?: HastNode[];
}

function walk(node: HastNode) {
  const properties = node.properties;
  const ariaHidden = properties?.ariaHidden;
  if (
    node.type === 'element' &&
    node.tagName === 'a' &&
    properties &&
    (ariaHidden === true || ariaHidden === 'true')
  ) {
    properties.tabIndex = -1;
  }

  node.children?.forEach(walk);
}

export function rehypeA11y() {
  return walk;
}
