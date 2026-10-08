# 项目工作流

本文件记录此 fork 当前采用的开发与部署流程。具体部署命令和备份步骤以 [`deploy/README.md`](deploy/README.md) 为准。

## 开始修改前

1. 检查当前分支、`git status` 和远端状态。已有未提交改动视为其他正在进行的工作；不要覆盖、混入提交或擅自清理。如工作会冲突，先停止并说明冲突。
2. 从最新的 fork `main` 为新需求创建功能分支；需要隔离时使用独立工作区。先查看项目中至少三处类似实现，再遵循现有代码、命名和测试风格。
3. 每次项目或部署分析都在 Markdown 文件中留下结论、依据和下一步，例如 `deploy/ANALYSIS-YYYY-MM-DD.md`。记录中不得包含密码、令牌、密钥或 SMTP 授权码。

## 开发与发布镜像

1. 在功能分支做小步修改，运行相关测试和构建。只暂存、提交本次任务直接产生的文件，提交前核对暂存区。
2. 推送分支并发起 PR。等待仓库的 `ci` 工作流通过，再将 PR 合并到 fork 的 `main`。
3. 从已合并且通过检查的 `main` 提交创建**未使用过**的 `vX.Y.Z-N` 标签并推送（例如 `v0.1.0-1`）。`X.Y.Z` 沿用维护的版本基线，同一基线的发布序号从 1 递增，升级基线后重置为 1。不要重用或移动已发布的标签，旧 `-kano.N` 标签保留原名。
4. 标签推送触发 `.github/workflows/publish-image.yml`：再次运行 Go 和前端测试，成功后发布 `ghcr.io/qianmokano/dujiao-next:<标签>` 的 amd64/arm64 镜像。确认工作流成功且 VPS 有权限拉取镜像。

## 手动更新 VPS

线上使用 Docker Compose 运行应用和 Redis；主机 Nginx 提供 HTTPS，应用只绑定 `127.0.0.1:8080`。VPS 不编译源码，也不使用程序后台的上游一键更新功能管理此 fork。

1. 进入 `/opt/dujiao-next`，在 `.env` 把 `APP_IMAGE` 改为新镜像标签，先执行 `docker compose pull app`。镜像拉取成功前保留旧容器运行。
2. 按 `deploy/README.md` 停止应用和 Redis，并备份 `config.yml`、完整 SQLite 数据目录、上传目录及 Redis 数据卷。确认备份存在后再切换版本；不要只复制运行中 SQLite 的单个 `.db` 文件。
3. 执行 `docker compose up -d`，检查 `docker compose ps`、日志、`https://store.kanoapi.top/health` 和关键业务流程。
4. 若新版本迁移了数据库，回退时同时恢复旧镜像和更新前的数据备份。定期将备份复制到 VPS 之外并演练恢复。

`/opt/dujiao-next/config.yml`、`.env`、数据库、上传文件、Nginx 证书与站点配置属于 VPS 运行状态，不提交到 Git。Certbot 已修改的 Nginx 站点配置不能用首次安装模板直接覆盖。

## 当前基线

截至 2026-09-29，首次部署的镜像标签为 `v0.1.0-kano.1`。此信息会随发布过期；每次更新前应核对 VPS 实际运行的镜像标签和健康状态。本地尚有未提交改动，提交与发布时需逐项确认归属。
