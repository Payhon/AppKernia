import { normalizeImagePath } from '@rspress/core/runtime';
import { useId, useState, type KeyboardEvent } from 'react';
import antdesignLogo from 'simple-icons/icons/antdesign.svg';
import dockerLogo from 'simple-icons/icons/docker.svg';
import goLogo from 'simple-icons/icons/go.svg';
import openapiLogo from 'simple-icons/icons/openapiinitiative.svg';
import postgresqlLogo from 'simple-icons/icons/postgresql.svg';
import reactLogo from 'simple-icons/icons/react.svg';
import typescriptLogo from 'simple-icons/icons/typescript.svg';
import viteLogo from 'simple-icons/icons/vite.svg';

type HomeLocale = 'zh-CN' | 'en-US';

type ProductSlide = {
  src: string;
  alt: string;
  title: string;
  detail: string;
};

type SliderCopy = {
  label: string;
  previous: string;
  next: string;
  jumpTo: string;
};

type ProductSliderProps = {
  kind: 'admin' | 'mobile';
  slides: ProductSlide[];
  copy: SliderCopy;
};

const PRODUCT_COPY = {
  'zh-CN': {
    eyebrow: 'PRODUCT, END TO END',
    heading: '从运营后台到用户手机，围绕同一份业务继续生长',
    lead: '在 Admin 中管理应用、内容、版本与存储，在 Mobile 中承接登录、通知、阅读与账户体验。两端通过同一套 API 和业务规则自然衔接。',
    slider: {
      previous: '上一张',
      next: '下一张',
      jumpTo: '切换到',
    },
    adminLabel: 'Web 管理端界面轮播',
    mobileLabel: '移动端界面轮播',
    adminSlides: [
      {
        src: '/screenshots/admin-applications.png',
        alt: 'AppKernia React 管理端应用管理页面',
        title: '应用管理',
        detail: '从一个工作台组织应用配置、团队权限与日常运营。',
      },
      {
        src: '/screenshots/admin-content-management.png',
        alt: 'AppKernia React 管理端内容管理页面',
        title: '内容管理',
        detail: '在统一工作台完成内容、状态与发布流程管理。',
      },
      {
        src: '/screenshots/admin-mobile-releases.png',
        alt: 'AppKernia React 管理端移动版本发布页面',
        title: '移动版本',
        detail: '集中维护移动版本、升级策略和发布状态。',
      },
      {
        src: '/screenshots/admin-cloud-storage.jpg',
        alt: 'AppKernia React 管理端云存储配置页面',
        title: '云存储配置',
        detail: '通过统一配置接入对象存储，并为业务提供稳定的文件能力。',
      },
    ],
    mobileSlides: [
      {
        src: '/screenshots/mobile-login-ios.png',
        alt: 'AppKernia Mobile 登录页面',
        title: '安全登录',
        detail: '登录、会话和语言能力从移动端入口开始协同。',
      },
      {
        src: '/screenshots/mobile-notifications-ios.png',
        alt: 'AppKernia Mobile 通知页面',
        title: '通知中心',
        detail: '把消息、已读状态和用户操作组织成清晰的通知体验。',
      },
      {
        src: '/screenshots/mobile-articles-ios.png',
        alt: 'AppKernia Mobile 文章页面',
        title: '内容阅读',
        detail: '内容模型与服务端发布链路在移动端自然落地。',
      },
      {
        src: '/screenshots/mobile-profile-ios.png',
        alt: 'AppKernia Mobile 个人中心',
        title: '个人中心',
        detail: '账户、设置与用户能力保持清晰的信息层级。',
      },
    ],
  },
  'en-US': {
    eyebrow: 'PRODUCT, END TO END',
    heading: 'From the operations console to the phone, grow one product across every surface',
    lead: 'Manage applications, content, releases, and storage in Admin, then carry sign-in, notifications, reading, and account experiences into Mobile. Both surfaces meet through the same API and product rules.',
    slider: {
      previous: 'Previous slide',
      next: 'Next slide',
      jumpTo: 'Go to',
    },
    adminLabel: 'Web admin interface carousel',
    mobileLabel: 'Mobile interface carousel',
    adminSlides: [
      {
        src: '/screenshots/admin-applications.png',
        alt: 'Applications page in AppKernia React Admin',
        title: 'Applications',
        detail: 'Organize application settings, team access, and daily operations in one place.',
      },
      {
        src: '/screenshots/admin-content-management.png',
        alt: 'Content management page in AppKernia React Admin',
        title: 'Content management',
        detail: 'Manage content, state, and publishing workflows in one workspace.',
      },
      {
        src: '/screenshots/admin-mobile-releases.png',
        alt: 'Mobile releases page in AppKernia React Admin',
        title: 'Mobile releases',
        detail: 'Maintain app versions, update policy, and release state centrally.',
      },
      {
        src: '/screenshots/admin-cloud-storage.jpg',
        alt: 'Cloud storage settings in AppKernia React Admin',
        title: 'Cloud storage',
        detail:
          'Connect object storage once and offer a consistent file capability to the product.',
      },
    ],
    mobileSlides: [
      {
        src: '/screenshots/mobile-login-ios.png',
        alt: 'AppKernia Mobile sign-in screen',
        title: 'Secure sign-in',
        detail: 'Identity, sessions, and locale start working together at the mobile entry point.',
      },
      {
        src: '/screenshots/mobile-notifications-ios.png',
        alt: 'AppKernia Mobile notifications screen',
        title: 'Notifications',
        detail:
          'Bring messages, read state, and user actions into one clear notification experience.',
      },
      {
        src: '/screenshots/mobile-articles-ios.png',
        alt: 'AppKernia Mobile articles screen',
        title: 'Content reading',
        detail: 'Server-side content and publishing contracts land naturally in the app.',
      },
      {
        src: '/screenshots/mobile-profile-ios.png',
        alt: 'AppKernia Mobile profile screen',
        title: 'Profile',
        detail: 'Account, settings, and user capabilities keep a clear information hierarchy.',
      },
    ],
  },
} satisfies Record<
  HomeLocale,
  {
    eyebrow: string;
    heading: string;
    lead: string;
    slider: Omit<SliderCopy, 'label'>;
    adminLabel: string;
    mobileLabel: string;
    adminSlides: ProductSlide[];
    mobileSlides: ProductSlide[];
  }
