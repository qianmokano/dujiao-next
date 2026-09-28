# 单台 VPS 部署（Docker Compose + 主机 Nginx）

目标域名：`store.kanoapi.top`。应用镜像由本 fork 的 GitHub Actions 构建；VPS 不编译源码。Nginx 安装在 VPS 主机上，代理 Docker 仅绑定到 `127.0.0.1:8080` 的应用端口。

## 1. 发布自己的镜像

将本目录和 `.github/workflows/publish-image.yml` 合并到 fork 的 `main` 后，从已通过 CI 的提交创建一个未用过的 `v*` 标签并推送：

```bash
git tag v0.1.0-kano.1
git push origin v0.1.0-kano.1
```

等待 GitHub Actions 的 **Publish fork image** 完成。镜像地址为 `ghcr.io/qianmokano/dujiao-next:<标签>`。GitHub Container Registry 首次发布的包默认是私有的；在 GitHub 的 Packages 页面将该镜像设为 Public，VPS 才能免登录拉取。不要使用程序后台的上游一键更新功能管理这个 fork。

## 2. 准备 VPS

在 VPS 安装 Docker Engine、Compose 插件和 Nginx。Docker 安装命令按实际系统选择 [Ubuntu](https://docs.docker.com/engine/install/ubuntu/) 或 [Debian](https://docs.docker.com/engine/install/debian/) 官方步骤；不要混用两个发行版的仓库配置。确认 `docker compose version` 可运行，域名已指向 VPS，80/443 可访问。

从本机把本目录复制到 VPS（替换 SSH 用户、端口和公网 IP）。此域名当前走 Cloudflare 代理，SSH 应连接 VPS 的公网 IP 或单独的 DNS only 主机名，不使用代理后的 `store.kanoapi.top`：

```bash
rsync -av -e 'ssh -p <SSH_PORT>' deploy/ <SSH_USER>@<VPS_IP>:/tmp/dujiao-deploy/
```

随后在 VPS 上执行：

```bash
sudo install -d -m 0755 /opt/dujiao-next
sudo cp -a /tmp/dujiao-deploy/. /opt/dujiao-next/
cd /opt/dujiao-next
test -e .env || sudo cp .env.example .env
test -e config.yml || sudo cp config.yml.example config.yml
sudo chmod 0600 .env config.yml
sudo mkdir -p db uploads logs backups
sudo chmod 0750 db uploads logs backups
```

编辑 `.env`，把 `APP_IMAGE` 改为刚发布的完整版本镜像。若是新安装，编辑 `config.yml`：用三次 `openssl rand -hex 32` 生成三个不同的密钥，设置一个符合密码策略的管理员密码。若 VPS 已有 `config.yml` 和数据目录，先备份并保留现有密钥、数据库与后台路径，只核对 `server.mode: release`、反向代理地址、Redis/queue 主机及 CORS 来源。首次管理员创建并能登录后，删掉 `bootstrap.default_admin_password` 的值并重启应用。`config.yml` 和 `.env` 只放在 VPS，不提交 Git。

本配置固定 Docker 子网 `172.29.51.0/24`，应用仅信任该网络的网关 `172.29.51.1` 与回环地址作为反向代理。若 VPS 已使用该子网，应同时修改 `compose.yaml` 的子网/网关与 `config.yml` 的 `server.trusted_proxies`。Redis 同时提供 `redis` 和 `dujiao-redis` 两个内网名称，以兼容已有配置；它不发布主机端口。

验证配置并启动：

```bash
sudo docker compose config --quiet
sudo docker compose pull
sudo docker compose up -d
sudo docker compose ps
curl -fsS http://127.0.0.1:8080/health
```

如果容器未健康，先看 `sudo docker compose logs --tail=100 app redis`。

## 3. 配置 Nginx 与 HTTPS

`nginx.conf` 是首次启用的 HTTP 配置，安装后 Certbot 会在 Nginx 配置中加入 HTTPS 设置；之后不要直接用此模板覆盖已签发证书的站点配置。此域名经过 Cloudflare 代理；`cloudflare-real-ip.conf` 仅信任 [Cloudflare 官方 IP 网段](https://www.cloudflare.com/ips/)传来的 `CF-Connecting-IP`，使 Nginx 和应用的限流、审计拿到访客 IP。Cloudflare 网段变更时应更新此文件并重新加载 Nginx；不要直接信任来自任意地址的同名请求头。

```bash
sudo install -Dm0644 cloudflare-real-ip.conf /etc/nginx/snippets/dujiao-cloudflare-real-ip.conf
sudo cp nginx.conf /etc/nginx/sites-available/store.kanoapi.top
sudo ln -s /etc/nginx/sites-available/store.kanoapi.top /etc/nginx/sites-enabled/store.kanoapi.top
sudo nginx -t
sudo systemctl reload nginx
```

按 [Certbot 的 Nginx 安装说明](https://certbot.eff.org/instructions?os=snap&ws=nginx)安装 Certbot，然后运行：

```bash
sudo certbot --nginx -d store.kanoapi.top --redirect
curl -fsS https://store.kanoapi.top/health
```

若 Cloudflare 的 HTTPS 设置使首次 HTTP 验证无法到达源站，可暂时把此域名改为 DNS only，完成证书签发后恢复代理，并在 Cloudflare 选择 Full (strict) 模式；确认 DNS 记录指向这台 VPS、源站 80/443 端口可达，且 Certbot 生成的 HTTPS server 块仍包含上述 `include`。Cloudflare 的 [521 说明](https://developers.cloudflare.com/support/troubleshooting/http-status-codes/cloudflare-5xx-errors/error-521/)和 [Full (strict) 要求](https://developers.cloudflare.com/ssl/origin-configuration/ssl-modes/full-strict/)可用于排查。

最后访问 `https://store.kanoapi.top/admin`，完成管理员登录，并检查商品浏览、测试下单和异步任务日志。管理员密码从 `config.yml` 移除后，执行 `sudo docker compose restart app`。

## 4. 后续更新

每次修改走功能分支和 PR；CI 通过后合并到 `main`，创建新标签，等待新镜像发布。生产发布由你在 VPS 手动执行：

1. 把 `.env` 中的 `APP_IMAGE` 改成新标签，执行 `sudo docker compose pull app`，此时旧应用仍在运行。
2. 停止应用与 Redis，备份 `config.yml`、`db/`、`uploads/` 和 Redis 数据卷。SQLite 使用 WAL，运行中的数据库不要只复制单个 `.db` 文件：

   ```bash
   cd /opt/dujiao-next
   BACKUP_ID=$(date -u +%Y%m%dT%H%M%SZ)
   sudo docker compose stop app redis
   sudo tar -czf "backups/app-${BACKUP_ID}.tar.gz" config.yml db uploads
   sudo docker run --rm --entrypoint tar -v dujiao-next-redis-data:/data:ro -v "$PWD/backups:/backup" redis:7-alpine -C /data -czf "/backup/redis-${BACKUP_ID}.tar.gz" .
   ```

3. 确认两个归档文件已生成，再执行 `sudo docker compose up -d`，检查 `sudo docker compose ps`、`https://store.kanoapi.top/health` 和日志。

数据库会在应用启动时迁移。若新版本迁移后需要回退，应一起恢复更新前的数据备份和旧镜像；仅改回镜像标签不能保证旧程序兼容新数据库。定期把备份复制到 VPS 之外，并演练恢复。
