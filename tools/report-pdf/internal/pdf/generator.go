package pdf

import (
	"fmt"
	"net/url"
	"path/filepath"
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

// printPageURL builds the URL of the print-optimised statistics page
// for a given org + report type. Extracted so the URL contract can be
// pinned by unit tests without spinning up Playwright — the path
// shape and `month` query parameter are part of the API/frontend
// contract a renaming on either side would silently break. month is
// passed through verbatim (already validated to YYYY-MM at the CLI
// layer); we don't re-validate here so the helper stays a pure
// formatter.
func printPageURL(baseURL, orgID, reportType, month string) string {
	return fmt.Sprintf("%s/organizations/%s/statistics/%s/print?month=%s", baseURL, orgID, reportType, month)
}

type Generator struct {
	pw      *playwright.Playwright
	browser playwright.Browser
	cookies []playwright.OptionalCookie
	baseURL string
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

// GenerateReport navigates to a print page and exports it as a PDF.
// month is the YYYY-MM form of the report month — passed verbatim into
// the print page's `?month=` query so every API call the page makes is
// scoped to the same period.
func (g *Generator) GenerateReport(reportType, orgID, month, outputDir string) error {
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

	pageURL := printPageURL(g.baseURL, orgID, reportType, month)
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
	err = page.Locator("[data-print-ready='true']").WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(30000),
	})
	if err != nil {
		if isLoginBounce(page.URL()) {
			return fmt.Errorf("page bounced to %s mid-render — session expired during PDF generation", page.URL())
		}
		return fmt.Errorf("timeout waiting for page to be ready: %w", err)
	}

	// Inject print-optimized CSS:
	// - Remove max-width so wide tables aren't clipped by the container
	// - Remove body margin so content uses the full paper width
	// - Ensure overflow is visible everywhere
	if _, err := page.AddStyleTag(playwright.PageAddStyleTagOptions{
		Content: playwright.String(`
			body { margin: 0 !important; padding: 0 !important; }
			[data-print-ready] {
				max-width: none !important;
				overflow: visible !important;
				padding: 0 20px !important;
			}
			table { overflow: visible !important; }
		`),
	}); err != nil {
		return fmt.Errorf("inject print CSS: %w", err)
	}

	// Brief stabilization delay for chart animations
	time.Sleep(1 * time.Second)

	filename := fmt.Sprintf("%s-%s-%s.pdf", reportType, orgID, month)
	outputPath := filepath.Join(outputDir, filename)

	marginMM := "10mm"
	_, err = page.PDF(playwright.PagePdfOptions{
		Path:            playwright.String(outputPath),
		Landscape:       playwright.Bool(true),
		PrintBackground: playwright.Bool(true),
		Format:          playwright.String("A4"),
		Scale:           playwright.Float(0.55),
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

	fmt.Printf("  Saved %s\n", outputPath)
	return nil
}

// MergeFiles combines multiple PDF files into a single output file.
func MergeFiles(inputPaths []string, outputPath string) error {
	return pdfcpuapi.MergeCreateFile(inputPaths, outputPath, false, nil)
}

// AddProperties adds custom properties to an existing PDF file.
func AddProperties(pdfPath string, properties map[string]string) error {
	return pdfcpuapi.AddPropertiesFile(pdfPath, "", properties, nil)
}

// Close shuts down the browser and Playwright.
func (g *Generator) Close() {
	g.browser.Close()
	g.pw.Stop()
}
