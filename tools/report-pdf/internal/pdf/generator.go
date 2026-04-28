package pdf

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	pdfcpuapi "github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/playwright-community/playwright-go"
)

// loginPathPrefix is the SPA route the frontend redirects to when no
// session cookie is present or it has expired. We detect bounces both
// right after the initial navigation and after waiting for the print-
// ready signal — that way an operator gets a clear "auth failed"
// message instead of a 30-second generic timeout when the frontend
// silently route-changed away from the print page.
const loginPathPrefix = "/login"

// isLoginBounce returns true when rawURL points at the login route.
// Parsing the URL (instead of strings.Contains over the whole string)
// guards against false positives like "/?next=/login" or report
// content that happens to mention login text in the path.
func isLoginBounce(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return u.Path == loginPathPrefix || strings.HasPrefix(u.Path, loginPathPrefix+"/")
}

// combinedReportURL builds the URL of the combined report print page
// for a given org. Extracted so the URL contract can be pinned by
// unit tests without spinning up Playwright — the path shape and
// `month` query parameter are part of the API/frontend contract a
// renaming on either side would silently break. month is passed
// through verbatim (already validated to YYYY-MM at the CLI layer);
// we don't re-validate here so the helper stays a pure formatter.
func combinedReportURL(baseURL, orgID, month string) string {
	return fmt.Sprintf("%s/organizations/%s/statistics/report/print?month=%s", baseURL, orgID, month)
}

type Generator struct {
	pw      *playwright.Playwright
	browser playwright.Browser
	cookies []playwright.OptionalCookie
	baseURL string
	// webVersion is captured from the rendered DOM the first time
	// GenerateReport sees a `<meta name="kitamanager-version">` tag.
	// We read it from the page rather than via a separate HTTP call so
	// the version we record is guaranteed to come from the same build
	// that actually rendered the print pages — no race, no parallel
	// route handler that can drift out of sync with the layout.
	webVersion string
}

// NewGenerator installs Playwright browsers if needed and launches a headless Chromium instance.
func NewGenerator(cookies []playwright.OptionalCookie, baseURL string) (*Generator, error) {
	if err := playwright.Install(); err != nil {
		return nil, fmt.Errorf("install playwright: %w", err)
	}

	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("start playwright: %w", err)
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		pw.Stop()
		return nil, fmt.Errorf("launch chromium: %w", err)
	}

	return &Generator{
		pw:      pw,
		browser: browser,
		cookies: cookies,
		baseURL: strings.TrimRight(baseURL, "/"),
	}, nil
}

// GenerateCombinedReport navigates to the combined /report/print page
// and exports it as a single PDF at outputPath. The frontend page
// drives all four section renders + cover + page-break CSS in one
// continuous document, so the tool no longer needs to render each
// section separately and merge afterwards.
//
// month is the YYYY-MM form of the report month — passed verbatim into
// the print page's `?month=` query so every API call the page makes is
// scoped to the same period.
func (g *Generator) GenerateCombinedReport(orgID, month, outputPath string) error {
	ctx, err := g.browser.NewContext(playwright.BrowserNewContextOptions{
		Viewport: &playwright.Size{Width: 1600, Height: 900},
	})
	if err != nil {
		return fmt.Errorf("create browser context: %w", err)
	}
	defer ctx.Close()

	if err := ctx.AddCookies(g.cookies); err != nil {
		return fmt.Errorf("add cookies: %w", err)
	}

	page, err := ctx.NewPage()
	if err != nil {
		return fmt.Errorf("create page: %w", err)
	}

	pageURL := combinedReportURL(g.baseURL, orgID, month)
	fmt.Printf("  Navigating to %s\n", pageURL)

	resp, err := page.Goto(pageURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
		Timeout:   playwright.Float(30000),
	})
	if err != nil {
		return fmt.Errorf("navigate to %s: %w", pageURL, err)
	}

	// Fast-path bounce detection: the SPA redirects unauthenticated
	// requests straight to the login route, so we can fail in <100ms
	// rather than waiting 30s for the print-ready signal that will
	// never arrive.
	if isLoginBounce(page.URL()) {
		return fmt.Errorf("redirected to %s — authentication failed (cookies expired or service account lacks access to org %s)", page.URL(), orgID)
	}

	if resp != nil && resp.Status() >= 400 {
		return fmt.Errorf("page returned HTTP %d", resp.Status())
	}

	// Slow-path: wait for the print-ready signal, but if it times out
	// re-check the URL — if the SPA route-changed to /login during the
	// wait (mid-render session expiry, race with token refresh), give
	// the operator the same clear auth-failure message rather than a
	// generic timeout.
	//
	// Timeout is generous (60s): the combined page composes ~30 parallel
	// queries across cover + 4 sections, plus chart-render time. Most
	// runs settle in 5–10s but a slow API or under-resourced runner can
	// take longer.
	err = page.Locator("[data-print-ready='true']").WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(60000),
	})
	if err != nil {
		if isLoginBounce(page.URL()) {
			return fmt.Errorf("page bounced to %s mid-render — session expired during PDF generation", page.URL())
		}
		return fmt.Errorf("timeout waiting for page to be ready: %w", err)
	}

	// Capture the web-version meta tag the first time we see it.
	// Subsequent reports won't change the value (same frontend build)
	// so we only sniff once. EvaluateOptions returns nil for missing
	// attributes — we just leave webVersion empty in that case.
	if g.webVersion == "" {
		if v, err := page.Evaluate(`document.querySelector('meta[name="kitamanager-version"]')?.content ?? ""`); err == nil {
			if s, ok := v.(string); ok {
				g.webVersion = s
			}
		}
	}

	// Inject defensive overflow:visible so chart libraries that clip
	// internally (recharts in particular) don't get cut off in the
	// PDF render. The combined report page is already laid out for
	// A4 landscape — its container is max-w-[1100px] and the page
	// owns its own @page CSS for size + margins — so we no longer
	// override max-width / padding here (those overrides came from
	// the per-section days when the page didn't know it was a
	// print target).
	if _, err := page.AddStyleTag(playwright.PageAddStyleTagOptions{
		Content: playwright.String(`
			body { margin: 0 !important; padding: 0 !important; }
			table, [data-print-ready] { overflow: visible !important; }
		`),
	}); err != nil {
		return fmt.Errorf("inject print CSS: %w", err)
	}

	// Brief stabilization delay for chart animations
	time.Sleep(1 * time.Second)

	marginMM := "10mm"
	_, err = page.PDF(playwright.PagePdfOptions{
		Path:            playwright.String(outputPath),
		Landscape:       playwright.Bool(true),
		PrintBackground: playwright.Bool(true),
		Format:          playwright.String("A4"),
		// Scale 1.0 (default): render at the same proportions a
		// user would see in the browser print dialog. Previous
		// value (0.55) was a workaround from when individual print
		// pages had content wider than A4 landscape — the combined
		// page is already laid out to fit (~1047px printable area
		// vs the 1100px container), and shrinking at the renderer
		// stage made charts like the financial-overview bar chart
		// render as 1-pixel sticks instead of readable bars.
		Scale: playwright.Float(1.0),
		Margin: &playwright.Margin{
			Top:    &marginMM,
			Bottom: &marginMM,
			Left:   &marginMM,
			Right:  &marginMM,
		},
	})
	if err != nil {
		return fmt.Errorf("generate PDF: %w", err)
	}
	return nil
}

