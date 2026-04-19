# Phase 6 — Security: AuthN, AuthZ, Secrets & Hardening

> **Thời lượng:** 3–4 tuần
> **Trước đó:** [Phase 5 — API Gateway](./phase_05_api_gateway.md)
> **Tiếp theo:** [Phase 7 — Fullstack](./phase_07_fullstack.md)
> **Milestone:** JWT RS256 auth, RBAC, HTTPS, quản lý secrets, tự kiểm tra OWASP Top 10

---

## 📖 Từ Chuyên Ngành Trong Phase Này

| Từ chuyên ngành | Giải thích |
|---|---|
| **AuthN** (Authentication) | Xác thực danh tính — "Bạn là ai?" |
| **AuthZ** (Authorization) | Phân quyền — "Bạn được phép làm gì?" |
| **JWT** (JSON Web Token) | Token mang thông tin người dùng, có chữ ký số — không lưu state trên server |
| **RS256** | Thuật toán ký JWT dùng RSA — asymmetric (khóa bất đối xứng) |
| **HS256** | Thuật toán ký JWT dùng HMAC — symmetric (khóa đối xứng) |
| **RBAC** (Role-Based Access Control) | Phân quyền dựa trên vai trò (admin, author, reader...) |
| **bcrypt** | Thuật toán hash password chậm có chủ ý — an toàn hơn MD5/SHA256 |
| **OWASP Top 10** | Danh sách 10 lỗ hổng bảo mật web phổ biến nhất |
| **Access Token** | Token ngắn hạn (15 phút) — dùng để gọi API |
| **Refresh Token** | Token dài hạn (7 ngày) — dùng để lấy Access Token mới |
| **httpOnly Cookie** | Cookie không thể đọc bằng JavaScript — bảo vệ khỏi XSS |
| **HTTPS/TLS** | HTTP được mã hóa bằng TLS — bảo vệ dữ liệu truyền tải |
| **SQL Injection** | Tấn công chèn code SQL độc hại vào query |
| **XSS** (Cross-Site Scripting) | Tấn công chèn script vào trang web |
| **govulncheck** | Tool chính thức của Go để quét lỗ hổng trong dependencies |
| **gitleaks** | Tool phát hiện secret (key, password) bị commit nhầm vào git |
| **Docker Secrets** | Cơ chế Docker Swarm để inject secret an toàn vào container |

---

## 6.1 Tư Duy Bảo Mật — Threat Model Trước Tiên

Trước khi viết code bảo mật, hãy xác định **threat model** (mô hình mối đe dọa):

```
Kẻ tấn công là ai?
  → Người dùng internet ẩn danh cố truy cập data của người khác
  → Người dùng đã xác thực cố leo thang quyền
  → Service bị compromise bên trong mạng

Chúng ta bảo vệ gì?
  → Thông tin đăng nhập (password hash)
  → Nháp bài viết (nội dung riêng tư của tác giả)
  → Khả năng admin

Bảo vệ như thế nào?
  → Authentication: xác minh danh tính (JWT)
  → Authorization: xác minh quyền (RBAC)
  → Encryption: TLS khi truyền tải, bcrypt khi lưu trữ
  → Secrets management: không có credential trong code hoặc log
  → Input validation: từ chối input sai định dạng sớm nhất có thể
```

---

## 6.2 Authentication — JWT RS256

### HS256 vs RS256 — Tại Sao Asymmetric Quan Trọng Trong Microservices

| | HS256 (symmetric — đối xứng) | RS256 (asymmetric — bất đối xứng) |
|---|---|---|
| Thuật toán | HMAC-SHA256 | RSA-SHA256 |
| Khóa | Một shared secret duy nhất | Private key (ký) + Public key (verify) |
| Rủi ro | Mọi service verify đều phải biết secret | Chỉ auth service có private key |
| Nếu bị lộ | Kẻ tấn công có thể giả mạo token | Kẻ tấn công chỉ verify được, không ký được |
| Dùng khi | Monolith (một service) | Microservices (nhiều service verify) |

