## 应用简介

新增应用：`NginxPulse`

NginxPulse 是一个轻量级 Nginx 访问日志分析与可视化面板，支持实时统计、PV 过滤、IP 归属地解析与客户端信息解析，适合用于 Nginx 访问日志的可视化分析。

项目地址：

- GitHub: https://github.com/qianlima-666/nginxpulse
- 文档: https://nginx-pulse-docs.kaisir.cn/

## 镜像版本

- `magiccoders/nginxpulse:v1.6.15`

## 主要环境变量说明

- `WEB_HTTP_PORT`：Web UI 访问端口，默认 `8088`
- `LOG_PATH`：宿主机 Nginx 日志目录，默认 `/var/log/nginx`
- `PUID`：容器内应用用户 UID，默认 `1000`
- `PGID`：容器内应用用户 GID，默认 `1000`
- `TZ`：时区，默认 `Asia/Shanghai`
- `DB_DSN`：可选外置 PostgreSQL 连接串，留空时使用内置 PostgreSQL
- `HOST_IP`：主机监听 IP
- `CPUS`：CPU 限制
- `MEMORY_LIMIT`：内存限制
- `APP_PATH`：应用数据目录

## 依赖说明

- 无强制依赖
- 默认使用镜像内置 PostgreSQL
- 也支持通过 `DB_DSN` 配置外置 PostgreSQL

## 自测情况

- 已补充 `app.json`、图标、`.env`、`docker-compose.yml`
- `docker-compose.yml` 已按仓库规范使用 `${HOST_IP}:${PORT}` 映射端口
- 已添加 `labels.createdBy: "bt_apps"`
- 数据目录已统一挂载到 `${APP_PATH}` 下
- 已确认目录结构符合仓库规范
