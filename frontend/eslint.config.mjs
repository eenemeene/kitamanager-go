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
    ignores: ['.next/', 'node_modules/', 'coverage/', 'jest.config.cjs', 'jest.setup.js'],
  },
];

export default config;
