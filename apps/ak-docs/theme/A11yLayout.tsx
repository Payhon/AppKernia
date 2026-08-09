import { useEffect } from 'react';
import { Layout as OriginalLayout, type LayoutProps } from '@rspress/core/theme-original';

const SCROLLABLE_REGION_SELECTOR = '.rp-table-scroll-container, .ak-diagram';

/** Keep keyboard users able to scroll responsive tables and diagram canvases. */
export function Layout(props: LayoutProps) {
  useEffect(() => {
    let frame = 0;
    const enhanceScrollableRegions = () => {
      for (const element of document.querySelectorAll(SCROLLABLE_REGION_SELECTOR)) {
        if (element instanceof HTMLElement && element.tabIndex !== 0) {
          element.tabIndex = 0;
        }
      }
    };
    const scheduleEnhancement = () => {
      cancelAnimationFrame(frame);
      frame = requestAnimationFrame(enhanceScrollableRegions);
    };

    enhanceScrollableRegions();
    const observer = new MutationObserver(scheduleEnhancement);
    observer.observe(document.body, { childList: true, subtree: true });

    return () => {
      cancelAnimationFrame(frame);
      observer.disconnect();
    };
  }, []);

  return <OriginalLayout {...props} />;
}
