# Phase 7 — Fullstack Integration: Blog Engine Hoàn Chỉnh

> **Thời lượng:** 4–6 tuần
> **Trước đó:** [Phase 6 — Security](./phase_06_security.md)
> **Tiếp theo:** [Phase 8 — Observability](./phase_08_observability.md)
> **Milestone:** Blog Engine hoàn chỉnh — đăng ký, đăng nhập, viết bài, comment, tìm kiếm, thông báo realtime

---

## 📖 Từ Chuyên Ngành Trong Phase Này

| Từ chuyên ngành | Giải thích |
|---|---|
| **SSR** (Server-Side Rendering) | Server render HTML trước khi gửi về browser — tốt cho SEO |
| **CSR** (Client-Side Rendering) | Browser render HTML sau khi nhận JavaScript — như SPA truyền thống |
| **ISR** (Incremental Static Regeneration) | Next.js: render tĩnh nhưng tự động renovate sau N giây |
| **RSC** (React Server Components) | Component chạy trên server, không gửi JS về client |
| **App Router** | Hệ thống routing mới của Next.js dùng thư mục `app/` |
| **WebSocket** | Giao thức kết nối hai chiều lâu dài — dùng cho realtime |
| **Presigned URL** | URL tạm thời có quyền truy cập giới hạn — dùng để upload thẳng lên storage |
| **MinIO** | Object storage tự host tương thích S3 — lưu file, ảnh |
| **TipTap** | Thư viện rich text editor headless cho React |
| **SWR** | Thư viện data fetching cho React — Stale While Revalidate |
| **Metadata API** | Next.js API để khai báo SEO metadata (title, description, og:image) |
| **httpOnly Cookie** | Cookie không thể đọc bằng JavaScript — bảo vệ khỏi XSS |
| **Edge Runtime** | Runtime của Next.js middleware — thực thi ở CDN edge, rất nhanh |
| **Idempotency Key** | UUID kèm theo request để phát hiện request trùng lặp |
| **SEO** (Search Engine Optimization) | Tối ưu hóa để xuất hiện cao trên công cụ tìm kiếm |

---

## 7.1 Kiến Trúc Hệ Thống Cuối Cùng

```
Browser / iOS App
        │  HTTPS :443
        ▼
   Traefik API Gateway
        ├── /*              → Next.js Frontend      (SSR, ISR, RSC)
        ├── /api/auth/*     → user-service   :8080  (Go)
        ├── /api/posts/*    → post-service   :8081  (Go)
        ├── /api/comments/* → comment-service:8082  (Go)
        ├── /api/search/*   → search-service :8083  (Go + Elasticsearch)
        └── /ws/*           → ws-service     :8084  (Go WebSocket)
                   ↕ mạng nội bộ blog-data
              postgres-users    postgres-posts
              postgres-comments elasticsearch
              redis             minio (S3-compatible)
```

---

## 7.2 Cấu Trúc Next.js App Router

```
frontend/
├── app/
│   ├── layout.tsx                  ← Root layout: font, theme, navbar
│   ├── page.tsx                    ← Trang chủ: bài viết mới nhất (SSR)
│   ├── (auth)/                     ← Route group — không tạo segment trong URL
│   │   ├── login/page.tsx          ← Form đăng nhập
│   │   └── register/page.tsx       ← Form đăng ký
│   ├── posts/
│   │   ├── page.tsx                ← Danh sách bài với phân trang
│   │   ├── [id]/page.tsx           ← Chi tiết bài + comment (SSR)
│   │   └── new/page.tsx            ← Tạo bài mới (protected)
│   ├── dashboard/
│   │   └── page.tsx                ← Dashboard tác giả (protected)
│   └── admin/
│       └── page.tsx                ← Admin panel (role: admin)
├── components/
│   ├── posts/
│   │   ├── PostCard.tsx
│   │   ├── PostList.tsx
│   │   └── PostEditor.tsx          ← TipTap rich text editor
│   └── comments/
│       └── CommentSection.tsx      ← Realtime qua WebSocket
├── lib/
│   ├── api/
│   │   ├── client.ts               ← Base API client (fetch wrapper)
│   │   ├── posts.ts                ← Post API calls
│   │   ├── auth.ts                 ← Auth API calls
│   │   └── comments.ts
│   └── websocket.ts                ← WebSocket client
└── middleware.ts                    ← Bảo vệ route (Edge Runtime)
```

