import nextConfig from 'eslint-config-next';
import coreWebVitals from 'eslint-config-next/core-web-vitals';

/** @type {import('eslint').Linter.Config[]} */
const config = [
  ...coreWebVitals,
  {
    // Disable React Compiler rules — not using React Compiler
    rules: {
      'react-hooks/purity': 'off',
      'react-hooks/immutability': 'off',
      'react-hooks/incompatible-library': 'off',
      'react-hooks/preserve-manual-memoization': 'off',
      'react-hooks/globals': 'off',
    },
  },
  {
    // Playwright specs only. A rejected expectation is a promise rejection, so
    // `.catch()` anywhere on an assertion chain turns a real failure into a
    // silent pass -- dashboard.spec.ts carried one for months and the test could
    // not go red no matter what the dashboard did. Cleanup in afterAll may still
    // swallow, which is why this is scoped to chains containing an expect().
    files: ['e2e/**/*.ts'],
    rules: {
      'no-restricted-syntax': [
        'error',
        {
          selector: 'CallExpression[callee.property.name="catch"]:has(CallExpression[callee.name="expect"])',
          message:
            'Do not attach .catch() to an expect() chain -- it discards the rejection and the assertion can never fail. Assert the condition directly (toHaveCount(0), toBeVisible()), or restructure the check.',
        },
      ],
    },
  },
  {
    ignores: ['.next/', 'node_modules/', 'coverage/', 'jest.config.cjs', 'jest.setup.js'],
  },
];

export default config;
