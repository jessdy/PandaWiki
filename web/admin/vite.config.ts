import react from '@vitejs/plugin-react';
import path from 'path';
import { visualizer } from 'rollup-plugin-visualizer';
import { defineConfig, loadEnv, Plugin } from 'vite';
import { execSync } from 'child_process';

// 创建路由生成插件
function generateRoutesPlugin(): Plugin {
  return {
    name: 'generate-routes',
    buildStart() {
      // 构建开始时生成路由
      try {
        execSync('node scripts/generate-routes.js', { stdio: 'inherit' });
      } catch (error) {
        console.error('生成路由失败:', error);
      }
    },
    handleHotUpdate({ file, server }) {
      // 开发模式下监听路由文件变化
      const routerPath = path.resolve(__dirname, 'src/router.tsx');
      if (file === routerPath) {
        console.log('🔄 检测到路由文件变化，正在更新路由列表...');
        try {
          execSync('node scripts/generate-routes.js', { stdio: 'inherit' });
          // 触发 HMR 更新 index.html
          server.ws.send({
            type: 'update',
            updates: [
              {
                type: 'js-update',
                path: '/index.html',
                acceptedPath: '/index.html',
                timestamp: Date.now(),
              },
            ],
          });
        } catch (error) {
          console.error('❌ 更新路由列表失败:', error);
        }
      }
    },
  };
}

export default defineConfig(({ command, mode }) => {
  // 加载环境变量 - 第二个参数是目录路径，不是文件名
  const env = loadEnv(mode, process.cwd(), '');
  const shouldAnalyze =
    process.argv.includes('--analyze') || env.ANALYZE === 'true';
  return {
    build: {
      assetsDir: 'panda-wiki-admin-assets',
    },
    server: {
      hmr: true,
      proxy: (() => {
        const proxyConfig: Record<string, any> = {};

        // 只有当环境变量存在且是有效 URL 时才添加代理配置
        if (
          env.TARGET &&
          (env.TARGET.startsWith('http://') ||
            env.TARGET.startsWith('https://'))
        ) {
          proxyConfig['/api'] = {
            target: env.TARGET.trim(),
            secure: false,
            changeOrigin: true,
          };
        }

        if (
          env.STATIC_FILE_TARGET &&
          (env.STATIC_FILE_TARGET.startsWith('http://') ||
            env.STATIC_FILE_TARGET.startsWith('https://'))
        ) {
          proxyConfig['/static-file'] = {
            target: env.STATIC_FILE_TARGET.trim(),
            secure: false,
            changeOrigin: true,
          };
        }

        if (
          env.SHARE_TARGET &&
          (env.SHARE_TARGET.startsWith('http://') ||
            env.SHARE_TARGET.startsWith('https://'))
        ) {
          proxyConfig['/share'] = {
            target: env.SHARE_TARGET.trim(),
            secure: false,
            changeOrigin: true,
          };
        } else if (
          env.TARGET &&
          (env.TARGET.startsWith('http://') ||
            env.TARGET.startsWith('https://'))
        ) {
          // 如果没有 SHARE_TARGET，使用 TARGET 作为后备
          proxyConfig['/share'] = {
            target: env.TARGET.trim(),
            secure: false,
            changeOrigin: true,
          };
        }

        return proxyConfig;
      })(),
      host: '0.0.0.0',
    },
    esbuild: {
      // 保留函数和类名，避免第三方库依赖 constructor.name 的逻辑在压缩后失效
      keepNames: true,
    },
    plugins: [
      react(),
      generateRoutesPlugin(),
      ...(command === 'build' && shouldAnalyze
        ? [
            visualizer({
              open: true, // 在默认浏览器中自动打开报告
              gzipSize: true, // 显示 gzip 格式下的包大小
              brotliSize: true, // 显示 brotli 格式下的包大小
              filename: 'dist/stats.html', // 分析图生成的文件名
            }),
          ]
        : []),
    ],
    resolve: {
      alias: {
        '@': path.resolve(__dirname, 'src'),
      },
    },
  };
});