---

## 7.3 Chiến Lược Data Fetching

| Chiến lược | Next.js API | Khi dùng | Blog Engine |
|---|---|---|---|
| SSR | `{cache: 'no-store'}` | Data mới mỗi request | Danh sách bài, comment |
| SSG | `fetch(url)` (mặc định) | Hiếm khi thay đổi | Trang About |
| ISR | `{next: {revalidate: 60}}` | Bán tĩnh | Chi tiết bài viết |
| CSR | `useEffect` / SWR | Cá nhân hóa, tương tác | Dashboard, editor |

```typescript
// app/page.tsx — Trang chủ (SSR: luôn fresh)
export default async function HomePage() {
  // cache: 'no-store' → fetch mới mỗi request (SSR)
  const posts = await fetch(`${process.env.API_URL}/api/posts?limit=10`, {
    cache: 'no-store'
  }).then(r => r.json());

  return (
    <main>
      <h1>Bài Viết Mới Nhất</h1>
      <PostList posts={posts.data} />
    </main>
  );
}

// app/posts/[id]/page.tsx — Chi tiết bài (ISR + SSR cho comment)
export default async function PostDetailPage({ params }: { params: { id: string } }) {
  // Fetch song song để nhanh hơn
  const [post, comments] = await Promise.all([
    fetch(`${process.env.API_URL}/api/posts/${params.id}`, {
      next: { revalidate: 60 }    // ISR: có thể cũ tối đa 60 giây
    }).then(r => r.json()),
    fetch(`${process.env.API_URL}/api/comments?post_id=${params.id}`, {
      cache: 'no-store'           // Comment luôn fresh
    }).then(r => r.json()),
  ]);

  return (
    <article>
      <PostDetail post={post} />
      <CommentSection postId={params.id} initialComments={comments.data} />
    </article>
  );
}
```

---

## 7.4 Typed API Client (Client API Có Type)

```typescript
// lib/api/client.ts — Wrapper của fetch, tự thêm auth header

const API_BASE = process.env.NEXT_PUBLIC_API_URL!;

class ApiClient {
  private getAuthHeader(): HeadersInit {
    // typeof window === 'undefined' → đang ở SSR, không có localStorage
    if (typeof window === 'undefined') return {};
    const token = localStorage.getItem('access_token');
    return token ? { Authorization: `Bearer ${token}` } : {};
  }

  async get<T>(path: string, options?: RequestInit): Promise<T> {
    const res = await fetch(`${API_BASE}${path}`, {
      ...options,
      headers: { ...this.getAuthHeader(), ...options?.headers },
    });
    if (!res.ok) throw new Error((await res.json()).message);
    return res.json();
  }

  async post<T>(path: string, body: unknown, options?: RequestInit): Promise<T> {
    const res = await fetch(`${API_BASE}${path}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...this.getAuthHeader(),
        ...options?.headers,
      },
      body: JSON.stringify(body),
      ...options,
    });
    if (!res.ok) throw new Error((await res.json()).message);
    return res.json();
  }
}

export const apiClient = new ApiClient();
```

```typescript
// lib/api/posts.ts — API layer theo domain

export const postsApi = {
  list: (page = 1, limit = 10, tag?: string): Promise<PostListResponse> => {
    const params = new URLSearchParams({ page: String(page), limit: String(limit) });
    if (tag) params.append('tag', tag);
    return apiClient.get(`/api/posts?${params}`);
  },

  getById: (id: string): Promise<Post> => apiClient.get(`/api/posts/${id}`),

  create: (req: CreatePostRequest): Promise<Post> => {
    // Idempotency key để tránh tạo bài trùng khi retry
    const idempotencyKey = crypto.randomUUID();
    return apiClient.post('/api/posts', req, {
      headers: { 'Idempotency-Key': idempotencyKey },
    });
  },

  publish: (id: string): Promise<Post> => apiClient.post(`/api/posts/${id}/publish`, {}),
  delete: (id: string): Promise<void> => apiClient.delete(`/api/posts/${id}`),
};
```

---

## 7.5 Luồng Authentication

### Next.js Middleware — Bảo Vệ Route

```typescript
// middleware.ts (chạy ở Edge Runtime — rất nhanh, không cần full Node.js)

