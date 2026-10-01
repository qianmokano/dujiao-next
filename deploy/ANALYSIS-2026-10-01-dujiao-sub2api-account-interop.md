# dujiao(store.kanoapi.top)与 sub2api(api.kanoapi.top)账户互联方案对比

日期:2026-10-01。仅分析,未修改代码。前篇:[ANALYSIS-2026-09-30 无 / 2026-10-01 sub2api 登录可行性](ANALYSIS-2026-10-01-sub2api-login.md)。

## 结论(方案总览)

| 方案 | 思路 | 开发量 | 难度 | 上线周期 |
|---|---|---|---|---|
| 1a 共享 Google 登录 | 两边各自开启 Google OAuth,同一 Google 账号 = 同一个人 | **dujiao 零开发**(已内置);sub2api 纯后台配置 | ★ | 半天~1 天 |
| 1b/1c 共享 GitHub / LinuxDO 登录 | 同上,但 dujiao 需新增 provider 切片 | dujiao 每种 2-3 天;sub2api 纯配置 | ★★ | 3-5 天/种 |
| 2 统一身份源(IdP) | 部署 Logto/Casdoor 等,两站都作为 OIDC client | dujiao 通用 OIDC 切片 3-5 天;sub2api 纯配置;IdP 部署+存量账号迁移 | ★★★ | 1-2 周+长期运维 |
| 3 密码代理(sub2api 为账号源) | dujiao 登录页代理验证 sub2api 邮箱密码 | dujiao 2-3 天 | ★★ | 约 1 周 |
| 4 fork sub2api 加 OAuth Server(store 为账号源) | 改 sub2api 信任 store 身份 | sub2api 侧 Go+Vue 改造+长期跟上游合并 | ★★★★ | 2 周+,不推荐 |
| 5 业务互联:购买自动交付 | 支付成功后调 sub2api Admin API 充值/开通 | dujiao 交付器 2-4 天 | ★★ | 约 1 周 |

- **推荐路线**:先上 1a(Google 共享登录)+ 5(自动交付),约一周得到"同一账户体感 + 购买自动到账";客群若以 LinuxDO/GitHub 为主再补 1b/1c;将来需要真·单点登录或接入第三个服务时再上方案 2。
- 方案 1 与 3/4 的本质区别:1 是"两边各自登录、身份同源",不是 SSO;2 才是统一账号库+单点登录。
- **安全前置**:sub2api ≤ v0.1.171 存在 OAuth 账户接管漏洞(CVSS 8.8,仅凭邮箱即可接管,2026-08 披露)。启用任何第三方登录前,必须确认 api.kanoapi.top 已升级到修复版本。

## 依据

### dujiao-next(本仓库)

- 已内置 Google + Telegram 两种第三方登录,自研切片模式,绑定表 `user_oauth_identities` provider 无关(见前篇分析)。
- Google 登录仅剩后台配置工作:`google_auth` 设置(`enabled`、`client_id`)已在 admin Settings 存在。
- 新增 provider 的模板:Telegram OIDC 切片(`internal/modules/identity/telegramauth/application/oidc.go` 是完整 OAuth2 code+PKCE client),泛化为可配置 issuer 的通用 OIDC 切片即方案 2 所需。

### sub2api(api.kanoapi.top 线上实例,2026-10-01 探测其公开前端资源)

- 从线上前端主 JS(/assets/index-*.js)确认,登录方式支持:`email、linuxdo、oidc、wechat、github、google、dingtalk` 七种 + passkey;相关设置项 `oidc_oauth_enabled / github_oauth_enabled / google_oauth_enabled / wechat_oauth_* / passkey_enabled`;登录发起端点 `/api/v1/auth/oauth/{provider}/start`,回调视图 OAuthCallbackView / LinuxDoCallbackView / OidcCallbackView / WechatCallbackView 等齐备。
- **支持通用 OIDC client**(可接任意 OIDC 身份源)是方案 2 难度大幅下降的关键。
- Admin API(官方 skill 文档):用户管理(创建/封禁/角色)、API Key 管理、订阅管理、`redeem-codes create-and-redeem`(官方注明"用于支付回调或人工充值",支持 `idempotency-key`)。这是方案 5 的官方支持路径。
- 自身登录 API:`POST /api/v1/auth/login`(邮箱密码→JWT)、`GET /api/v1/auth/me`、refresh;支持 TOTP、Turnstile(均可在后台配置)。方案 3 的接口基础。
- `.env.example` 无第三方登录的 env;登录方式配置在后台设置中。

