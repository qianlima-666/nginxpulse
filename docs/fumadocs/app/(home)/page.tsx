import Link from 'next/link';
import {
  Activity,
  ArrowRight,
  BookOpen,
  Boxes,
  Cable,
  Cpu,
  Database,
  ExternalLink,
  FileSearch,
  Globe2,
  Github,
  Rocket,
  Search,
  Server,
  Settings2,
  Shield,
  TerminalSquare,
} from 'lucide-react';

export default function HomePage() {
  const docsCards = [
    {
      title: '快速开始',
      description: '5 分钟内完成部署与基础配置。',
      href: '/docs/Quick-Start',
      icon: Rocket,
    },
    {
      title: '部署方式',
      description: 'Docker、二进制与常见部署参数。',
      href: '/docs/Deployment',
      icon: Boxes,
    },
    {
      title: '配置说明',
      description: '系统、数据库、网站与日志源配置。',
      href: '/docs/Configuration',
      icon: Settings2,
    },
    {
      title: '日志解析',
      description: '覆盖 Nginx/Caddy/Apache/IIS/雷池 WAF 等多类日志。',
      href: '/docs/Log-Parsing',
      icon: FileSearch,
    },
    {
      title: '数据库结构',
      description: 'PostgreSQL 表结构与字段含义。',
      href: '/docs/Database-Schema',
      icon: Database,
    },
    {
      title: '常见问题',
      description: '启动、时区、性能与迁移问题排查。',
      href: '/docs/FAQ',
      icon: Activity,
    },
  ];
  const heroHighlights = [
    {
      icon: Server,
      label: '支持平台',
      value: 'Nginx / Caddy / Nginx Proxy Manager / Apache / IIS / HAProxy / Traefik / Envoy / Tengine / Safeline WAF',
    },
    {
      icon: Cable,
      label: '采集方式',
      value: 'local / sftp / http / s3 / agent',
    },
    {
      icon: Cpu,
      label: '部署模式',
      value: '单机部署 / 中心服务 + 多 Agent 分布式部署',
    },
  ];

  return (
    <div className="relative mx-auto w-full max-w-(--fd-layout-width) flex-1 overflow-hidden px-4 pb-14 pt-1 md:pt-2">
      <section className="np-home-reveal relative mt-1 rounded-2xl border border-fd-border bg-fd-card/76 p-6 shadow-[0_12px_32px_-30px_rgba(37,99,235,.30)] backdrop-blur sm:mt-2 sm:p-8 lg:p-10">
        <div className="absolute right-4 top-4 hidden items-center gap-2 sm:flex">
          <div className="inline-flex items-center gap-1.5 rounded-full border border-fd-border bg-fd-background/85 px-3 py-1 text-xs text-fd-muted-foreground">
            <Globe2 className="size-3.5 text-fd-primary" />
            <Link href="/docs/Home" className="font-medium text-fd-foreground/85 hover:text-fd-foreground">
              中文
            </Link>
            <span className="text-fd-border">/</span>
            <Link href="/docs/Home-EN" className="font-medium text-fd-foreground/85 hover:text-fd-foreground">
              EN
            </Link>
          </div>
          <Link
            href="https://github.com/qianlima-666/nginxpulse/?tab=MIT-1-ov-file"
            target="_blank"
            rel="noreferrer"
            className="inline-flex items-center gap-1.5 rounded-full border border-fd-border bg-fd-background/85 px-3 py-1 text-xs text-fd-muted-foreground hover:text-fd-foreground"
          >
            <Shield className="size-3.5 text-fd-primary" />
            MIT License
          </Link>
        </div>

        <div className="max-w-4xl">
          <p className="mt-3 inline-flex items-center gap-2 rounded-full border border-fd-border bg-fd-background px-3 py-1 text-xs font-medium text-fd-muted-foreground">
            <TerminalSquare className="size-3.5 text-fd-primary" />
            NginxPulse-多平台访问日志分析
          </p>
          <h1 className="mt-5 text-3xl font-semibold tracking-tight text-fd-foreground sm:text-5xl">
            轻量级 Nginx 访问日志分析与可视化平台
          </h1>
          <p className="mt-4 max-w-2xl text-base text-fd-muted-foreground sm:text-lg">
            不止支持 Nginx：同时支持 Caddy、Nginx Proxy Manager、Apache、IIS、HAProxy、Traefik、Envoy、Tengine、雷池 WAF 等多种日志格式。
            支持本地、远端与 Agent 多源采集，并可通过中心服务 + 多 Agent 进行分布式部署。
          </p>
          <div className="mt-4 inline-flex items-center gap-2 text-sm text-fd-muted-foreground sm:hidden">
            <Globe2 className="size-4 text-fd-primary" />
            <span>语言:</span>
            <Link href="/docs/Home" className="font-medium text-fd-foreground/85 hover:text-fd-foreground">
              中文
            </Link>
            <span>/</span>
            <Link href="/docs/Home-EN" className="font-medium text-fd-foreground/85 hover:text-fd-foreground">
              EN
            </Link>
          </div>

          <div className="mt-5 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {heroHighlights.map((item) => {
              const Icon = item.icon;
              return (
                <div key={item.label} className="rounded-lg border border-fd-border bg-fd-background/80 px-3.5 py-3">
                  <div className="flex items-center gap-2">
                    <span className="inline-flex rounded-md border border-fd-border bg-fd-card p-1.5 text-fd-primary">
                      <Icon className="size-3.5" />
                    </span>
                    <p className="text-sm font-semibold text-fd-foreground">{item.label}</p>
                  </div>
                  <p className="mt-2 text-sm leading-relaxed text-fd-muted-foreground">{item.value}</p>
                </div>
              );
            })}
          </div>

          <div className="np-home-action-row mt-6">
            <Link
              href="/docs/Quick-Start"
              className="np-home-card np-home-action-btn group inline-flex items-center justify-center rounded-lg border border-fd-primary bg-fd-primary px-4 py-2 text-sm font-semibold text-fd-primary-foreground"
            >
              <span className="np-home-action-content">
                <Rocket className="np-home-action-icon size-4" />
                <span className="np-home-action-label">立即开始</span>
              </span>
              <ArrowRight className="np-home-action-arrow np-home-action-arrow-intro size-4" />
            </Link>
            <Link
              href="https://github.com/qianlima-666/nginxpulse"
              target="_blank"
              rel="noreferrer"
              className="np-home-card np-home-action-btn group inline-flex items-center rounded-lg border border-fd-border bg-fd-card px-4 py-2 text-sm font-medium text-fd-muted-foreground hover:text-fd-foreground"
            >
              <span className="np-home-action-content">
                <Github className="np-home-action-icon size-4" />
                <span className="np-home-action-label">GitHub 仓库</span>
              </span>
            </Link>
            <Link
              href="https://nginx-pulse.kaisir.cn/"
              target="_blank"
              rel="noreferrer"
              className="np-home-card np-home-action-btn group inline-flex items-center rounded-lg border border-fd-border bg-fd-card px-4 py-2 text-sm font-medium text-fd-muted-foreground hover:text-fd-foreground"
            >
              <span className="np-home-action-content">
                <ExternalLink className="np-home-action-icon size-4" />
                <span className="np-home-action-label">演示站点</span>
              </span>
            </Link>
            <div className="np-home-shortcut group inline-flex items-center rounded-lg border border-fd-border bg-fd-background px-4 py-2">
              <span className="np-home-action-content">
                <Search className="np-home-shortcut-icon size-4 text-fd-primary" />
                <span className="np-home-shortcut-label">快捷搜索:</span>
              </span>
              <kbd className="np-home-kbd rounded border border-fd-border bg-fd-card px-1.5 py-0.5 text-xs text-fd-foreground">
                Cmd/Ctrl + K
              </kbd>
            </div>
          </div>
        </div>
      </section>

      <section className="np-home-reveal np-home-delay-1 mt-6 grid grid-cols-1 gap-3 sm:grid-cols-3">
        <div className="rounded-xl border border-fd-border bg-fd-card p-4">
          <p className="text-xs font-medium uppercase tracking-[0.08em] text-fd-muted-foreground">多平台解析</p>
          <p className="mt-2 text-base font-semibold text-fd-foreground">一套平台覆盖多种日志格式</p>
          <p className="mt-2 text-sm text-fd-muted-foreground">
            内置多 logType 解析器，适配 Web 服务器、网关、Ingress 与 WAF 场景。
          </p>
        </div>
        <div className="rounded-xl border border-fd-border bg-fd-card p-4">
          <p className="text-xs font-medium uppercase tracking-[0.08em] text-fd-muted-foreground">多来源采集</p>
          <p className="mt-2 text-base font-semibold text-fd-foreground">本地 + 远端 + Agent 统一接入</p>
          <p className="mt-2 text-sm text-fd-muted-foreground">
            支持 local / sftp / http / s3 / agent，支持轮询与流式模式组合。
          </p>
        </div>
        <div className="rounded-xl border border-fd-border bg-fd-card p-4">
          <p className="text-xs font-medium uppercase tracking-[0.08em] text-fd-muted-foreground">分布式部署</p>
          <p className="mt-2 text-base font-semibold text-fd-foreground">中心服务 + 多节点采集</p>
          <p className="mt-2 text-sm text-fd-muted-foreground">
            Agent 可部署在多台机器或集群节点，汇聚到统一 NginxPulse 服务端。
          </p>
        </div>
      </section>

      <section className="np-home-reveal np-home-delay-2 mt-8">
        <div className="mb-3 flex items-center gap-2">
          <BookOpen className="size-4.5 text-fd-primary" />
          <h2 className="text-lg font-semibold text-fd-foreground">文档地图</h2>
        </div>
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
          {docsCards.map((item) => {
            const Icon = item.icon;
            return (
              <Link
                key={item.href}
                href={item.href}
                className="np-home-card group rounded-xl border border-fd-border bg-fd-card p-4"
              >
                <div className="mb-3 inline-flex rounded-md border border-fd-border bg-fd-background p-2 text-fd-primary">
                  <Icon className="size-4.5" />
                </div>
                <h3 className="text-base font-semibold text-fd-foreground">{item.title}</h3>
                <p className="mt-2 text-sm text-fd-muted-foreground">{item.description}</p>
              </Link>
            );
          })}
        </div>
      </section>

    </div>
  );
}
