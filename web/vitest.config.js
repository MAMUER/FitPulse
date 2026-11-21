import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';

export default defineConfig({
  plugins: [react()],
  resolve: {
    extensions: ['.js', '.jsx', '.json'],
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: './src/test/setup.js',
    include: ['src/**/*.{test,spec}.{js,jsx}'],
    // 4 vCPU runner: 4 workers instead of 2 roughly halves the wall time.
    // NOTE: pool 'vmThreads' would reuse the jsdom environment across files, but
    // window.location is non-configurable there, which breaks tests that stub it
    // (src/hooks/useAuth.test.js, src/screens/TwoFAVerify.test.jsx).
    maxWorkers: process.env.CI ? 4 : 1,
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html', 'lcov'],
      // Падение теста не должно отменять сбор coverage: по умолчанию Vitest
      // пишет отчёт только при зелёном прогоне, и один красный тест убирал
      // bin/lcov.info. SonarCloud при этом падал на «покрытие нового кода 0%»,
      // а не на настоящей причине — отсутствующем отчёте.
      reportOnFailure: true,
      exclude: [
        'node_modules/',
        'dist/',
        'src/main.jsx',
        'src/App.jsx',
        'src/test/setup.js',
      ],
    },
  },
});
