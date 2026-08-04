import { adminRouteRegistry, type AdminComponentKey } from './static-route-registry';

interface ApiMenuNode { code: string; title: string; componentKey?: string; children?: ApiMenuNode[]; }
interface ResolvedMenuNode extends ApiMenuNode { to?: string; children?: ResolvedMenuNode[]; }

export function resolveMenus(nodes: ApiMenuNode[], onUnknownKey: (key: string, menuCode: string) => void): ResolvedMenuNode[] {
  return nodes.flatMap((node) => {
    let to: string | undefined;
    if (node.componentKey) {
      const registration = adminRouteRegistry[node.componentKey as AdminComponentKey];
      if (!registration) { onUnknownKey(node.componentKey, node.code); return []; }
      to = registration.to;
    }
    const children = resolveMenus(node.children ?? [], onUnknownKey);
    if (!to && children.length === 0) return [];
    const resolved: ResolvedMenuNode = { ...node, children };
    if (to) resolved.to = to;
    return [resolved];
  });
}
