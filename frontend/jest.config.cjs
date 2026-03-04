const nextJest = require('next/jest');

const createJestConfig = nextJest({
  // Provide the path to your Next.js app to load next.config.js and .env files
  dir: './',
});

/** @type {import('jest').Config} */
const customJestConfig = {
  setupFilesAfterEnv: ['<rootDir>/jest.setup.js'],
  testEnvironment: 'jest-environment-jsdom',
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/src/$1',
  },
  testPathIgnorePatterns: ['<rootDir>/node_modules/', '<rootDir>/.next/', '<rootDir>/e2e/'],
  collectCoverageFrom: [
    'src/**/*.{js,jsx,ts,tsx}',
    '!src/**/*.d.ts',
    '!src/**/types.ts',
    '!src/app/**/layout.tsx',
    '!src/components/ui/**', // shadcn components
  ],
  coverageThreshold: {
    global: {
      statements: 55,
      branches: 40,
      functions: 35,
      lines: 55,
    },
    './src/**/*.{ts,tsx}': {
      lines: 5,
    },
  },
};

// Override next/jest's transformIgnorePatterns to allow MSW ESM dependencies
const jestConfig = createJestConfig(customJestConfig);
module.exports = async () => {
  const config = await jestConfig();
  config.transformIgnorePatterns = [
    'node_modules/(?!(msw|@mswjs|until-async|@bundled-es-modules)/)',
    '^.+\\.module\\.(css|sass|scss)$',
  ];
  return config;
};
