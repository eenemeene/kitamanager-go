import type { Metadata } from 'next';
import { NextIntlClientProvider } from 'next-intl';
import { getLocale, getMessages } from 'next-intl/server';
import { Inter } from 'next/font/google';
import './globals.css';
import { Providers } from './providers';

const inter = Inter({ subsets: ['latin'] });

export const metadata: Metadata = {
  title: 'KitaManager',
  description: 'Kindergarten management system',
  // The report-pdf tool reads `kitamanager-version` from the rendered
  // DOM (via Playwright) and stamps it into the colophon at the foot of
  // every generated PDF — that's how the API and the CLI know which
  // frontend build actually rendered the print pages they consumed.
  // Embedding it here means every page across the app carries it,
  // including the (print) routes, with no per-page wiring.
  //
  // process.env.APP_VERSION is read at *build* time (the Next.js
  // metadata API runs on the server during `npm run build` for static
  // pages). Dockerfile.frontend's builder stage takes APP_VERSION as a
  // build arg so the value flows through. We deliberately do NOT set
  // `dynamic = 'force-dynamic'` here because that turned out to hang
  // the multi-arch buildx run for the frontend image — the static
  // build-time read is fine since the image is immutable per release.
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
