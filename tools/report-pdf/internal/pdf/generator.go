package pdf

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

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
func (g *Generator) GenerateReport(reportType, orgID string, year int, outputDir string) error {
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

	pageURL := fmt.Sprintf("%s/organizations/%s/statistics/%s/print?year=%d", g.baseURL, orgID, reportType, year)
	fmt.Printf("  Navigating to %s\n", pageURL)

	resp, err := page.Goto(pageURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(30000),
	})
	if err != nil {
		return fmt.Errorf("navigate to %s: %w", pageURL, err)
	}

	// Check if we were redirected to login (auth failure)
	finalURL := page.URL()
	if strings.Contains(finalURL, "/login") {
		return fmt.Errorf("redirected to login page — authentication failed")
	}

	if resp != nil && resp.Status() >= 400 {
		return fmt.Errorf("page returned HTTP %d", resp.Status())
	}

	// Wait for data-print-ready="true" attribute
	_, err = page.WaitForSelector("[data-print-ready='true']", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(15000),
	})
	if err != nil {
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

	filename := fmt.Sprintf("%s-%s-%d.pdf", reportType, orgID, year)
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

// Close shuts down the browser and Playwright.
func (g *Generator) Close() {
	g.browser.Close()
	g.pw.Stop()
}
