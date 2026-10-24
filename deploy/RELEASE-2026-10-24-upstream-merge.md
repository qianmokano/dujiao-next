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

按 deploy/README.md 执行；本次无数据库迁移，仍按完整备份流程操作：

1. 备份 config.yml、SQLite 数据目录、上传目录、Redis 数据卷。
2. `.env` 中 `APP_IMAGE=ghcr.io/qianmokano/dujiao-next:v0.1.0-1`。
3. `docker compose pull app` → 停应用与 Redis → `docker compose up -d`。
4. 验证 `https://store.kanoapi.top/health`、登录（SSO-only）、下单、采购单状态。

## 部署后重点回归

- 登录/注册：SSO-only 正常；本地降级路径（`?local=1`）正常。
- 邀请码注册：线上 SSO-only 模式下本地注册被守卫拒绝，邀请码功能不生效，属预期。
- 采购单：状态流转、取消、退款限额；佣金创建与重试。
- 上游图片下载与上传目录脚本禁用两项安全策略不影响正常商品图展示。
