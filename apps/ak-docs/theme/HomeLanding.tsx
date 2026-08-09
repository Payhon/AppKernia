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
    eyebrow: 'PRODUCT TOUR',
    heading: '从 Web 管理端到移动端，看看这套工程真正运行起来的样子',
    lead: '使用左右按钮、圆点或键盘方向键浏览。全部画面来自仓库对应的本机隔离测试环境，不使用概念稿替代运行证据。',
    note: 'Admin 截图来自本机 Docker/API 环境；Mobile 截图来自 iPhone 16 Pro、iOS 18.6 模拟器。本展示不等同于生产部署或 iOS、Android、HarmonyOS 真机验收。',
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
        detail: '以租户和权限边界组织应用配置与运营入口。',
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
        detail: '以受控配置管理对象存储能力与供应商边界。',
      },
    ],
    mobileSlides: [
      {
        src: '/screenshots/mobile-login-ios.png',
        alt: 'AppKernia 在 iOS 18.6 模拟器中的登录页面',
        title: '安全登录',
        detail: '登录、会话和语言能力从移动端入口开始协同。',
      },
      {
        src: '/screenshots/mobile-notifications-ios.png',
        alt: 'AppKernia 在 iOS 18.6 模拟器中的通知页面',
        title: '通知中心',
        detail: '面向真实 App 场景组织通知列表与已读状态。',
      },
      {
        src: '/screenshots/mobile-articles-ios.png',
        alt: 'AppKernia 在 iOS 18.6 模拟器中的文章页面',
        title: '内容阅读',
        detail: '内容模型与服务端发布链路在移动端自然落地。',
      },
      {
        src: '/screenshots/mobile-profile-ios.png',
        alt: 'AppKernia 在 iOS 18.6 模拟器中的个人中心',
        title: '个人中心',
        detail: '账户、设置与用户能力保持清晰的信息层级。',
      },
    ],
  },
  'en-US': {
    eyebrow: 'PRODUCT TOUR',
    heading: 'See the system running—from the Web console to the mobile app',
    lead: 'Browse with the arrow buttons, dots, or keyboard arrow keys. Every screen comes from the repository in an isolated local test environment, never from a concept mockup.',
    note: 'Admin images come from a local Docker/API environment. Mobile images come from an iPhone 16 Pro iOS 18.6 simulator. This gallery is not production or iOS, Android, or HarmonyOS physical-device acceptance.',
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
        detail:
          'Organize app configuration and operations around tenant and permission boundaries.',
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
          'Control object-storage capabilities and provider boundaries through safe configuration.',
      },
    ],
    mobileSlides: [
      {
        src: '/screenshots/mobile-login-ios.png',
        alt: 'AppKernia sign-in screen on an iOS 18.6 simulator',
        title: 'Secure sign-in',
        detail: 'Identity, sessions, and locale start working together at the mobile entry point.',
      },
      {
        src: '/screenshots/mobile-notifications-ios.png',
        alt: 'AppKernia notifications on an iOS 18.6 simulator',
        title: 'Notifications',
        detail: 'Notification lists and read state designed for a real application workflow.',
      },
      {
        src: '/screenshots/mobile-articles-ios.png',
        alt: 'AppKernia articles on an iOS 18.6 simulator',
        title: 'Content reading',
        detail: 'Server-side content and publishing contracts land naturally in the app.',
      },
      {
        src: '/screenshots/mobile-profile-ios.png',
        alt: 'AppKernia profile on an iOS 18.6 simulator',
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
    note: string;
    slider: Omit<SliderCopy, 'label'>;
    adminLabel: string;
    mobileLabel: string;
    adminSlides: ProductSlide[];
    mobileSlides: ProductSlide[];
  }
>;

const TECH_COPY = {
  'zh-CN': {
    eyebrow: 'TECHNOLOGY STACK',
    heading: '选择成熟、透明、能长期维护的技术栈',
    lead: '不是为了罗列 Logo，而是让每一层都使用适合其职责的工具：移动端面向原生能力，Web 端重视类型与交互，服务端坚持契约和数据边界。',
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
    eyebrow: 'TECHNOLOGY STACK',
    heading: 'Built on mature, transparent technology that teams can maintain',
    lead: 'The logos are not a checklist. Each layer uses tools that fit its job: native capability on mobile, typed interaction on the Web, and explicit contracts and data boundaries on the server.',
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
      <p className="ak-proof-note">{copy.note}</p>
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