> 💡 **Ví dụ thực tế với microservices:**
> - **HS256:** post-service, comment-service, search-service đều cần biết secret → 4 chỗ có thể bị lộ
> - **RS256:** public key (*.pub) an toàn để share với mọi service. Chỉ user-service có private key → chỉ 1 chỗ cần bảo mật

### Tạo Cặp Khóa RSA

```bash
openssl genrsa -out secrets/jwt_private_key.pem 4096
# 4096 bit: đủ mạnh cho production (2048 tối thiểu)
openssl rsa -in secrets/jwt_private_key.pem -pubout -out secrets/jwt_public_key.pem
chmod 400 secrets/jwt_private_key.pem  # Chỉ owner đọc được private key
chmod 444 secrets/jwt_public_key.pem   # Mọi người đọc được public key — an toàn
```

### Cấu Trúc JWT

```
Header.Payload.Signature

Header:  {"alg": "RS256", "typ": "JWT"}
Payload: {
  "sub": "550e8400-...",      ← user UUID
  "email": "user@example.com",
  "roles": ["author"],
  "iat": 1713400000,          ← issued at (Unix timestamp — thời điểm phát hành)
  "exp": 1713400900,          ← expires at (iat + 15 phút)
  "jti": "abc123"             ← JWT ID — duy nhất để có thể revoke
}
Signature: RS256(base64(Header) + "." + base64(Payload), privateKey)
```

> 💡 **Payload KHÔNG được mã hóa** — chỉ được ký! Bất kỳ ai cũng có thể đọc payload. Đừng bao giờ để thông tin nhạy cảm (password, credit card) trong JWT.

### Implementation trong user-service

```go
type Claims struct {
    UserID string   `json:"sub"`
    Email  string   `json:"email"`
    Roles  []string `json:"roles"`
    jwt.RegisteredClaims
}

func (s *AuthServiceImpl) IssueAccessToken(ctx context.Context,
    userID, email string, roles []string) (string, error) {

    now := time.Now()
    claims := Claims{
        UserID: userID,
        Email:  email,
        Roles:  roles,
        RegisteredClaims: jwt.RegisteredClaims{
            Subject:   userID,
            IssuedAt:  jwt.NewNumericDate(now),
            ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)), // Token ngắn hạn!
            NotBefore: jwt.NewNumericDate(now),
            ID:        uuid.New().String(), // JWT ID duy nhất
            Issuer:    "user-service",
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
    signed, err := token.SignedString(s.privateKey)
    if err != nil {
        return "", fmt.Errorf("AuthService.IssueAccessToken: %w", err)
    }
    return signed, nil
}

func (s *AuthServiceImpl) ValidateToken(ctx context.Context, tokenStr string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenStr, &Claims{},
        func(t *jwt.Token) (interface{}, error) {
            // QUAN TRỌNG: phải kiểm tra signing method
            // Nếu không check, attacker có thể gửi token HS256 với "alg: none"!
            if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
                return nil, fmt.Errorf("signing method không mong đợi: %v", t.Header["alg"])
            }
            return s.publicKey, nil
        })
    if err != nil {
        return nil, fmt.Errorf("AuthService.ValidateToken: %w", err)
    }
    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, fmt.Errorf("AuthService.ValidateToken: claims không hợp lệ")
    }
    return claims, nil
}
```

### Luồng Đăng Nhập

