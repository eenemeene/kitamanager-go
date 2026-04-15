# KitaManager Report PDF Tool

A standalone tool that generates PDF reports from KitaManager's statistics pages. It uses Playwright to render the print-optimized pages in a headless browser and exports them as PDF.

The tool is **independent from the API and frontend** — it authenticates via HTTP and renders pages through the frontend. This means the reports include the exact same charts and tables that users see in the browser.

## Reports

The tool generates 4 report types, merged into a single PDF:

| Report | Content |
|--------|---------|
| **Children** | Enrollment trends, age distribution, contract properties |
| **Occupancy** | Monthly matrix by age group and care type, supplements |
| **Staffing** | Hours history, section balance, monthly grid, employee hours |
| **Financials** | Income/expenses, actual vs calculated funding, cumulative balance, budget overview |

## Quick Start

### One-shot mode

Generate a report directly:

```bash
# Build
cd tools/report-pdf && go build -o ../../bin/report-pdf .

# Generate all reports for org 1
./bin/report-pdf \
  --email superadmin@example.com \
  --password supersecret \
  --org-id 1 \
  --output-dir /tmp/reports
```

This produces:
- `report-1-2026.pdf` — combined report (all sections in one file)
- `children-1-2026.pdf`, `occupancy-1-2026.pdf`, etc. — individual reports

### Scheduled mode

Run as a long-lived service that sends reports via email on a schedule:

```bash
./bin/report-pdf --config report-config.yaml
```

See [report-config.example.yaml](report-config.example.yaml) for the configuration format.

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--email` | (required) | Login email for the KitaManager API |
| `--password` | (required) | Login password |
| `--org-id` | (required) | Organization ID to generate reports for |
| `--api-url` | `http://localhost:8080` | API server URL |
| `--base-url` | `http://localhost:3000` | Frontend URL (used for Playwright rendering) |
| `--year` | current year | Report year |
| `--output-dir` | `.` | Output directory for PDF files |
| `--reports` | `all` | Comma-separated: `children,occupancy,staffing,financials` |
| `--config` | — | Path to YAML config file (enables scheduled mode) |

## Docker

Build and run as a container:

```bash
docker build -f Dockerfile.report -t kitamanager-report .

# One-shot
docker run --rm -v /tmp/reports:/output kitamanager-report \
  --email superadmin@example.com \
  --password supersecret \
  --org-id 1 \
  --output-dir /output

# Scheduled
docker run -d -v ./report-config.yaml:/config/report-config.yaml:ro \
  kitamanager-report --config /config/report-config.yaml
```

## Requirements

- KitaManager API server running and accessible
- KitaManager frontend running and accessible
- Playwright-compatible environment (Chromium is installed automatically)
