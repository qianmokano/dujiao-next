# Casdoor 统一登录部署手册(store + sub2api 账号互通)

## 当前流程（2026-10-07）

商城使用独立 `sso_auth` 设置：`enabled`、`only_enabled`、`issuer`、`organization`、`application_id`、`display_name`；后台 API 为 `/api/v1/admin/settings/sso-auth`。网关使用独立的 `sso_issuer_url`、组织、应用及三个策略开关，参见网关仓库的 `deploy/KANO-SSO.md`。

两个站点均通过 `/api/v1/auth/sso/password-login`、`/sso/mfa`、`/sso/register/send-code`、`/sso/register` 在页内完成认证。旧 OIDC 跳转、回调和绑定入口已移除；新配置无需 OAuth 客户端、秘密或回调 URI。一次性迁移保留旧行，明确的 false 和空值优先，完成后只读新设置。原身份的 provider、subject 和网关 issuer 保持不变，原账号及业务数据继续关联。

`/api/v1/auth/sso/captcha` 读取配置应用的 CAPTCHA，支持 Casdoor Default 图片及 Cloudflare Turnstile。挑战在 Redis 保存五分钟，绑定配置、动作和账号，提交前原子消费；上游验证码与本站验证码分别验证。上线不修改 Casdoor 当前 CAPTCHA 策略。

按 `deploy/README.md` 发布固定镜像，先更新网关再更新商城。切换前备份完整配置、SQLite 数据目录、上传目录和 Redis，并实际恢复到隔离目录及私有端口验证。回退须同时恢复旧镜像和更新前数据。

## 首次 OIDC 部署历史（以下配置和验收流程已过时）

日期:2026-10-01。对应代码:分支 `kano/generic-oidc-login`(dujiao 侧通用 OIDC 登录)。
背景与方案对比见 [ANALYSIS-2026-10-01-dujiao-sub2api-account-interop.md](ANALYSIS-2026-10-01-dujiao-sub2api-account-interop.md)。

本文不含任何密钥;`<占位>` 在执行时填充。

## 架构

- `auth.kanoapi.top` → Casdoor(VPS,Docker,SQLite),作为唯一 OIDC 身份源。
- `store.kanoapi.top`(dujiao):后台"单点登录"设置填 Casdoor 应用信息;登录页出现"统一登录"按钮。
- `api.kanoapi.top`(sub2api,另一台服务器):后台开启 OIDC 登录,指向同一 Casdoor 应用体系。
- 账号规则:Casdoor 下发的邮箱即账号锚点。dujiao 首登按邮箱自动关联/建号;sub2api 的 OIDC 登录本就按邮箱匹配既有账号,余额/Key 原地保留。

## 前置检查

1. **sub2api 版本 ≥ v0.1.171 之后的安全修复版**(OAuth 账户接管漏洞),在 api.kanoapi.top 后台或镜像 tag 确认;不满足先升级。
2. Cloudflare 为 `auth.kanoapi.top` 添加 A 记录指向 VPS(仅 DNS 或橙云均可;橙云需放行 OIDC 回调路径)。
3. VPS 建议先加 ~2G swap 兜底(1.9G 内存无 swap):
   `sudo fallocate -l 2G /swapfile && sudo chmod 600 /swapfile && sudo mkswap /swapfile && sudo swapon /swapfile`(并写入 /etc/fstab)。

## VPS 部署 Casdoor

1. `/opt/casdoor/`(独立于 dujiao-next 的 compose,避免互相牵连)。**已在 2026-10-01 按 此配置部署跑通**,要点:

   - 镜像 `casbin/casdoor:4.13.0`(Docker Hub tag **无 v 前缀**,`v4.13.0` 不存在;amd64/arm64 双架构)。
   - 配置经挂载 `/conf/app.conf`(整文件,非 env),关键项:`driverName = sqlite`(**必须是不带 3 的 sqlite**,modernc 纯 Go 驱动;官方镜像没编译 CGO 的 sqlite3,写 sqlite3 会 panic "unknown driver")、`dataSourceName = file:/data/casdoor.db?_busy_timeout=5000`、`origin = "https://auth.kanoapi.top"`、`runmode = prod`、`defaultLanguage = "zh"`。
   - 容器以 **USER 1000** 运行:挂载的 `/opt/casdoor/data`、`/opt/casdoor/logs` 需 `chown 1000:1000`,`conf/app.conf` 需 644(600 会 permission denied panic)。
   - compose 端口 `127.0.0.1:8000:8000`;volumes:`./conf:/conf`、`./data:/data`、`./logs:/logs`。
   - 验证:`curl -s http://127.0.0.1:8000/.well-known/openid-configuration` 返回 issuer 与三端点(issuer 应为 auth.kanoapi.top)。