// WebVersion returns the kitamanager-version meta tag value the
// generator captured from the rendered DOM during the most recent
// GenerateReport call. Returns empty string if no report has been
// generated yet or if the layout doesn't expose the meta tag (e.g.
// against a frontend at a version older than this feature).
func (g *Generator) WebVersion() string {
	return g.webVersion
}

// MergeFiles combines multiple PDF files into a single output file.
func MergeFiles(inputPaths []string, outputPath string) error {
	return pdfcpuapi.MergeCreateFile(inputPaths, outputPath, false, nil)
}

// AddProperties adds custom properties to an existing PDF file.
func AddProperties(pdfPath string, properties map[string]string) error {
	return pdfcpuapi.AddPropertiesFile(pdfPath, "", properties, nil)
}

// StampColophon writes `text` as a small bottom-center stamp on the
// last page of pdfPath in place. Used by the report tool to record the
// CLI + API versions and the build/render time so a reader can later
// reproduce or audit which code rendered the PDF in front of them —
// otherwise an artifact found a year later carries no provenance.
//
// pos:bc → bottom-center; sc:0.4 → smallish; op:0.6 → translucent so
// it never overpowers the chart on the page below it. onTop=true
// (stamp, not watermark) so it stays readable on top of any
// background fill the print page might have.
func StampColophon(pdfPath, text string) error {
	pages, err := pdfcpuapi.PageCountFile(pdfPath)
	if err != nil {
		return fmt.Errorf("count pages of %s: %w", pdfPath, err)
	}
	if pages == 0 {
		return fmt.Errorf("%s has no pages to stamp", pdfPath)
	}
	lastPage := strconv.Itoa(pages)
	// pdfcpu's stamp DSL: position bottom-center, horizontal (rotation:0
	// — the default would tilt the text ~45° like a watermark), 9pt
	// font, 60% opaque, slight upward offset so the line clears the
	// page margin. `scalefactor:1.0 abs` keeps the text at literal
	// font size rather than fitting it to page width.
	//
	// `scalefactor` and `rotation` are spelled out rather than using
	// the short prefix because `sc` collides with `strokecolor` and
	// `rot` with `rendermode` — pdfcpu rejects ambiguous prefixes.
	desc := "pos:bc, offset:0 20, points:9, opacity:0.6, rotation:0, scalefactor:1.0 abs"
	if err := pdfcpuapi.AddTextWatermarksFile(pdfPath, "", []string{lastPage}, true, text, desc, nil); err != nil {
		return fmt.Errorf("stamp colophon on %s: %w", pdfPath, err)
	}
	return nil
}

// Close shuts down the browser and Playwright.
func (g *Generator) Close() {
	g.browser.Close()
	g.pw.Stop()
}