```go
func (ac *AuthController) Login(c *gin.Context) {
    var req LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.ErrorResponseWithHTTP(c, 400, response.ErrCodeBadRequest, "request không hợp lệ")
        return
    }

    user, err := ac.authService.FindByEmail(c.Request.Context(), req.Email)
    if err != nil {
        // QUAN TRỌNG: cùng một thông báo lỗi cho "không tìm thấy" và "sai password"!
        // Nếu khác nhau → kẻ tấn công có thể dò email hợp lệ (user enumeration attack)
        response.ErrorResponseWithHTTP(c, 401, response.ErrCodeUnauthorized, "thông tin đăng nhập không hợp lệ")
        return
    }

    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
        response.ErrorResponseWithHTTP(c, 401, response.ErrCodeUnauthorized, "thông tin đăng nhập không hợp lệ")
        return
    }

    accessToken, _ := ac.authService.IssueAccessToken(c.Request.Context(), user.ID, user.Email, user.Roles)
    refreshToken, _ := ac.authService.IssueRefreshToken(c.Request.Context(), user.ID)

    // Refresh token trong httpOnly cookie — JS không đọc được, tránh XSS
    c.SetCookie("refresh_token", refreshToken,
        int((7 * 24 * time.Hour).Seconds()),
        "/api/auth/refresh",    // Path: chỉ gửi cookie khi request đến endpoint này
        "",
        true,   // Secure: chỉ gửi qua HTTPS
        true,   // HttpOnly: JS không đọc được
    )

    response.SuccessResponse(c, response.ErrCodeSuccess, LoginResponse{
        AccessToken: accessToken,
        ExpiresIn:   900, // 15 phút = 900 giây
    })
}
```

---

## 6.3 Bảo Mật Password

```go
const BcryptCost = 12   // ~600ms mỗi lần hash — chậm có chủ ý!

func HashPassword(plain string) (string, error) {
    if len(plain) < 8 {
        return "", fmt.Errorf("password phải ít nhất 8 ký tự")
    }
    hash, err := bcrypt.GenerateFromPassword([]byte(plain), BcryptCost)
    if err != nil {
        return "", fmt.Errorf("HashPassword: %w", err)
    }
    return string(hash), nil
}
```

> 💡 **Tại sao bcrypt chậm quan trọng?**
> - MD5: ~10 tỷ guess/giây → crack trong vài phút
> - SHA256: ~1 tỷ guess/giây → crack trong vài giờ
> - bcrypt cost=12: ~1,600 guess/giây → crack trong nhiều năm
>
> **Tuyệt đối không dùng:** MD5, SHA1, SHA256 cho password — chúng quá nhanh!

---

## 6.4 Authorization — RBAC (Phân Quyền Theo Vai Trò)

### Thiết Kế Role Cho Blog Engine

```go
const (
    RoleAdmin  = "admin"   // Toàn quyền — xóa bất kỳ bài, quản lý user
    RoleAuthor = "author"  // Tạo/sửa bài của mình
    RoleReader = "reader"  // Chỉ đọc (mặc định sau đăng ký)
)
```

### RBAC Middleware

