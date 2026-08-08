import { defineConfig, js, jsxA11yPlugin, reactHooksPlugin, reactPlugin, ts } from '@rslint/core';

export default defineConfig([
  {
    ignores: ['doc_build/**', 'docs/public/openapi.yaml'],
  },
  js.configs.recommended,
  ts.configs.recommended,
  reactPlugin.configs.recommended,
  reactHooksPlugin.configs.recommended,
  jsxA11yPlugin.configs.recommended,
]);
