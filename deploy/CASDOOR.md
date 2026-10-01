# Casdoor 统一登录部署手册(store + sub2api 账号互通)

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

1. `/opt/casdoor/docker-compose.yml`(独立于 dujiao-next 的 compose,避免互相牵连):

   ```yaml
   services:
     casdoor:
       image: casbin/casdoor:latest   # 部署时锁定具体版本 tag
       restart: unless-stopped
       ports:
         - "127.0.0.1:8000:8000"
       environment:
         origin: "https://auth.kanoapi.top"
         runmode: "prod"
         driverName: "sqlite"
         dbName: "casdoor"
         dataSourceName: "file:casdoor.db?_busy_timeout=5000"
         signupItem: "[]"
       volumes:
         - ./data:/conf
   ```

   (SQLite 数据落在 `/opt/casdoor/data/casdoor.db`;参数以所选版本的官方 compose 为准,此处为最小形态。)

2. host nginx 新站点 `auth.kanoapi.top` 反代 `127.0.0.1:8000`(client_max_body_size 适度放宽,WebSocket 升级头按官方示例),`certbot --nginx -d auth.kanoapi.top` 签证书。
3. 初始化:浏览器打开站点,`admin/123` 首登**立即改密**并启用 2FA。
4. 验证:`curl -s https://auth.kanoapi.top/.well-known/openid-configuration | jq .` 能返回三端点。

## Casdoor 配置

1. 组织:默认 `built-in` 或新建 `kano`。
2. 修改身份源应用或新建应用 `dujiao-store`:
   - Redirect URL:`https://store.kanoapi.top/auth/oidc/callback`
   - Client ID / Client Secret 记录到密码管理器。
   - Token 签名算法保持 RS256(dujiao 侧仅验 RS256)。
3. sub2api 侧应用:sub2api 后台"第三方登录 → OIDC"里通常自带 client(或按其文档创建),Redirect URL 填 `https://api.kanoapi.top/login/oauth/oidc`(以其后台提示为准)。
4. 登录页品牌:应用 → Login UI(背景图 URL / Form CSS / 侧边面板 HTML / 主题主色圆角),见官方 Login UI customization。

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
