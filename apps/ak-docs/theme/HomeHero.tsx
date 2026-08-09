import { normalizeImagePath, useFrontmatter } from '@rspress/core/runtime';
import '@rspress/core/dist/theme/components/HomeHero/index.css';
import { Button, Link, renderHtmlOrText, type HomeHeroProps } from '@rspress/core/theme-original';

type HeroShowcaseItem = {
  src: string;
  alt: string;
};

export type HeroShowcase = {
  admin: HeroShowcaseItem;
  mobile: [HeroShowcaseItem, HeroShowcaseItem];
};

function normalizeResponsiveField(field?: string | string[]) {
  const value = (Array.isArray(field) ? field : [field]).filter(Boolean).join(', ');
  return value || undefined;
}

function readShowcase(value: unknown): HeroShowcase | undefined {
  if (!value || typeof value !== 'object') return undefined;
  const candidate = value as Partial<HeroShowcase>;
  const validItem = (item: HeroShowcaseItem | undefined) =>
    typeof item?.src === 'string' &&
    item.src.length > 0 &&
    typeof item.alt === 'string' &&
    item.alt.length > 0;
  if (
    !validItem(candidate.admin) ||
    !Array.isArray(candidate.mobile) ||
    candidate.mobile.length !== 2
  ) {
    return undefined;
  }
  if (!validItem(candidate.mobile[0]) || !validItem(candidate.mobile[1])) return undefined;
  return candidate as HeroShowcase;
}

function ProductShowcase({ showcase }: { showcase: HeroShowcase }) {
  return (
    <figure className="ak-hero-showcase">
      <div className="ak-hero-browser">
        <div aria-hidden="true" className="ak-hero-browser__bar">
          <span />
          <span />
          <span />
          <strong>appkernia / admin</strong>
        </div>
        <img alt={showcase.admin.alt} src={normalizeImagePath(showcase.admin.src)} />
      </div>
      {showcase.mobile.map((item, index) => (
        <div
          className={`ak-hero-phone ak-hero-phone--${index === 0 ? 'primary' : 'secondary'}`}
          key={item.src}
        >
          <span aria-hidden="true" className="ak-hero-phone__sensor" />
          <img alt={item.alt} src={normalizeImagePath(item.src)} />
        </div>
      ))}
    </figure>
  );
}

/**
 * Rspress renders the stock hero heading with div elements. Preserve its theme
 * contracts while exposing the product statement as one real H1 and supporting
 * a local, typed product-screenshot composition.
 */
export function HomeHero({ beforeHeroActions, afterHeroActions, image }: HomeHeroProps) {
  const { frontmatter } = useFrontmatter();
  const hero = frontmatter.hero;
  const showcase = readShowcase(frontmatter.heroShowcase);
  const hasImage = showcase !== undefined || hero?.image !== undefined || image !== undefined;
  const lines = hero?.text
    ? hero.text
        .toString()
        .split(/\n/g)
        .filter((line) => line !== '')
    : [];
  const imageSource =
    typeof hero?.image?.src === 'string'
      ? { light: hero.image.src, dark: hero.image.src }
      : (hero?.image?.src ?? { light: '', dark: '' });

  return (
    <div className={`rp-home-hero${hasImage ? '' : ' rp-home-hero--no-image'}`}>
      <div className="rp-home-hero__container">
        {hero?.badge ? (
          typeof hero.badge === 'string' ? (
            <div className="rp-home-hero__badge">{hero.badge}</div>
          ) : hero.badge.link ? (
            <Link className="rp-home-hero__badge" href={hero.badge.link}>
              {hero.badge.text}
            </Link>
          ) : (
            <div className="rp-home-hero__badge">{hero.badge.text}</div>
          )
        ) : null}

        <h1 className="rp-home-hero__content">
          {hero?.name ? (
            <span className="rp-home-hero__title">
              <span className="rp-home-hero__title-brand" {...renderHtmlOrText(hero.name)} />
            </span>
          ) : null}
          {lines.map((line) => (
            <span className="rp-home-hero__subtitle" key={line} {...renderHtmlOrText(line)} />
          ))}
        </h1>

        {hero?.tagline ? (
          <p className="rp-home-hero__tagline" {...renderHtmlOrText(hero.tagline)} />
        ) : null}

        {beforeHeroActions}
        <div className="rp-home-hero__actions">
          {hero?.actions?.map((action) => (
            <Button
              className="rp-home-hero__action"
              href={action.link}
              key={action.link}
              theme={action.theme}
              type="a"
              {...renderHtmlOrText(action.text)}
            />
          ))}
        </div>
        {afterHeroActions}
      </div>

      {showcase ? (
        <div className="rp-home-hero__image">
          <ProductShowcase showcase={showcase} />
        </div>
      ) : image ? (
        <div className="rp-home-hero__image">{image}</div>
      ) : hero?.image ? (
        <div className="rp-home-hero__image">
          <img
            alt={hero.image.alt}
            className="rp-home-hero__image-img rp-home-hero__image-img--light"
            height={375}
            sizes={normalizeResponsiveField(hero.image.sizes)}
            src={normalizeImagePath(imageSource.light)}
            srcSet={normalizeResponsiveField(hero.image.srcset)}
            width={375}
          />
          <img
            alt={hero.image.alt}
            className="rp-home-hero__image-img rp-home-hero__image-img--dark"
            height={375}
            sizes={normalizeResponsiveField(hero.image.sizes)}
            src={normalizeImagePath(imageSource.dark)}
            srcSet={normalizeResponsiveField(hero.image.srcset)}
            width={375}
          />
        </div>
      ) : null}
    </div>
  );
}
