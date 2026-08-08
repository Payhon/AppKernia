import path from 'node:path';
import { defineConfig, type UserConfig } from '@rspress/core';
import { pluginSitemap } from '@rspress/plugin-sitemap';
import { rehypeA11y } from './plugins/rehype-a11y';

const siteOrigin = process.env.DOCS_ORIGIN?.trim() || 'https://appkernia.com';
const requestedBase = process.env.DOCS_BASE?.trim() || '/';
const base = requestedBase === '/' ? '/' : `/${requestedBase.replace(/^\/+|\/+$/g, '')}/`;
const siteBaseUrl = `${siteOrigin}${base === '/' ? '' : base.slice(0, -1)}`;

const config: UserConfig = {
  root: path.join(import.meta.dirname, 'docs'),
  outDir: path.join(import.meta.dirname, 'doc_build'),
  title: 'AppKernia',
  description: '一套贯通 Mobile、Admin 与 Server 的现代跨端应用开发基座。',
  lang: 'zh-CN',
  siteOrigin,
  base,
  icon: '/brand/favicon.svg',
  logo: {
    light: '/brand/logo.svg',
    dark: '/brand/logo-dark.svg',
  },
  logoText: 'AppKernia',
  locales: [
    {
      lang: 'zh-CN',
      label: '简体中文',
      title: 'AppKernia',
      description: '一套贯通 Mobile、Admin 与 Server 的现代跨端应用开发基座。',
    },
    {
      lang: 'en-US',
      label: 'English',
      title: 'AppKernia',
      description:
        'A modern cross-platform application foundation spanning mobile, admin, and server.',
    },
  ],
  llms: true,
  languageParity: {
    enabled: true,
  },
  plugins: [
    pluginSitemap({
      defaultChangeFreq: 'weekly',
      defaultPriority: '0.7',
    }),
  ],
  search: {
    mode: 'local',
  },
  mediumZoom: true,
  markdown: {
    rehypePlugins: [rehypeA11y],
    link: {
      checkDeadLinks: {
        excludes: ['/openapi.yaml'],
      },
      checkAnchors: true,
    },
    image: {
      checkDeadImages: true,
    },
  },
  route: {
    localeRedirect: 'auto',
  },
  head: [
    ['meta', { name: 'theme-color', content: '#07111f' }],
    ['meta', { name: 'author', content: 'AppKernia contributors' }],
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:site_name', content: 'AppKernia' }],
    ['meta', { property: 'og:image', content: `${siteBaseUrl}/hero-ecosystem.png` }],
    ['meta', { name: 'twitter:card', content: 'summary_large_image' }],
  ],
  themeConfig: {
    search: true,
    lastUpdated: true,
    enableContentAnimation: true,
    enableAppearanceAnimation: true,
    enableScrollToTop: true,
    editLink: {
      docRepoBaseUrl: 'https://github.com/Payhon/AppKernia/tree/main/apps/ak-docs/docs',
    },
    llmsUI: {
      placement: 'outline',
      viewOptions: ['markdownLink', 'chatgpt', 'claude'],
    },
    locales: [
      {
        lang: 'zh-CN',
        label: '简体中文',
      },
      {
        lang: 'en-US',
        label: 'English',
      },
    ],
  },
};

export default defineConfig(config);