2. host nginx 新站点 `auth.kanoapi.top` 反代 `127.0.0.1:8000`(client_max_body_size 适度放宽,WebSocket 升级头按官方示例),`certbot --nginx -d auth.kanoapi.top` 签证书。
3. 初始化:浏览器打开站点,`admin/123` 首登**立即改密**并启用 2FA。
4. 验证:`curl -s https://auth.kanoapi.top/.well-known/openid-configuration | jq .` 能返回三端点。

## Casdoor 配置(2026-10-01 已完成)

已用 API 完成初始化(脚本 `/tmp/casdoor-init2.py`、`/tmp/casdoor-addapp3.py` 思路):

- **admin 默认密码已改**(初始 `admin/123`,新密码在部署会话中交付,请存密码管理器并尽快在后台开 2FA)。
- 已建应用 `dujiao-store`(owner=admin,organization=built-in):Redirect URI `https://store.kanoapi.top/auth/oidc/callback`,tokenFormat=JWT(RS256),grantTypes=authorization_code,token 有效 7 天。

API 自动化的坑(重装时参考):

1. `/api/login` body 必须带 `"type":"login"`;响应会发**两个** Set-Cookie(轮换),以最后一个 `casdoor_session_id` 为准(Python `http.cookiejar` 对 IP 主机会拒收,需手动取头)。
2. **改密码必须用 `POST /api/set-password`**(form:userOwner/userName/oldPassword/newPassword);`/api/update-user` 的默认列白名单**不含 password 列**,返回 ok 但密码纹丝不动。
3. `/api/add-application` 用完整克隆 app-built-in 的对象会静默失败(返回 ok + "Unaffected",Insert 错误被源码吞掉);用**最小字段**(owner/name/displayName/organization/redirectUris/tokenFormat/grantTypes)创建,再按需 update。
4. 验证:`GET /login/oauth/authorize?client_id=...&redirect_uri=...&response_type=code&scope=openid+profile+email&state=x&code_challenge=<S256>&code_challenge_method=S256` 应返回 200 登录页。

sub2api 侧应用:在其后台"第三方登录 → OIDC"创建/启用,Redirect URL 按其后台提示填(形如 `https://api.kanoapi.top/login/oauth/oidc`),issuer 同为 `https://auth.kanoapi.top`。

登录页品牌:应用 → Login UI(背景图 URL / Form CSS / 侧边面板 HTML / 主题主色圆角),见官方 Login UI customization。

## dujiao 侧配置(镜像发布后)

后台 → 设置 → 单点登录(OIDC):

- 启用:开
- Issuer:`https://auth.kanoapi.top`
- Client ID / Client Secret:Casdoor 应用凭据
- 回调地址:`https://store.kanoapi.top/auth/oidc/callback`(须与 Casdoor 应用完全一致)
- 按钮文案:如"统一登录"

验证:登录页出现按钮 → 跳转 Casdoor → 用测试账号登录 → 回调后进入商店并正确建号;后台登录日志 source=oidc。

## 存量用户导入(sub2api → Casdoor)

从 sub2api 的 Postgres 导出 `users`(email、password_hash=bcrypt、status),经 Casdoor API 导入,密码哈希原样迁移(用户无感):

```bash
# 1) 在 sub2api 所在机器导出(用户数少可直接 psql)
psql "$SUB2API_DB_URL" -At -c "SELECT email || '|' || password_hash FROM users WHERE status='active'" > /tmp/s2a_users.txt

# 2) 导入脚本骨架(执行时按 Casdoor API token 方式补全认证;passwordType=bcrypt)
#    POST /api/add-user  {"owner":"<org>","name":"<唯一用户名>","email":"<email>",
#                         "password":"<bcrypt哈希>","passwordType":"bcrypt",
#                         "displayName":"<email前缀>","type":"normal-user"}
```

注意:导出后到切换前的增量注册/改密需二次执行(幂等:先查后插);开过 TOTP 的少量用户在 Casdoor 重绑。

## 备份与回退

- 备份对象新增:`/opt/casdoor/data/`(SQLite + attachments)。纳入现有备份节奏,VPS 外留存。
- 回退 dujiao:后台关闭"单点登录"即可,账号数据无损(外部身份绑定表保留)。
- 回退 sub2api:后台关闭 OIDC 登录,密码登录不受影响(sub2api 本地密码登录无禁用开关,天然双轨)。

## 风险与运维注意

- Casdoor 成为认证单点:宕机则新登录不可用(已签发的 dujiao JWT 在有效期内不受影响;sub2api 可密码直登兜底)。纳入健康监控(公开的 /.well-known 探测)。
- Casdoor 版本跟随:升级前看 release note,镜像 tag 锁定版本,勿用 latest 盲更。
- dujiao 登录信任 IdP 邮箱声明(同邮箱自动关联)。不要将此 OIDC 配置指向不受信的第三方 IdP。
