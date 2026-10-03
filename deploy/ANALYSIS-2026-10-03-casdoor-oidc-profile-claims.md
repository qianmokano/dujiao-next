# Casdoor OIDC 资料字段兼容修复

日期：2026-10-03。从已通过 CI 的 fork main `3a9cefcb` 创建 `kano/casdoor-oidc-profile-claims`。延续两站账户面板任务的上线授权；保持所有预存未提交文件及 Casdoor 仓库原样。本记录不包含凭据或令牌。

## 生产验收发现

商城 `v0.1.0-kano.15` 与网关 `v0.2.11-kano.5` 均已健康上线并完成备份、实际恢复和隔离业务数据比较。测试账户的业务指纹与上线前一致，但商城在页内密码登录后再执行 OIDC 回调，昵称由通行证显示名变成了用户名。测试尚未修改 Casdoor 昵称或头像，失败在修改前即检出。

## 原因与证据

- Casdoor `object/token_jwt.go` 的默认 `Claims`、`UserShort` 和 `UserWithoutThirdIdp` 以 `name` 表示用户名，并以 `displayName`、`avatar`、`emailVerified` 表示资料。其 Standard 格式使用 `name`、`preferred_username`、`picture`、`email_verified`。
- 商城页内登录的 `getAccount` 正确读取 Casdoor 格式；OIDC 回调只解析标准字段，因而把默认格式中的用户名当作显示名。
- 参考现有标准 OIDC 验签／资料读取、Casdoor 页内账户读取，以及网关已有资料同步实现。修复集中在已验签 OIDC 声明到内部身份的转换，不改变 subject、邮箱、绑定、令牌签发、MFA 或持久化结构。

## 修复与验证

补充 Casdoor 默认字段兼容，标准 `picture` 与 `email_verified` 有值时优先（包括明确清空和 false）；仅缺失或 null 时使用 Casdoor 对应字段。Casdoor 明确空显示名交给既有用户名／邮箱前缀回退。头像保持缺失、null、明确清空的区分，无效地址仍由原有同步层处理。

增加实际 RS256 签名、discovery、PKCE、JWKS 验签与回调测试，在标准格式成功后再次回调 Casdoor 格式，确认显示名、用户名、头像和邮箱验证读取一致。另覆盖 12 组字段格式、优先级及缺失／null／空值场景。

本地 `go test ./...`、`go vet ./...` 与服务端构建通过；针对性 OIDC 与真实 SQLite 认证集成测试通过。新增声明转换方法语句覆盖率 100%。标准格式与 Casdoor 默认格式均通过真实签名／回调测试。

创建修复 PR 后等待完整 CI；合并后再次确认 main CI，再发布未使用的新标签，预拉取并执行新的完整备份／实际恢复／隔离比较后更新商城。CI、修复发布和生产资料同步验收结果记录在本次发布记录中；保留已发布的 `.15` 标签及备份记录。
