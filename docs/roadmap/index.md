# 🗺️ Lộ Trình Học Solution Architect — Mục Lục Tổng Quan

> **Mục tiêu:** Fullstack → Microservices → Cloud-Native → Solution Architect
> **Dự án thực hành:** Blog Engine (Next.js + Go microservices + Docker Compose)
> **Tổng thời gian:** ~6–9 tháng (2–3 giờ/ngày)
> **Cập nhật lần cuối:** 2026-04-18

---

## 📖 Giải Thích Từ Chuyên Ngành Dùng Xuyên Suốt

| Từ chuyên ngành | Giải thích |
|---|---|
| **Solution Architect** | Kiến trúc sư giải pháp — người thiết kế tổng thể hệ thống, quyết định công nghệ |
| **Microservices** | Kiến trúc chia ứng dụng thành nhiều dịch vụ nhỏ, độc lập, chạy riêng |
| **Cloud-Native** | Ứng dụng được thiết kế để chạy trên môi trường đám mây (cloud) |
| **Docker** | Công cụ đóng gói ứng dụng vào container để chạy nhất quán mọi môi trường |
| **Container** | "Hộp" cô lập chứa ứng dụng + thư viện, nhẹ hơn máy ảo (VM) |
| **Docker Compose** | Công cụ định nghĩa và chạy nhiều container cùng lúc bằng file YAML |
| **API Gateway** | Cổng vào duy nhất cho toàn bộ hệ thống, xử lý routing, auth, rate limit |
| **Next.js** | Framework React để build web app, hỗ trợ SSR, ISR, RSC |
| **Go / Golang** | Ngôn ngữ lập trình của Google, nhanh, nhẹ, phổ biến cho backend |

---

## Các Phase Học

| # | Phase | File | Thời lượng | Milestone |
|---|-------|------|----------|-----------|
| 1 | Nền Tảng & Tư Duy Kiến Trúc | [phase_01_foundations.md](./phase_01_foundations.md) | 3–4 tuần | Scaffold repo, ADR đầu tiên |
| 2 | Docker & Containerization | [phase_02_docker.md](./phase_02_docker.md) | 3–4 tuần | Tất cả service đóng gói, image < 20 MB |
| 3 | Docker Compose & Đa Dịch Vụ | [phase_03_docker_compose.md](./phase_03_docker_compose.md) | 2–3 tuần | Toàn bộ stack chạy local với healthcheck |
| 4 | Kiến Trúc Microservices | [phase_04_microservices.md](./phase_04_microservices.md) | 4–6 tuần | 4 service độc lập, event-driven, circuit breaker |
| 5 | API Gateway & Reverse Proxy | [phase_05_api_gateway.md](./phase_05_api_gateway.md) | 2–3 tuần | Traefik routing, rate-limit, CORS |
| 6 | Security — AuthN, AuthZ, Secrets | [phase_06_security.md](./phase_06_security.md) | 3–4 tuần | JWT RS256, RBAC, HTTPS, quản lý secrets |
| 7 | Fullstack Integration (Blog Engine) | [phase_07_fullstack.md](./phase_07_fullstack.md) | 4–6 tuần | Blog Engine hoàn chỉnh, chạy được |
| 8 | Observability & Production Hardening | [phase_08_observability.md](./phase_08_observability.md) | 3–4 tuần | Grafana + Jaeger dashboard hoạt động |
| ☆ | Capstone — Thiết Kế Hệ Thống Từ Đầu | [capstone.md](./capstone.md) | 2–3 tuần | Thiết kế hệ thống dạng Twitter |

---

## Theo Dõi Tiến Độ

```
[ ] Phase 1 — Nền Tảng
[ ] Phase 2 — Docker
[ ] Phase 3 — Docker Compose
[ ] Phase 4 — Microservices
[ ] Phase 5 — API Gateway
[ ] Phase 6 — Security
[ ] Phase 7 — Fullstack
[ ] Phase 8 — Observability
[ ] Capstone
```

---

## Blog Engine — Bản Đồ Service

```
Trình duyệt
  │  HTTPS :443
  ▼
Traefik API Gateway         ← Cổng vào duy nhất
  ├── /*             → frontend     (Next.js SSR)
  ├── /api/auth      → user-service (Go)
  ├── /api/posts     → post-service (Go)
  ├── /api/comments  → comment-service (Go)
  └── /api/search    → search-service (Go)
               ↕ mạng nội bộ blog-internal
     postgres   redis   elasticsearch
```

---

## Tổng Quan Timeline

| Phase | Chủ đề | Thời lượng |
|---|---|---|
| 1 | Nền tảng & Tư duy | 3–4 tuần |
| 2 | Docker | 3–4 tuần |
| 3 | Docker Compose | 2–3 tuần |
| 4 | Microservices | 4–6 tuần |
| 5 | API Gateway | 2–3 tuần |
| 6 | Security | 3–4 tuần |
| 7 | Fullstack | 4–6 tuần |
| 8 | Observability | 3–4 tuần |
| ☆ | Capstone | 2–3 tuần |
| **Tổng** | | **26–37 tuần** |

---

## Sách Nên Đọc (theo thứ tự ưu tiên)

1. **"Designing Data-Intensive Applications"** — Kleppmann ⭐ Đọc đầu tiên
2. **"Building Microservices"** — Sam Newman
3. **"System Design Interview"** — Alex Xu (Tập 1 & 2)
4. **"Let's Go!"** + **"Let's Go Further!"** — Alex Edwards
5. **"Docker Deep Dive"** — Nigel Poulton
6. **"The Web Application Hacker's Handbook"** (lần 2)

---

*Roadmap v1.0 — 2026-04-18*