```go
// RequireAuth: validate JWT, gắn user vào context
func RequireAuth(authService service.IAuthService) gin.HandlerFunc {
    return func(c *gin.Context) {
        header := c.GetHeader("Authorization")
        if !strings.HasPrefix(header, "Bearer ") {
            response.ErrorResponseWithHTTP(c, 401, response.ErrCodeUnauthorized,
                "thiếu authorization header")
            c.Abort()
            return
        }
        claims, err := authService.ValidateToken(c.Request.Context(),
            strings.TrimPrefix(header, "Bearer "))
        if err != nil {
            response.ErrorResponseWithHTTP(c, 401, response.ErrCodeUnauthorized,
                "token không hợp lệ hoặc đã hết hạn")
            c.Abort()
            return
        }
        // Gắn thông tin user vào context để handler phía sau dùng
        c.Set("user_id", claims.UserID)
        c.Set("user_email", claims.Email)
        c.Set("user_roles", claims.Roles)
        c.Next()
    }
}

// RequireRole: kiểm tra user có role cần thiết không
func RequireRole(roles ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userRoles, _ := c.Get("user_roles")
        userRoleList, _ := userRoles.([]string)

        for _, required := range roles {
            for _, has := range userRoleList {
                if has == required {
                    c.Next()
                    return
                }
            }
        }

        response.ErrorResponseWithHTTP(c, 403, response.ErrCodeForbidden,
            fmt.Sprintf("cần role: %v", roles))
        c.Abort()
    }
}

// RequireOwnership: kiểm tra user có phải chủ sở hữu resource không
func RequireOwnership(resourceOwnerFn func(c *gin.Context) (string, error)) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := c.GetString("user_id")
        userRoles := c.GetStringSlice("user_roles")

        // Admin bỏ qua kiểm tra ownership
        for _, r := range userRoles {
            if r == model.RoleAdmin {
                c.Next()
                return
            }
        }

        ownerID, err := resourceOwnerFn(c)
        if err != nil {
            response.ErrorResponseWithHTTP(c, 404, response.ErrCodeNotFound, "resource không tìm thấy")
            c.Abort()
            return
        }

        if ownerID != userID {
            response.ErrorResponseWithHTTP(c, 403, response.ErrCodeForbidden, "từ chối: không phải chủ sở hữu")
            c.Abort()
            return
        }
        c.Next()
    }
}
```

### Đăng Ký Route Với Auth

```go
func SetupPostRoutes(r *gin.Engine, ctrl *controller.PostController,
    authSvc service.IAuthService) {
    api := r.Group("/api/posts")

    // Route công khai — không cần auth
    api.GET("", ctrl.ListPosts)
    api.GET("/:id", ctrl.GetPost)

    // Route cần xác thực
    authed := api.Group("")
    authed.Use(middleware.RequireAuth(authSvc))
    {
        authed.GET("/feed", ctrl.GetPersonalFeed)  // mọi user đã đăng nhập

        // Chỉ author hoặc admin mới tạo được bài
        authed.POST("", middleware.RequireRole(model.RoleAuthor, model.RoleAdmin), ctrl.CreatePost)

        // Sửa bài: phải có quyền VÀ phải là chủ bài
        authed.PUT("/:id",
            middleware.RequireRole(model.RoleAuthor, model.RoleAdmin),
            middleware.RequireOwnership(ctrl.GetPostOwnerID),
            ctrl.UpdatePost,
        )

        // Xóa bài: chỉ admin
        authed.DELETE("/:id", middleware.RequireRole(model.RoleAdmin), ctrl.DeletePost)
    }
}
```

---

## 6.5 Quản Lý Secrets

### Các Cấp Độ Quản Lý Secrets

```
Cấp 1 (Dev):           File .env (gitignored)
Cấp 2 (CI/CD):         GitHub Actions Secrets / GitLab CI Variables
Cấp 3 (Production):    Docker Secrets (Swarm)
Cấp 4 (Enterprise):    HashiCorp Vault (dynamic secrets, audit log, rotation)
```

### Docker Secrets Pattern

```yaml
services:
  user-service:
    environment:
      # Dùng suffix _FILE — service đọc secret từ path file
      - JWT_PRIVATE_KEY_FILE=/run/secrets/jwt_private_key
      - DB_PASSWORD_FILE=/run/secrets/db_password
    secrets:
      - jwt_private_key
      - db_password

secrets:
  jwt_private_key:
    file: ./secrets/jwt_private_key.pem
```

```go
// Đọc secret từ file (hỗ trợ pattern _FILE)
func loadSecret(envKey string) (string, error) {
    if path := os.Getenv(envKey + "_FILE"); path != "" {
        data, err := os.ReadFile(path)
        if err != nil {
            return "", fmt.Errorf("loadSecret(%s): %w", envKey, err)
        }
        return strings.TrimSpace(string(data)), nil
    }
    if value := os.Getenv(envKey); value != "" {
        return value, nil
    }
    return "", fmt.Errorf("loadSecret: %s chưa được set", envKey)
}
```

