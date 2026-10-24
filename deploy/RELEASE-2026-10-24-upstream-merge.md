# 发布记录：v0.1.0-1（2026-10-24）

## 概要

- 内容：合入上游 dujiao-next/dujiao-next main 的 60 个提交（2026-09-16 至 2026-10-11），
  含邀请码注册、采购单/佣金修复、安全加固（SSRF/XSS/鉴权缓存）、注册限流独立等。
  冲突处理详见 [ANALYSIS-2026-10-24-upstream-merge.md](ANALYSIS-2026-10-24-upstream-merge.md)。
- 流程：PR #23（分支 kano/merge-upstream-20261024）→ ci 通过 → 合并至 main（0a1b19d4）
  → main push ci 通过 → 标签 `v0.1.0-1` → Publish fork image 成功。
- 命名：首个采用 PR #22 规则的 `vX.Y.Z-N` 标签（旧 `-kano.N` 标签保留原名）。
- 镜像：`ghcr.io/qianmokano/dujiao-next:v0.1.0-1`（amd64/arm64）。
- 中途插曲：fork 仓库 GitHub Actions 当日对所有事件不触发，用户恢复后 close/reopen
  PR 补触发；首次 ci 因 sso_policy_test.go gofmt 未过失败，已修复后通过。

## VPS 更新步骤

已于 2026-10-24 19:03 (UTC+8) 执行完毕，线上现为 `v0.1.0-1`：

- 使用 `deploy/update-image.sh`（scp 至 VPS /tmp 后以 sudo 运行）。
- 备份：`backups/predeploy-20261024T110345Z`（dujiao-files.tar.gz 4.4MB + redis-data.tar.gz，校验通过）。
- 回滚参考：更新前镜像为 `v0.1.0-kano.19`（`.env.bak.20261024190340`）；
  若需回退且数据库已迁移，须连同上述备份一起恢复。
- 验证通过：内外 health ok；首页/admin 200；`/api/v1/public/products`、
  `/api/v1/public/config` 返回真实数据；app.log 无 error/fatal；
  `sso_auth` 下发正常（enabled=true, only_enabled=true, auth.kanoapi.top），
  registration_enabled=false（SSO-only 策略），invite_only_enabled=false（新开关存在且关闭）。
- 脚本小瑕疵：1/6 步骤记录旧镜像的 python 解析在 compose 新版 JSON 数组输出下报错
  （不影响流程，旧镜像可从 .env.bak 查得），待修。

## 部署后重点回归

- 登录/注册：SSO-only 正常；本地降级路径（`?local=1`）正常。
- 邀请码注册：线上 SSO-only 模式下本地注册被守卫拒绝，邀请码功能不生效，属预期。
- 采购单：状态流转、取消、退款限额；佣金创建与重试。
- 上游图片下载与上传目录脚本禁用两项安全策略不影响正常商品图展示。
