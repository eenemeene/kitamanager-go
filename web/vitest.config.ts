import { fileURLToPath } from 'node:url'
import { defineConfig, mergeConfig } from 'vitest/config'
import viteConfig from './vite.config'

export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      environment: 'happy-dom',
      include: ['src/**/*.{test,spec}.{js,ts}'],
      coverage: {
        provider: 'v8',
        reporter: ['text', 'json', 'html'],
        include: ['src/**/*.{js,ts,vue}'],
        exclude: [
          'src/**/*.d.ts',
          'src/**/*.{test,spec}.{js,ts}',
          'src/main.ts',
          'src/vite-env.d.ts'
        ]
      },
      root: fileURLToPath(new URL('./', import.meta.url))
    }
  })
)
