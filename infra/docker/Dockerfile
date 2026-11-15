# Multi-stage build: static frontend + Go API in one container.
FROM node:20-alpine AS web
WORKDIR /repo
COPY package.json package-lock.json ./
COPY apps/web ./apps/web
RUN npm install --workspace=apps/web --include-workspace-root=false
# Base of canonical links and sitemap.xml in the prerendered public pages (ADR-0042); empty = production address.
ARG VITE_SITE_URL=
# Builds the app, then prerenders the public pages into dist/prerendered/ with dist/routes.json and dist/sitemap.xml;
# all of dist/ is copied below as before.
RUN npm run build -w apps/web

FROM golang:1.25-alpine AS api
WORKDIR /src
COPY apps/api ./apps/api
WORKDIR /src/apps/api
RUN go build -o /out/api ./cmd/server

FROM alpine:3.20
RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=api /out/api ./api
COPY --from=web /repo/apps/web/dist ./public
RUN chown -R app:app /app
USER app
ENV APP_PORT=8080
EXPOSE 8080
ENTRYPOINT ["./api"]
