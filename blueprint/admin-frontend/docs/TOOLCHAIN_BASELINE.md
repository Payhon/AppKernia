# 工具链基线

Node.js 24 LTS、pnpm 11、React 19.2、Vite 8.1、Ant Design 6、TanStack Router v1、TanStack Query v5、TypeScript strict。Phase 0 安装当前安全 patch，验证 Router 代码生成、AntD/Pro Components、RHF/Zod、Vitest/MSW/Playwright 后冻结 `pnpm-lock.yaml`。Patch 可自动合并；minor/major 必须经过 CI 和视觉回归。