>;

const TECH_COPY = {
  'zh-CN': {
    eyebrow: 'BUILT ON PROVEN TOOLS',
    heading: '一套技术栈，各做自己最擅长的事',
    lead: 'uni-app x 连接移动平台，React 组织管理体验，Go 与 PostgreSQL 承载业务和数据，OpenAPI 让三端始终说同一种语言。',
    label: 'AppKernia 技术栈',
    roles: {
      mobile: '移动端',
      admin: '管理端',
      adminUi: '管理端 UI',
      server: '服务端',
      data: '数据层',
      contract: '契约',
      runtime: '运行环境',
    },
  },
  'en-US': {
    eyebrow: 'BUILT ON PROVEN TOOLS',
    heading: 'One stack, with every tool doing what it does best',
    lead: 'uni-app x connects mobile platforms, React shapes the operations experience, Go and PostgreSQL carry the business and its data, and OpenAPI keeps every surface speaking the same language.',
    label: 'AppKernia technology stack',
    roles: {
      mobile: 'Mobile',
      admin: 'Admin',
      adminUi: 'Admin UI',
      server: 'Server',
      data: 'Data',
      contract: 'Contract',
      runtime: 'Runtime',
    },
  },
} satisfies Record<
  HomeLocale,
  {
    eyebrow: string;
    heading: string;
    lead: string;
    label: string;
    roles: Record<TechRole, string>;
  }
>;

type TechRole = 'mobile' | 'admin' | 'adminUi' | 'server' | 'data' | 'contract' | 'runtime';

const TECH_STACK: Array<{
  name: string;
  role: TechRole;
  icon?: string;
  color?: string;
  image?: string;
}> = [
  {
    name: 'uni-app x',
    role: 'mobile',
    image: '/tech/uni-app.svg',
  },
  { name: 'React', role: 'admin', icon: reactLogo, color: '#61dafb' },
  { name: 'TypeScript', role: 'admin', icon: typescriptLogo, color: '#3178c6' },
  { name: 'Vite', role: 'admin', icon: viteLogo, color: '#9135ff' },
  { name: 'Go', role: 'server', icon: goLogo, color: '#00add8' },
  { name: 'PostgreSQL', role: 'data', icon: postgresqlLogo, color: '#4169e1' },
  { name: 'OpenAPI', role: 'contract', icon: openapiLogo, color: '#6ba539' },
  { name: 'Docker', role: 'runtime', icon: dockerLogo, color: '#2496ed' },
  { name: 'Ant Design', role: 'adminUi', icon: antdesignLogo, color: '#0170fe' },
];

function ArrowIcon({ direction }: { direction: 'left' | 'right' }) {
  return (
    <svg aria-hidden="true" viewBox="0 0 24 24">
      <path
        d={direction === 'left' ? 'm15 18-6-6 6-6' : 'm9 18 6-6-6-6'}
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="2"
      />
    </svg>
  );
}

