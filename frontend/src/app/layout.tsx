import type { Metadata } from 'next';
import { NextIntlClientProvider } from 'next-intl';
import { getLocale, getMessages } from 'next-intl/server';
import { Inter } from 'next/font/google';
import './globals.css';
import { Providers } from './providers';

const inter = Inter({ subsets: ['latin'] });

// Prevent Next.js from statically rendering this layout at build time.
// The kitamanager-version meta tag below reads APP_VERSION from the
// runtime container env (set by Dockerfile.frontend) — a static render
// would freeze the build-time value, which is empty in CI.
export const dynamic = 'force-dynamic';

export const metadata: Metadata = {
  title: 'KitaManager',
  description: 'Kindergarten management system',
  // The report-pdf tool reads `kitamanager-version` from the rendered
  // DOM (via Playwright) and stamps it into the colophon at the foot of
  // every generated PDF — that's how the API and the CLI know which
  // frontend build actually rendered the print pages they consumed.
  // Embedding it here means every page across the app carries it,
  // including the (print) routes, with no per-page wiring.
  other: {
    'kitamanager-version': process.env.APP_VERSION ?? 'dev',
  },
};

export default async function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const locale = await getLocale();
  const messages = await getMessages();

  return (
    <html lang={locale} suppressHydrationWarning>
      <body className={inter.className}>
        <NextIntlClientProvider messages={messages}>
          <Providers>{children}</Providers>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
