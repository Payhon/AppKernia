import { normalizeImagePath, useFrontmatter } from '@rspress/core/runtime';
import '@rspress/core/dist/theme/components/HomeHero/index.css';
import { Button, Link, renderHtmlOrText, type HomeHeroProps } from '@rspress/core/theme-original';

function normalizeResponsiveField(field?: string | string[]) {
  const value = (Array.isArray(field) ? field : [field]).filter(Boolean).join(', ');
  return value || undefined;
}

/**
 * Rspress 2 renders its stock hero heading with div elements. Keep the stock
 * classes and behavior while exposing the product statement as one real H1.
 */
export function HomeHero({ beforeHeroActions, afterHeroActions, image }: HomeHeroProps) {
  const { frontmatter } = useFrontmatter();
  const hero = frontmatter.hero;
  const hasImage = hero?.image !== undefined || image !== undefined;
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

      {image ? (
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
