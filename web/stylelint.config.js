/** @type {import('stylelint').Config} */
export default {
  extends: ['stylelint-config-standard', 'stylelint-config-recommended-vue'],
  rules: {
    // Allow PrimeVue CSS custom properties
    'custom-property-pattern': null,
    // Allow kebab-case and camelCase for Vue scoped styles
    'selector-class-pattern': null,
    // Disable unknown at-rules for Vue <style scoped>
    'at-rule-no-unknown': [
      true,
      {
        ignoreAtRules: ['tailwind', 'apply', 'variants', 'responsive', 'screen']
      }
    ],
    // Allow empty source files
    'no-empty-source': null
  },
  overrides: [
    {
      files: ['**/*.vue'],
      customSyntax: 'postcss-html'
    }
  ]
}