### Những Gì KHÔNG BAO GIỜ Được Log

```go
// ❌ KHÔNG BAO GIỜ log thông tin nhạy cảm
global.Logger.Info("đăng nhập", zap.String("password", req.Password))  // KHÔNG!
global.Logger.Info("token", zap.String("token", accessToken))           // KHÔNG!

// ✅ Chỉ log identifier an toàn
global.Logger.Info("user đã đăng nhập",
    zap.String("user_id", user.ID),
    zap.String("trace_id", traceID),
)
```

> 💡 **Tại sao quan trọng?** Log thường được gửi đến nhiều hệ thống (Elasticsearch, Cloudwatch...). Password/token trong log = data breach!

---

## 6.6 Input Validation & Phòng Chống SQL Injection

```go
// ✅ Struct tags cho Gin validation tự động
type CreatePostRequest struct {
    Title   string   `json:"title"   binding:"required,min=1,max=255"`
    Content string   `json:"content" binding:"required,min=1"`
    Tags    []string `json:"tags"    binding:"omitempty,max=10,dive,min=1,max=50"`
}

// ✅ Parameterized queries qua squirrel — KHÔNG BAO GIỜ string interpolation
query, args, _ := sq.Select("*").From("posts").
    Where(sq.Eq{"author_id": authorID}).  // ← tham số hóa, an toàn
    PlaceholderFormat(sq.Dollar).
    ToSql()

// ❌ KHÔNG BAO GIỜ string interpolation trong SQL
query := fmt.Sprintf(
    "SELECT * FROM posts WHERE author_id = '%s'",
    authorID,  // SQL INJECTION nếu authorID chứa ' OR '1'='1
)
```

> 💡 **SQL Injection ví dụ tấn công:**
> Nếu `authorID = "' OR '1'='1"` thì query trở thành:
> ```sql
> SELECT * FROM posts WHERE author_id = '' OR '1'='1'
> ```
> → Trả về TẤT CẢ bài viết của mọi người! Parameterized queries xử lý giá trị như data, không như code SQL.

---

## 6.7 OWASP Top 10 — Checklist Blog Engine

| # | Lỗ hổng | Biện pháp bảo vệ |
|---|---|---|
| A01 | Broken Access Control | RBAC middleware + kiểm tra ownership |
| A02 | Cryptographic Failures | RS256 JWT, bcrypt cost=12, TLS 1.2+ |
| A03 | Injection | Parameterized queries, ShouldBindJSON validation |
| A04 | Insecure Design | Threat model, principle of least privilege |
| A05 | Security Misconfiguration | Không có default password, config trong env |
| A06 | Vulnerable Components | `govulncheck ./...`, `npm audit` |
| A07 | Auth & Session Failures | JWT RS256, refresh rotation, httpOnly cookie |
| A08 | Software Integrity | Pin phiên bản Docker image |
| A09 | Logging Failures | Structured log, không có PII trong log |
| A10 | SSRF | Allowlist cho HTTP calls ra bên ngoài |

> 💡 **SSRF** (Server-Side Request Forgery): Tấn công khiến server gửi request đến địa chỉ tùy ý (kể cả internal service). Ngăn bằng cách chỉ cho phép gọi đến các domain đã định sẵn (allowlist).
> **PII** (Personally Identifiable Information): Thông tin nhận dạng cá nhân — email, số điện thoại, địa chỉ.
> **Principle of Least Privilege**: Mỗi component chỉ có quyền tối thiểu cần thiết.

---

## 6.8 Công Cụ Quét Bảo Mật

```bash
# Quét lỗ hổng Go (tool chính thức)
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...

# Phân tích static
go vet ./...
staticcheck ./...

# Quét Docker image
docker scout cves blog/post-service:local

# Quét npm dependencies
cd frontend && npm audit

# Phát hiện secret bị commit nhầm
brew install gitleaks
gitleaks detect --source . --verbose
gitleaks git --verbose   # quét toàn bộ git history

# Thêm pre-commit hook để ngăn commit secret
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/sh
gitleaks protect --staged -v
EOF
chmod +x .git/hooks/pre-commit
```

