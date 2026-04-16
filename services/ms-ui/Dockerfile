# services/ms-ui/Dockerfile
#
# Phase hiện tại (placeholder): Nginx serve static HTML trực tiếp
# Phase tương lai (Sprint 4-5): Uncomment stage Node.js để build Vue 3 + Vite
#
# ── FUTURE: Uncomment khi có Vue 3 source ──────────────────────────────────────
# FROM node:20-alpine AS builder
# WORKDIR /app
# COPY package*.json ./
# RUN npm ci
# COPY . .
# RUN npm run build
# ──────────────────────────────────────────────────────────────────────────────

FROM nginx:1.27-alpine

# Phase hiện tại: copy static HTML placeholder
COPY index.html /usr/share/nginx/html/index.html

# Phase tương lai: thay bằng dòng này
# COPY --from=builder /app/dist /usr/share/nginx/html

COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