### 漏洞通告

- sub2api v0.1.171 及更早:OAuth pending-session exchange 缺陷,仅凭注册邮箱即可完成第三方登录绑定并接管账户(控制 API Key 与配额)。来源:linux.do / NodeSeek 2026-08 安全通告。

### 密码哈希迁移能力(2026-10-01 修正)

本文件早先把方案 2 的存量迁移成本写作"密码哈希不可逆,只能邀请重置",**该结论有误,特此修正**:

- 哈希不可逆 ≠ 哈希不能搬家。迁移哈希不需要还原明文,只需要目标 IdP 能用**相同算法**验证同一哈希。
- sub2api 用 bcrypt:`backend/internal/service/user.go:100`(GenerateFromPassword, DefaultCost)、`auth_service.go:1464`;哈希为标准 `$2a$` 自含盐格式。
- dujiao-next 同样用 bcrypt:`internal/modules/identity/userauth/application/service.go:292/352`。
- Logto:Management API 建用户支持带哈希导入,`passwordAlgorithm` 支持 Bcrypt(及 Argon2、Legacy 盐式 MD5/SHA);官方文档有 User migration 批量迁移指引。
- Casdoor:用户字段 `passwordType` 支持 `bcrypt`(新组织默认即 bcrypt)/`argon2`/`md5-salt` 等,`cred/bcrypt.go` 直接按标准 bcrypt 验证。
- 结论:两边存量密码哈希都能**无感导入** IdP,用户不需要因为"哈希搬不动"而重置密码。
- 迁移的**真实**成本与风险(为什么仍不是零成本):
  1. **账号合并只能保一个哈希**:同一邮箱在 dujiao 与 sub2api 是两个账号、两个可能不同的密码;合并为一个 IdP 账号时只能保留其中一个 bcrypt 哈希,密码设成另一个值的用户仍需一次重置。且两边 salt 不同,离线无法判断"两边密码是否相同"(可在切换时提示用户"沿用商店密码/沿用 API 站密码"来缓解,或者干脆按注册时间保留较新一边)。
  2. **切换窗口的增量变更**:导出后用户改密/新注册需要停写窗口或做二次差异同步。
  3. **附属数据**:TOTP 密钥通常拿不到明文,2FA 用户需重绑;封禁状态需映射。
  4. 实施细节:验证 `$2a$`/`$2b$` 前缀兼容(Go 生成 `$2a$`,主流实现均识别);保留原 cost 参数;导入后 IdP 是否支持登录时透明升级哈希算法(Logto 新密码用 Argon2)。
- 若目标 IdP 不支持源哈希算法,还有"懒迁移"兜底:登录时回验旧系统(sub2api `/auth/login`),成功即按 IdP 格式重新落库——Logto 无原生支持,Keycloak 有用户联邦;本场景两边皆 bcrypt,用不上。

### 范围更新(2026-10-01:确认 dujiao 无存量用户,只迁 sub2api)

合并双哈希的问题不存在了,方案 2 难度由 ★★★ 下调为 **★★☆**,迁移范围缩到最小。sub2api 源码(/tmp 克隆)核实的关键事实:

