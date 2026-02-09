# KitaManager Go Website

This directory contains the Hugo-based website for KitaManager Go.

## Prerequisites

- [Hugo Extended](https://gohugo.io/installation/) v0.155.2 or later
- Node.js 18+ (for screenshot capture)

## Development

### Start the development server

```bash
cd website
hugo server -D
```

The site will be available at `http://localhost:1313/`.

### Build for production

```bash
hugo --minify
```

Output will be in the `public/` directory.

## Multilingual Support

The site is available in English and German:
- English: `/en/`
- German: `/de/`

## Screenshots

Screenshots are stored in `static/images/screenshots/` and displayed on the Screenshots page.

### Capturing Screenshots

To capture screenshots from the running application:

1. Start the KitaManager Go development environment:
   ```bash
   cd /path/to/kitamanager-go
   make dev
   ```

2. Wait for the application to be ready at `http://localhost:3000`

3. Install Playwright dependencies:
   ```bash
   cd frontend
   npx playwright install chromium
   ```

4. Run the screenshot capture script:
   ```bash
   cd website
   npx tsx scripts/capture-screenshots.ts
   ```

Alternatively, you can capture screenshots manually and place them in `static/images/screenshots/` with these names:
- `login.png` - Login page
- `dashboard.png` - Dashboard overview
- `organizations.png` - Organizations list
- `new-organization-dialog.png` - Create organization dialog
- `employees.png` - Employees list
- `children.png` - Children list with funding
- `child-detail.png` - Child detail page
- `government-funding.png` - Government funding configuration
- `sections.png` - Organizational sections

## Theme

The site uses the [Hextra](https://github.com/imfing/hextra) Hugo theme.

## Directory Structure

```
website/
├── content/
│   ├── en/              # English content
│   │   ├── _index.md    # Homepage
│   │   ├── docs/        # Documentation
│   │   ├── features/    # Features page
│   │   └── screenshots/ # Screenshots page
│   └── de/              # German content
│       ├── _index.md    # Homepage
│       ├── docs/        # Dokumentation
│       ├── features/    # Funktionen
│       └── screenshots/ # Bildschirmfotos
├── static/
│   └── images/
│       └── screenshots/ # Screenshot images
├── scripts/
│   └── capture-screenshots.ts
├── themes/
│   └── hextra/          # Hextra theme (git submodule)
├── hugo.yaml            # Hugo configuration
└── README.md            # This file
```

## Deployment

### GitHub Pages

Add a GitHub Actions workflow at `.github/workflows/hugo.yml`:

```yaml
name: Deploy Hugo site to Pages

on:
  push:
    branches: ["main"]
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: "pages"
  cancel-in-progress: false

defaults:
  run:
    shell: bash

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          submodules: recursive

      - name: Setup Hugo
        uses: peaceiris/actions-hugo@v3
        with:
          hugo-version: '0.155.2'
          extended: true

      - name: Build
        run: |
          cd website
          hugo --minify --baseURL "${{ steps.pages.outputs.base_url }}/"

      - name: Upload artifact
        uses: actions/upload-pages-artifact@v3
        with:
          path: ./website/public

  deploy:
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    runs-on: ubuntu-latest
    needs: build
    steps:
      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v4
```

### Other Hosting

Copy the `public/` directory to any static hosting provider (Netlify, Vercel, Cloudflare Pages, etc.).