import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

const protectedPaths = ['/dashboard', '/posts/new', '/admin'];
const adminPaths = ['/admin'];

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;
  // Đọc access_token từ cookie (được set sau khi đăng nhập)
  const token = request.cookies.get('access_token')?.value;

  const requiresAuth = protectedPaths.some(p => pathname.startsWith(p));
  if (requiresAuth && !token) {
    // Redirect về login, kèm URL để sau login redirect về đây
    const loginUrl = new URL('/login', request.url);
    loginUrl.searchParams.set('redirect', pathname);
    return NextResponse.redirect(loginUrl);
  }

  // Kiểm tra admin routes
  if (adminPaths.some(p => pathname.startsWith(p)) && token) {
    try {
      const [, payload] = token.split('.');
      const { roles } = JSON.parse(atob(payload));  // decode JWT payload
      if (!roles?.includes('admin')) {
        return NextResponse.redirect(new URL('/', request.url));
      }
    } catch {
      return NextResponse.redirect(new URL('/login', request.url));
    }
  }

  return NextResponse.next();  // cho phép request tiếp tục
}

export const config = {
  // Middleware chỉ chạy cho các path này
  matcher: ['/dashboard/:path*', '/posts/new', '/admin/:path*'],
};
```

### Token Refresh (Tự Động, Trong Suốt)

```typescript
// Hàm refresh token — gọi khi access token hết hạn
export async function refreshAccessToken(): Promise<string | null> {
  const res = await fetch('/api/auth/refresh', {
    method: 'POST',
    credentials: 'include',  // Gửi cookie (bao gồm httpOnly refresh_token)
  });
  if (!res.ok) return null;  // Refresh token hết hạn → phải đăng nhập lại
  const { access_token } = await res.json();
  return access_token;
}

export function isTokenExpired(token: string): boolean {
  try {
    const [, payload] = token.split('.');
    const { exp } = JSON.parse(atob(payload));
    return Date.now() >= exp * 1000;  // exp là Unix timestamp (giây)
  } catch {
    return true;
  }
}
```

---

## 7.6 Realtime Comment qua WebSocket

```typescript
// lib/websocket.ts — WebSocket client với auto-reconnect

class BlogWebSocket {
  private ws: WebSocket | null = null;
  private handlers: Map<string, Function[]> = new Map();

  connect(postId: string, token: string) {
    this.ws = new WebSocket(`wss://localhost/ws/posts/${postId}/comments?token=${token}`);

    this.ws.onmessage = (event) => {
      const message = JSON.parse(event.data);
      const handlers = this.handlers.get(message.type) || [];
      handlers.forEach(h => h(message));
    };

    this.ws.onclose = () => {
      // Auto-reconnect sau 3 giây
      setTimeout(() => this.connect(postId, token), 3000);
    };
  }

  on(type: string, handler: Function) {
    const handlers = this.handlers.get(type) || [];
    this.handlers.set(type, [...handlers, handler]);
  }

  disconnect() { this.ws?.close(); }
}

export const blogWS = new BlogWebSocket();
```

```typescript
// components/comments/CommentSection.tsx
'use client';  // Directive: component này chạy ở client (CSR)

export function CommentSection({ postId, initialComments }: {
  postId: string;
  initialComments: Comment[];
}) {
  const [comments, setComments] = useState<Comment[]>(initialComments);

  useEffect(() => {
    const token = localStorage.getItem('access_token');
    if (!token) return;

    blogWS.connect(postId, token);
    // Khi có comment mới từ server → thêm vào đầu danh sách
    blogWS.on('comment.created', ({ payload }) => {
      setComments(prev => [payload, ...prev]);  // Realtime! Không cần refetch
    });
    // Cleanup khi component unmount
    return () => blogWS.disconnect();
  }, [postId]);

  return (
    <section>
      <h2>{comments.length} Bình luận</h2>
      <CommentForm postId={postId} />
      <ul>{comments.map(c => <CommentItem key={c.id} comment={c} />)}</ul>
    </section>
  );
}
```

---

## 7.7 Upload Ảnh qua MinIO Presigned URL

```
Luồng Upload:
  Client → POST /api/posts/upload-url (filename, content_type)
  post-service → tạo presigned PUT URL từ MinIO (TTL 15 phút)
  ← 200 { presigned_url, key }
  Client → PUT presigned_url (gửi bytes ảnh thẳng lên MinIO — bypass server!)
  ← 200 (upload hoàn tất)
  Client → POST /api/posts { cover_image_key: key }