- **sub2api 侧零迁移**:`backend/internal/service/auth_service.go:714` 起,OAuth/OIDC 登录先 `GetByEmail` 匹配既有账号,匹配到即登录原账号(余额/Key/订阅不变);无账号时"OAuth 首次登录视为注册"(受注册开关与邀请码设置约束,且 OAuth 可配置绕过关闭的注册)。用户从 IdP 过来只要邮箱相同,自动落回原账号。
- **唯一要做的迁移**:把 sub2api 的 `users` 表(`email`、`password_hash` 即 bcrypt、`status`)导出,经 Logto Management API(`passwordAlgorithm=Bcrypt`)或 Casdoor(`passwordType=bcrypt`)导入 IdP,让用户在 IdP 登录页沿用原密码。脚本级工作量(半天~1 天)。
- **TOTP 迁不了**:`users.totp_secret_encrypted` 加密存储,开过 2FA 的用户需在 IdP 重绑(人数通常极少)。
- **sub2api 本地密码登录无禁用开关**:设置键只有 `RegistrationEnabled`、各 OAuth provider 的 `*_oauth_enabled` 等,没有"关闭邮箱密码登录"。因此 sub2api 长期处于双轨(本地密码 + OIDC 都能进)。影响:用户若直接在 sub2api 改密,会与 IdP 密码分叉,但 OIDC 登录按邮箱匹配、不校验本地密码,SSO 不受影响;且 IdP 故障时 sub2api 仍可直接密码登录,算韧性冗余。dujiao 侧则只信 IdP,单一入口。
- dujiao 无存量用户,通用 OIDC 切片上线后首登自动建号,零迁移。

修订后排期:IdP 部署 1 天 + dujiao 通用 OIDC 切片 3-5 天 + 导入脚本与联调 1-2 天,合计约 1 周;无用户侧中断(旧密码登录在 sub2api 一直可用)。

### IdP 选型与 VPS 资源(2026-10-01 实测)

生产 VPS(159.195.124.212,store.kanoapi.top 所在机)实测:2 核 / 1.9Gi 内存(用 461Mi,可用 ~1.5Gi)/ **无 swap** / 磁盘 58G 仅用 2.5G / 负载空载。dujiao 全家桶驻留约 350MB(dockerd 124M、dujiao 应用 79M、containerd 70M、Redis 14M、nginx 2×14M 等)。已监听端口仅 80/443/8080/1126;**本机没有 sub2api 与 Postgres**(sub2api 在另一台服务器,IdP 需经公网 HTTPS 被 api.kanoapi.top 回调,无碍)。

两个 IdP 的典型占用(经验值,正式选型前建议在 VPS 试跑 10 分钟用 docker stats 实测):

- **Casdoor**:单个 Go 二进制容器,空闲 ~50-100MB;可用 SQLite 落盘,**零额外依赖**(不需要新增数据库/缓存容器);磁盘镜像 ~150-250MB。→ 本机可轻松承载。
- **Logto**:Node.js 核心 ~200-350MB + 必需 PostgreSQL ~100-150MB(Redis 可选、可复用现有),合计 ~350-500MB;镜像+依赖约 1-2GB。→ 1.9G 无 swap 机器上会吃掉 1/4~1/3 内存余量,突发时有 OOM 被内核杀进程的风险(可能殃及 dujiao)。

结论:**这台 VPS 推荐 Casdoor**(SQLite 单容器,新增内存 ~100MB);若偏好 Logto,要么先加 2G swap 并接受更厚重的运维(多一个 Postgres 要备份),要么放到另一台 1Panel 服务器上(IdP 无需与 dujiao 同机)。顺带建议:无论选哪个,这台 1.9G 无 swap 的机器都应加一个 ~2G 的 swap 文件兜底。部署形态:host nginx 新增 auth.kanoapi.top 站点反代 127.0.0.1:8000(Casdoor 默认端口,与 8080 不冲突),certbot 签证书,SQLite 数据文件与 attachments 目录纳入现有备份流程,Cloudflare 加 DNS 记录。

#### Casdoor vs Logto 对比(2026-10-01)

