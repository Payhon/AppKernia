FROM node:24.18.1-alpine AS build

RUN corepack enable && corepack prepare pnpm@11.18.0 --activate
WORKDIR /workspace
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/ak-admin/package.json apps/ak-admin/package.json
RUN pnpm install --frozen-lockfile --filter @appkernia/admin...
COPY apps/ak-admin/ apps/ak-admin/
COPY blueprint/ blueprint/
COPY server/openapi/ server/openapi/
ARG VITE_AK_API_BASE_URL=/admin-api/v1
ARG VITE_AK_APP_ENV=production
ENV VITE_AK_API_BASE_URL=$VITE_AK_API_BASE_URL
ENV VITE_AK_APP_ENV=$VITE_AK_APP_ENV
RUN pnpm --filter @appkernia/admin generate:api \
    && pnpm --filter @appkernia/admin generate:i18n \
    && pnpm --filter @appkernia/admin generate:routes \
    && pnpm --filter @appkernia/admin build

FROM nginx:1.29-alpine
COPY apps/ak-admin/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /workspace/apps/ak-admin/dist/ /usr/share/nginx/html/
EXPOSE 8080
