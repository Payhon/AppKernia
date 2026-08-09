declare module '*.css';

declare module '*.svg' {
  const source: string;
  export default source;
}

interface ImportMetaEnv {
  readonly SSG_MD: boolean;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