> 💡 **Pre-commit hook**: Script tự động chạy trước khi commit — nếu phát hiện secret sẽ block commit lại.
> **Static analysis**: Phân tích code mà không cần chạy — tìm lỗi tiềm ẩn, anti-pattern.

---

## 6.9 Blog Engine — Lab Milestone 6

```
[ ] 1. Tạo cặp khóa RSA (4096-bit)
        openssl genrsa -out secrets/jwt_private_key.pem 4096
        openssl rsa -in secrets/jwt_private_key.pem -pubout -out secrets/jwt_public_key.pem
        echo "secrets/" >> .gitignore

[ ] 2. Implement JWT RS256 trong user-service:
        - POST /api/auth/register → bcrypt hash → tạo user
        - POST /api/auth/login → validate password → phát token
        - POST /api/auth/refresh → rotate refresh token
        - POST /api/auth/logout → revoke refresh token

[ ] 3. Implement RBAC middleware trong post-service
        Public: GET /api/posts, GET /api/posts/:id
        RequireRole(author, admin): POST /api/posts
        RequireOwnership + RequireRole: PUT /api/posts/:id
        RequireRole(admin): DELETE /api/posts/:id

[ ] 4. Cấu hình Docker Secrets cho jwt_private_key và db_password

[ ] 5. Chạy security checks:
        govulncheck ./...        → 0 lỗ hổng
        go vet ./...             → 0 cảnh báo
        gitleaks detect .        → 0 secret bị leak

[ ] 6. Test RBAC thủ công:
        # Đăng ký với role reader
        TOKEN=$(curl -s -X POST localhost/api/auth/login -d '...' | jq -r .access_token)
        # Thử tạo bài (phải fail với 403)
        curl -X POST localhost/api/posts -H "Authorization: Bearer $TOKEN" -d '{...}'
        → 403 Forbidden

        # Đăng ký với role author, thử lại
        → 201 Created

[ ] 7. Xác minh security headers:
        curl -I https://localhost/api/posts
        → X-Frame-Options: DENY
        → X-Content-Type-Options: nosniff
        → Strict-Transport-Security: max-age=31536000
```

---

## 6.10 Tài Nguyên

| Tài nguyên | URL |
|---|---|
| JWT.io debugger | https://jwt.io |
| OWASP Top 10 | https://owasp.org/www-project-top-ten/ |
| golang-jwt library | https://github.com/golang-jwt/jwt |
| govulncheck | https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck |
| gitleaks | https://github.com/gitleaks/gitleaks |
| HashiCorp Vault intro | https://developer.hashicorp.com/vault/tutorials |

---

## 6.11 Tự Kiểm Tra — Phase 6 Hoàn Thành Khi…

```
[ ] Có thể giải thích tại sao RS256 tốt hơn HS256 trong microservices
[ ] JWT access token hết hạn sau ≤ 15 phút
[ ] Refresh token là httpOnly cookie (JS không đọc được)
[ ] Password hash bằng bcrypt cost 12 (KHÔNG phải SHA hay MD5)
[ ] RBAC: reader không tạo được bài, không phải chủ không sửa được bài của người khác
[ ] govulncheck ./... → 0 lỗ hổng
[ ] gitleaks scan → không tìm thấy secret trong git history
[ ] Security headers có trên tất cả response (X-Frame-Options, CSP, HSTS)
[ ] Không có thông tin nhạy cảm trong bất kỳ dòng log nào
```

---

*← [Phase 5 — API Gateway](./phase_05_api_gateway.md) | [Phase 7 — Fullstack →](./phase_07_fullstack.md)*
