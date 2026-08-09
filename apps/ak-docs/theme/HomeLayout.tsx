import { Content } from '@rspress/core/runtime';
import { HomeBackground, HomeFooter, type HomeLayoutProps } from '@rspress/core/theme-original';
import { HomeHero } from './HomeHero';

/**
 * Rspress's stock home layout renders frontmatter but omits the MDX body.
 * Keep the theme extension points while making the authored homepage content
 * part of the page and of the generated Markdown output.
 */
export function HomeLayout({
  beforeHero,
  afterHero,
  beforeFeatures,
  afterFeatures,
  beforeHeroActions,
  afterHeroActions,
}: HomeLayoutProps) {
  if (import.meta.env.SSG_MD) return <Content />;

  return (
    <>
      <HomeBackground />
      <main className="ak-home-main">
        {beforeHero}
        <HomeHero afterHeroActions={afterHeroActions} beforeHeroActions={beforeHeroActions} />
        {afterHero}
        {beforeFeatures}
        <div className="ak-home-content">
          <Content />
        </div>
        {afterFeatures}
      </main>
      <HomeFooter />
    </>
  );
}
