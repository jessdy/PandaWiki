import { withSentryConfig } from '@sentry/nextjs';
import type { NextConfig } from 'next';
import path from 'path';
import { fileURLToPath } from 'url';

const appRoot = path.dirname(fileURLToPath(import.meta.url));

const nextConfig: NextConfig = {
  distDir: 'dist',
  reactStrictMode: false,
  allowedDevOrigins: ['10.10.18.71'],
  output: 'standalone',
  outputFileTracingRoot: path.join(appRoot, '..'),
  assetPrefix: '/panda-wiki-app-assets',
  logging: {
    fetches: {
      fullUrl: true,
    },
  },
  images: {
    unoptimized: true,
  },
  transpilePackages: [
    'mermaid',
    '@panda-wiki/icons',
    '@panda-wiki/themes',
    '@panda-wiki/ui',
    '@ctzhian/tiptap',
    '@ctzhian/ui',
  ],
  webpack: (config, { isServer }) => {
    // 把 app 自己的 node_modules 放最前，避免 monorepo 父级 node_modules 截胡
    // 不再手动 alias 'entities/decode'：webpack 5 已原生支持 package.json 的 exports
    // 子路径解析，旧 alias 指向的 lib/decode.js 在 entities@6+ 已不存在。
    config.resolve.modules = [
      path.join(appRoot, 'node_modules'),
      ...(config.resolve.modules ?? ['node_modules']),
    ];
    if (!isServer) {
      config.resolve.fallback = {
        ...(config.resolve.fallback || {}),
        fs: false,
        path: false,
        'fs/promises': false,
        'node:fs': false,
        'node:fs/promises': false,
        'node:path': false,
      };
    }
    return config;
  },
  async headers() {
    return [
      {
        source: '/cap@0.0.6/:path*',
        headers: [
          {
            key: 'Cache-Control',
            value: 'public, max-age=31536000, must-revalidate',
          },
        ],
      },
    ];
  },
  async rewrites() {
    const rewritesPath = [];
    if (process.env.NODE_ENV === 'development') {
      // 只有当环境变量存在且是完整 URL 时才添加 rewrite
      if (process.env.STATIC_FILE_TARGET) {
        const staticFileTarget = process.env.STATIC_FILE_TARGET.trim();
        // 确保是完整的 URL（以 http:// 或 https:// 开头）
        if (
          staticFileTarget.startsWith('http://') ||
          staticFileTarget.startsWith('https://')
        ) {
          rewritesPath.push({
            source: '/static-file/:path*',
            destination: `${staticFileTarget}/static-file/:path*`,
            basePath: false as const,
          });
        }
      }
      // demo 分支：禁用转发到 TARGET，全部由本地 Mock / proxy 处理
      // （保留 STATIC_FILE_TARGET 可选代理）
    }
    return rewritesPath;
  },
};

// 在开发环境下跳过 Sentry 配置
const isDevelopment = process.env.NODE_ENV === 'development';

export default isDevelopment
  ? nextConfig
  : withSentryConfig(nextConfig, {
      // For all available options, see:
      // https://www.npmjs.com/package/@sentry/webpack-plugin#options

      org: 'sentry',

      project: 'pandawiki-app',
      sentryUrl: 'https://sentry.baizhi.cloud/',

      // Only print logs for uploading source maps in CI
      silent: !process.env.CI,

      // For all available options, see:
      // https://docs.sentry.io/platforms/javascript/guides/nextjs/manual-setup/

      // Upload a larger set of source maps for prettier stack traces (increases build time)
      widenClientFileUpload: true,

      // Route browser requests to Sentry through a Next.js rewrite to circumvent ad-blockers.
      // This can increase your server load as well as your hosting bill.
      // Note: Check that the configured route will not match with your Next.js proxy, otherwise reporting of client-
      // side errors will fail.
      tunnelRoute: '/monitoring',

      // Automatically tree-shake Sentry logger statements to reduce bundle size
      disableLogger: true,

      // Enables automatic instrumentation of Vercel Cron Monitors. (Does not yet work with App Router route handlers.)
      // See the following for more information:
      // https://docs.sentry.io/product/crons/
      // https://vercel.com/docs/cron-jobs
      automaticVercelMonitors: true,
    });