function ProductSlider({ kind, slides, copy }: ProductSliderProps) {
  const [activeIndex, setActiveIndex] = useState(0);
  const sliderId = useId();
  const activeSlide = slides[activeIndex];

  const show = (index: number) => {
    setActiveIndex((index + slides.length) % slides.length);
  };
  const onKeyDown = (event: KeyboardEvent<HTMLButtonElement>) => {
    if (event.key === 'ArrowLeft') {
      event.preventDefault();
      show(activeIndex - 1);
    }
    if (event.key === 'ArrowRight') {
      event.preventDefault();
      show(activeIndex + 1);
    }
  };

  return (
    <article
      aria-label={copy.label}
      aria-roledescription="carousel"
      className={`ak-product-slider ak-product-slider--${kind}`}
      role="region"
    >
      <div className="ak-product-slider__viewport">
        {kind === 'admin' ? (
          <div className="ak-slider-browser">
            <div aria-hidden="true" className="ak-slider-browser__bar">
              <span />
              <span />
              <span />
              <strong>appkernia / admin</strong>
            </div>
            <img alt={activeSlide.alt} src={normalizeImagePath(activeSlide.src)} />
          </div>
        ) : (
          <div className="ak-slider-phone">
            <span aria-hidden="true" className="ak-slider-phone__sensor" />
            <img alt={activeSlide.alt} src={normalizeImagePath(activeSlide.src)} />
          </div>
        )}
      </div>

      <div className="ak-product-slider__caption">
        <div>
          <p>{activeSlide.title}</p>
          <span>{activeSlide.detail}</span>
        </div>
        <strong aria-hidden="true">
          {String(activeIndex + 1).padStart(2, '0')} / {String(slides.length).padStart(2, '0')}
        </strong>
      </div>

      <div className="ak-product-slider__controls">
        <button
          aria-label={copy.previous}
          onClick={() => show(activeIndex - 1)}
          onKeyDown={onKeyDown}
          type="button"
        >
          <ArrowIcon direction="left" />
        </button>
        <div aria-label={copy.label} className="ak-product-slider__dots" role="group">
          {slides.map((slide, index) => (
            <button
              aria-label={`${copy.jumpTo} ${slide.title}`}
              aria-pressed={index === activeIndex}
              className={index === activeIndex ? 'is-active' : undefined}
              key={slide.src}
              onClick={() => show(index)}
              onKeyDown={onKeyDown}
              type="button"
            />
          ))}
        </div>
        <button
          aria-label={copy.next}
          onClick={() => show(activeIndex + 1)}
          onKeyDown={onKeyDown}
          type="button"
        >
          <ArrowIcon direction="right" />
        </button>
      </div>

      <span aria-live="polite" className="ak-visually-hidden" id={sliderId}>
        {activeSlide.title}, {activeIndex + 1} / {slides.length}
      </span>
    </article>
  );
}

export function HomeProductShowcase({ locale }: { locale: HomeLocale }) {
  const copy = PRODUCT_COPY[locale];
  return (
    <section aria-labelledby="product-tour-title" className="ak-home-section ak-product-tour">
      <p className="ak-home-eyebrow">{copy.eyebrow}</p>
      <h2 className="ak-home-heading" id="product-tour-title">
        {copy.heading}
      </h2>
      <p className="ak-home-lead">{copy.lead}</p>
      <div className="ak-product-slider-grid">
        <ProductSlider
          copy={{ ...copy.slider, label: copy.adminLabel }}
          kind="admin"
          slides={copy.adminSlides}
        />
        <ProductSlider
          copy={{ ...copy.slider, label: copy.mobileLabel }}
          kind="mobile"
          slides={copy.mobileSlides}
        />
      </div>
    </section>
  );
}

function TechIcon({ icon, color, name }: { icon: string; color: string; name: string }) {
  return (
    <span
      aria-label={`${name} logo`}
      className="ak-tech-brand-mark"
      role="img"
      style={{
        backgroundColor: color,
        maskImage: `url(${icon})`,
        WebkitMaskImage: `url(${icon})`,
      }}
    />
  );
}

export function HomeTechStack({ locale }: { locale: HomeLocale }) {
  const copy = TECH_COPY[locale];
  return (
    <section aria-labelledby="tech-stack-title" className="ak-home-section ak-tech-stack-section">
      <p className="ak-home-eyebrow">{copy.eyebrow}</p>
      <h2 className="ak-home-heading" id="tech-stack-title">
        {copy.heading}
      </h2>
      <p className="ak-home-lead">{copy.lead}</p>
      <ul aria-label={copy.label} className="ak-tech-logo-grid">
        {TECH_STACK.map((item) => (
          <li className="ak-tech-logo-card" key={item.name}>
            <span className="ak-tech-logo-card__mark">
              {item.icon ? (
                <TechIcon color={item.color ?? 'currentColor'} icon={item.icon} name={item.name} />
              ) : (
                <img alt={`${item.name} logo`} src={normalizeImagePath(item.image ?? '')} />
              )}
            </span>
            <span>
              <strong>{item.name}</strong>
              <small>{copy.roles[item.role]}</small>
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}
