import { NextResponse } from 'next/server';

/**
 * GET /version — returns the frontend's build identity.
 *
 * The API and the frontend ship as separate container images and can
 * be at different versions in production. Each side advertises its
 * own version: the API at `/api/v1/health`, the frontend here. The
 * report-pdf tool reads both at startup and stamps the trio
 * (report-pdf · API · Web) onto every generated PDF so an artifact
 * found later carries enough provenance to reproduce the build.
 *
 * `force-dynamic` because Next.js would otherwise statically render
 * this route at build time and freeze the env values into the
 * published bundle — we need the response to reflect whatever ENV
 * was set on the running container, which is when APP_VERSION is
 * actually populated by Dockerfile.frontend.
 */
export const dynamic = 'force-dynamic';

export async function GET() {
  return NextResponse.json({
    version: process.env.APP_VERSION ?? 'dev',
    commit: process.env.APP_COMMIT ?? 'unknown',
    build_time: process.env.APP_BUILD_TIME ?? 'unknown',
  });
}
