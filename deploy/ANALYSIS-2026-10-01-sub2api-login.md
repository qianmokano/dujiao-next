# 接入 sub2api 账号登录(第三方登录)可行性

日期:2026-10-01。仅分析,未修改代码。

## 结论

- **可以接入,但不是开箱即用**:sub2api 不提供 OAuth2/OIDC Server(无 `/authorize`、`/token`、`/userinfo` 等对外身份端点),其文档中的 "OAuth" 均指接入上游 AI 账号,与对外登录无关。因此不存在标准的"用 sub2api 账号登录第三方应用"协议路径。
- dujiao-next 这**已具备**成熟的第三方登录扩展模式:现有 Google 与 Telegram 两种自研 provider(无第三方 OAuth 库),有 provider 无关的绑定表 `user_oauth_identities`,登录收尾 `completeExternalLogin`、后台设置热更新、公开配置下发、前端登录开关等基础设施完整。新增一种登录方式 = 照现有切片模式新写一个垂直切片,无需数据库迁移。
- **推荐方案 A(只改 dujiao,不动 sub2api)**:在 dujiao 新增 "sub2api" 登录方式,用户输入 sub2api 的邮箱+密码,dujiao 后端代理调用 sub2api `POST /api/v1/auth/login`,成功后用返回的 JWT 调 `GET /api/v1/auth/me` 取用户 id/邮箱,再 find-or-create 本地账号并写入 `user_oauth_identities(provider='sub2api')`,最后走 `completeExternalLogin` 签发 dujiao 自己的 JWT。密码仅用于一次代理调用,不落库。
- 备选方案 B(更标准但更重):部署独立统一身份源(如 Casdoor/Logto),dujiao 复用现有 OIDC client 模式接入;但 sub2api 自身不支持社交/OIDC 登录,要走这条路必须 fork sub2api 加 OIDC client 或接受"sub2api 账号不再是唯一账号源",维护成本高,暂不推荐。

## 依据

### dujiao-next 现状(代码调查)

- 登录路由:`internal/app/httpserver/routes_storefront.go:103-112`(`/api/v1/auth` 下,Google/Telegram 均在此挂载);JWT(HS256)机制:`internal/modules/identity/userauth/application/service.go:153 GenerateUserJWT`,中间件 `internal/app/httpserver/middleware/middleware.go:321`。
- 外部身份绑定表(通用于任意 provider):`internal/modules/identity/externalidentity/domain/identity.go:7`(`Provider` 为 varchar,唯一键 `(provider, provider_user_id)`),无需迁移。
- 可照抄的切片模板:
  - Telegram OIDC(完整 OAuth2 code+PKCE client):`internal/modules/identity/telegramauth/application/oidc.go` + `internal/modules/identity/userauth/application/telegram_oidc.go`;
  - Google(验签式):`internal/modules/identity/googleauth/application/service.go` + `userauth/application/google.go`(含 `completeExternalLogin:581`、绑定/解绑、`isUsableExternalIdentity:550`)。
- 设置与前端开关链路:`internal/modules/settings/schema/security/{google,telegram}_auth.go`、`settings/transport/http/routes.go:22-28`、公开配置 `internal/bootstrap/publicconfig/wiring.go`、前端 `frontend/user/src/composables/useLogin.ts:75-112` 与 `views/auth/Login.vue`、admin `Settings.vue`。
- DI 接线点:`internal/app/container/services_foundation.go:110-113`、`internal/bootstrap/userauth/wiring.go:23`。

### sub2api 现状(仓库与文档调查,Wei-Shaw/sub2api)

- 自身用户体系:邮箱+密码登录,`POST /api/v1/auth/login`(body `{"email","password"}`)返回 `data.access_token`(JWT);`GET /api/v1/auth/me`(Bearer JWT)返回用户信息;支持 TOTP 两步验证与 Cloudflare Turnstile 人机验证(可配置);另有 refresh 接口与 Admin API Key。
- README 与 deploy/README.md 均无对外 OAuth2/OIDC Server、也无以第三方 IdP 作为其用户登录方式的说明;「External System Integration」是后台 iframe 嵌入外部系统,不是身份开放。

## 方案 A 的注意事项(实现时要处理)

1. **2FA**:sub2api 用户若开启 TOTP,登录接口会要求二次验证,dujiao 需中继一次验证码输入(类似本项目 2FA 挑战 token 的做法)。
2. **Turnstile**:若 sub2api 对登录开了人机验证,服务器代理调用会被挡;需在 sub2api 侧关闭或对 dujiao 后端来源放行。
3. **网络**:dujiao 后端需能访问 sub2api 地址(当前 VPS 与 sub2api 是否同机/互通需确认)。
4. **邮箱冲突**:首次登录 find-or-create 时,若本地已存在同邮箱账号,应要求先登录本地账号再绑定(现有 Google/Telegram 绑定流程已有同类处理)。
5. **安全提示**:UI 需明示"将使用你的 sub2api 邮箱密码验证";密码只在后端内存中转一次,不存储、不写日志。
6. **账号生命周期**:sub2api 侧改密/停用后,dujiao 侧登录随之失效(符合预期);解绑保护需在 `isUsableExternalIdentity` 中加 provider case。

## 下一步

- 确认 sub2api 实例地址、是否开启 Turnstile/2FA,以及 VPS 上 dujiao 后端到 sub2api 的网络可达性。
- 如决定实施,按 Telegram OIDC 切片结构开功能分支:`sub2apiauth` 模块(密码代理 + `/auth/me`)+ `userauth/application/sub2api_login.go` + 设置项(`base_url` 等)+ 前端登录入口与绑定管理,预计为中等规模改动。
