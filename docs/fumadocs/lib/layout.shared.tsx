import type { BaseLayoutProps } from 'fumadocs-ui/layouts/shared';
import Image from 'next/image';

type BaseOptionsInput = {
  includeMainLinks?: boolean;
};

// fill this with your actual GitHub info, for example:
export const gitConfig = {
  user: 'qianlima-666',
  repo: 'nginxpulse',
  branch: 'main',
};

export function baseOptions({ includeMainLinks = true }: BaseOptionsInput = {}): BaseLayoutProps {
  return {
    nav: {
      title: (
        <span className="inline-flex items-center gap-2">
          <Image src="/brand-mark.svg" alt="NginxPulse Logo" width={20} height={20} className="rounded-sm" />
          <span>NginxPulse Docs</span>
        </span>
      ),
    },
    githubUrl: `https://github.com/${gitConfig.user}/${gitConfig.repo}`,
    links: includeMainLinks
      ? [
          {
            type: 'main',
            text: '快速开始',
            url: '/docs/Quick-Start',
            active: 'nested-url',
          },
          {
            type: 'main',
            text: '配置',
            url: '/docs/Configuration',
            active: 'nested-url',
          },
          {
            type: 'main',
            text: '常见问题',
            url: '/docs/FAQ',
            active: 'nested-url',
          },
          {
            type: 'main',
            text: '用户交流群',
            url: 'https://github.com/qianlima-666/nginxpulse/issues/9',
            external: true,
            active: 'none',
          },
        ]
      : [],
  };
}
