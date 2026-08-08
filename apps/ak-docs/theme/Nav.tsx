import { useNav, useSite } from '@rspress/core/runtime';
import '@rspress/core/dist/theme/components/Nav/index.css';
import {
  NavHamburger,
  NavTitle,
  Search,
  SwitchAppearance,
  type NavProps,
} from '@rspress/core/theme-original';
import {
  NavLangs,
  NavMenu,
  NavMenuDivider,
  NavVersions,
} from '@rspress/core/dist/theme/components/Nav/NavMenu.js';

/** Preserve the stock navbar while giving locale/version list items a list parent. */
export function Nav({
  beforeNavTitle,
  afterNavTitle,
  beforeNavMenu,
  afterNavMenu,
  navTitle,
}: NavProps) {
  const navList = useNav();
  const { site } = useSite();
  const hasAppearanceSwitch = site.themeConfig.darkMode !== false;

  return (
    <header className="rp-nav">
      <div className="rp-nav__left">
        {beforeNavTitle}
        {navTitle ?? <NavTitle />}
        <NavMenu menuItems={navList} position="left" />
        {afterNavTitle}
      </div>
      <div className="rp-nav__right">
        {beforeNavMenu}
        <Search />
        <NavMenu menuItems={navList} position="right" />
        <div className="rp-nav__others">
          <NavMenuDivider />
          <ul aria-label="Document language and version" className="ak-nav-utility-list">
            <NavLangs />
            <NavVersions />
          </ul>
          {hasAppearanceSwitch ? <SwitchAppearance /> : null}
        </div>
        <NavHamburger />
        {afterNavMenu}
      </div>
    </header>
  );
}