```

> 💡 **Tại sao dùng Presigned URL?**
> - Client upload thẳng lên storage, không qua server → server không bị nghẽn
> - URL hết hạn sau 15 phút → an toàn hơn
> - Tiết kiệm bandwidth và CPU của server

---

## 7.8 SEO qua Next.js Metadata API

```typescript
// app/posts/[id]/page.tsx — SEO metadata cho từng bài viết

export async function generateMetadata({ params }: { params: { id: string } }): Promise<Metadata> {
  const post = await fetch(`${process.env.API_URL}/api/posts/${params.id}`)
    .then(r => r.json())
    .catch(() => null);

  if (!post) return { title: 'Không Tìm Thấy Bài Viết' };
  // Lấy 160 ký tự đầu, bỏ HTML tags
  const description = post.content.substring(0, 160).replace(/<[^>]+>/g, '');

  return {
    title: `${post.title} | Blog Engine`,
    description,
    openGraph: {
      title: post.title,
      description,
      images: post.coverImageUrl ? [{ url: post.coverImageUrl }] : [],
      type: 'article',
      publishedTime: post.publishedAt,
      authors: [post.authorName],
    },
  };
}
```

> 💡 **Giải thích thuật ngữ:**
> - **Open Graph (og)**: Protocol của Facebook cho phép preview link đẹp khi share lên mạng xã hội
> - **generateMetadata**: Function async của Next.js để tạo metadata động theo params

---

## 7.9 Blog Engine — Lab Milestone 7

### User Journey 1: Người Đọc Mới
```
[ ] 1. Trang chủ load bài viết mới nhất (SSR)
[ ] 2. Click vào bài → trang chi tiết với comment
[ ] 3. Đăng ký → đăng nhập → được lưu access token
[ ] 4. Thêm comment → xuất hiện ngay qua WebSocket
```

### User Journey 2: Tác Giả
```
[ ] 5. Đăng nhập với role author → dashboard hiển thị bài của mình
[ ] 6. Tạo bài mới → TipTap editor với hỗ trợ tag
[ ] 7. Upload ảnh bìa → MinIO presigned URL flow
[ ] 8. Publish bài → follower nhận thông báo qua event
[ ] 9. Sửa bài đã publish → RBAC: chỉ sửa được bài của mình
```

### User Journey 3: Admin
```
[ ] 10. Admin panel: liệt kê tất cả user, quản lý role
[ ] 11. Xóa bất kỳ bài → comment bị cascade xóa qua event
```

### Hoàn Thiện Kỹ Thuật
```
[ ] 12. Lighthouse score ≥ 85 (Performance, SEO, Accessibility)
[ ] 13. Tất cả trang có <title> và <meta description>
[ ] 14. Token refresh tự động (không cần đăng nhập lại sau 15 phút)
[ ] 15. Trang 404 và 500 được style đẹp
[ ] 16. Layout responsive trên mobile
```

---

## 7.10 Tài Nguyên

| Tài nguyên | URL |
|---|---|
| Next.js App Router | https://nextjs.org/docs/app |
| TipTap editor | https://tiptap.dev |
| MinIO Go client | https://github.com/minio/minio-go |
| SWR (data fetching) | https://swr.vercel.app |
| RealWorld API spec | https://github.com/gothinkster/realworld |
| Auth.js | https://authjs.dev |

---

## 7.11 Tự Kiểm Tra — Phase 7 Hoàn Thành Khi…

```
[ ] Trang chủ load bài từ Go post-service qua SSR
[ ] Đăng nhập: JWT trong localStorage, refresh token trong httpOnly cookie
[ ] Protected route redirect về /login khi không có auth
[ ] WebSocket: submit comment thấy ngay ở tất cả người xem
[ ] Upload ảnh hoạt động qua MinIO presigned URL
[ ] Lighthouse performance score ≥ 85
[ ] RBAC: reader không viết được, author không xóa được bài người khác
```

---

*← [Phase 6 — Security](./phase_06_security.md) | [Phase 8 — Observability →](./phase_08_observability.md)*