| 维度 | Casdoor | Logto |
|---|---|---|
| 出身/技术栈 | Casbin 团队,Go 单二进制 | AVA 团队,TypeScript/Node.js |
| 依赖 | 无强制依赖(SQLite 即可;也支持 MySQL/PG 等) | 强制 PostgreSQL 14+,Redis 可选 |
| 内存(上节实测口径) | ~50-100M | ~400-500M(含 PG) |
| 协议广度 | OAuth2/OIDC/SAML/CAS/LDAP 等,偏"全家桶" | 聚焦 OIDC/OAuth2(基于 node-oidc-provider,规范实现) |
| 登录页 | 按应用配置:背景图 URL、Form CSS、面板左/中/右、可开侧边面板写任意 HTML;theme 层调主色/圆角,后台实时预览(官方 Login UI customization 文档) | 现代 hosted 登录页,可深度定制,体验与移动端明显更好 |
| 社交登录 | 40+ 内置 provider,含微信 | 40+ 连接器,另有通用 OIDC 连接器(可接 LinuxDO 等) |
| 密码哈希导入 | `passwordType=bcrypt` 逐用户导入 | Management API `passwordAlgorithm=Bcrypt` |
| MFA | TOTP/邮件/短信/WebAuthn | TOTP/短信/邮件/Passkey |
| 权限模型 | 内置 Casbin(RBAC/ABAC),与 dujiao 同源思路 | 组织+角色,偏应用侧授权 |
| 备份 | 一个 SQLite 文件 + attachments 目录 | pg_dump 整库 |
| 升级 | 换镜像,xorm 自动迁移 | 换镜像,自动迁移,链路更长 |
| License | Apache-2.0 | MPL-2.0(云版另收费) |
| 文档/中文 | 中英文档,国内社区活跃 | 中英文档质量高,迭代快 |

对本场景的判断:两者在"给 dujiao+sub2api 当 OIDC 身份源 + bcrypt 导入"这个核心需求上**能力等价**,差异不在功能而在**资源与运维面 vs 产品体验**。1.9G 无 swap 的单机、单人运维、用户量小 → Casdoor(少一个常驻数据库、备份即复制文件);若未来用户量上来、或 IdP 挪到资源宽裕的另一台机器、且更看重登录页观感 → Logto 同样合理。Keycloak/Authentik 对这台机器过重,不在候选;Zitadel(Go)是折中项但同样要外置 Postgres,相对 Casdoor 无决定性优势。

## 各方案要点与风险

1. **1a/1b/1c 共享社交登录**:dujiao 已有 Google(`components/auth/GoogleIdentityButton.vue` 等),sub2api 内置各 provider;两边用同一 OAuth App 体系,同一外部账号在两边登录后,靠同邮箱关联为"同一人"。风险:非 SSO(两边各自会话);邮箱密码注册的存量用户不受益;Google 依赖海外账号,LinuxDO 在 AI 中转客群渗透率高。
2. **统一 IdP**:sub2api 后台配 OIDC 即接;dujiao 做通用 OIDC 切片。真 SSO、账号库唯一。存量密码**哈希可直接导入**(两边都用 bcrypt,Logto/Casdoor 均支持 bcrypt 哈希导入,见下方"密码哈希迁移能力"),真正的成本在**账号合并与切换窗口**(见下),而非哈希不可逆;IdP 成为单点,需纳入备份监控;多一个常驻组件。
3. **密码代理**:仅改 dujiao,sub2api 无感知。风险:用户密码流经 dujiao(需明示);sub2api 开启 TOTP/Turnstile 时需中继或放行;改密只在 sub2api 生效(符合预期)。
4. **fork sub2api**:能实现"store 为唯一账号源",但 sub2api 迭代与安全补丁频繁,fork 合并成本长期化,不推荐。
5. **自动交付**:dujiao 订单支付成功 → asynq 任务调 `redeem-codes create-and-redeem`(`idempotency-key` 用订单号,天然幂等重试)→ 按买家邮箱定位 sub2api 账号充值/开通;账号不存在时降级为发兑换码(dujiao 发卡本来就能交付文本)。Admin API Key 存放于 dujiao 侧加密配置,不入 Git。

## 下一步

1. 确认 api.kanoapi.top 的 sub2api 版本 ≥ 修复版本(后台或镜像 tag 查看),不足先升级。
2. 确认目标客群主力登录方式(Google / GitHub / LinuxDO / 微信),决定先做 1a 还是 1b/1c。
3. 如采纳推荐路线:建 Google OAuth App(store、sub2api 各配回调)→ 两边后台开启 → 验证同账号两边登录;随后在 dujiao 开功能分支实现方案 5 交付器。
4. 若走方案 2:先用 Logto Management API(或 Casdoor add-user)试导入若干测试账号的 bcrypt 哈希验证登录,再定账号合并规则(保留哪边密码)与切换窗口流程。
