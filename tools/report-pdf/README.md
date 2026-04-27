# KitaManager Report PDF Tool

A standalone tool that generates PDF reports from KitaManager's statistics pages. It uses Playwright to render the print-optimized pages in a headless browser and exports them as PDF.

The tool is **independent from the API and frontend** — it authenticates via HTTP and renders pages through the frontend. This means the reports include the exact same charts and tables that users see in the browser.

The tool is **one-shot**: it logs in, generates reports, writes them to disk, and exits. Recurring delivery (weekly/monthly emails to stakeholders) is the responsibility of an external scheduler — see [External scheduling](#external-scheduling) below.

## Reports

The tool generates 4 report types, merged into a single PDF:

| Report | Content |
|--------|---------|
| **Children** | Enrollment trends, age distribution, contract properties |
| **Occupancy** | Monthly matrix by age group and care type, supplements |
| **Staffing** | Hours history, section balance, monthly grid, employee hours |
| **Financials** | Income/expenses, actual vs calculated funding, cumulative balance, budget overview |

## Quick Start

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

## Docker

Build and run as a container:

```bash
docker build -f Dockerfile.report -t kitamanager-report .

docker run --rm -v /tmp/reports:/output kitamanager-report \
  --email superadmin@example.com \
  --password supersecret \
  --org-id 1 \
  --output-dir /output
```

## External scheduling

To send reports on a schedule (e.g. monthly to a stakeholder group), wrap the binary in an external scheduler that handles "when to run" plus delivery. The tool intentionally has no built-in cron, retry, or email logic — those concerns are better solved by mature, system-native tools.

### cron

Run on the 1st of every month at 06:00, attach the PDF to an email:

```cron
0 6 1 * * /usr/local/bin/report-pdf \
    --email reports@example.com --password "$REPORT_PASSWORD" \
    --org-id 1 --output-dir /var/reports \
  && mailx -s "Monthly KitaManager report" \
       -a /var/reports/report-1-$(date +\%Y).pdf \
       boss@example.com < /dev/null
```

### systemd timer

`/etc/systemd/system/kitamanager-report.service`:

```ini
[Unit]
Description=Generate KitaManager monthly report

[Service]
Type=oneshot
EnvironmentFile=/etc/kitamanager/report.env
ExecStart=/usr/local/bin/report-pdf --org-id 1 --output-dir /var/reports
ExecStartPost=/usr/local/bin/email-report.sh
```

`/etc/systemd/system/kitamanager-report.timer`:

```ini
[Unit]
Description=Run KitaManager report monthly

[Timer]
OnCalendar=*-*-01 06:00:00
Persistent=true

[Install]
WantedBy=timers.target
```

`Persistent=true` makes systemd run any missed firings after a downtime, which the previous in-process scheduler did not handle.

### Kubernetes CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: kitamanager-report
spec:
  schedule: "0 6 1 * *"
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: OnFailure
          containers:
            - name: report
              image: ghcr.io/eenemeene/kitamanager-report:latest
              args:
                - --email=$(REPORT_EMAIL)
                - --password=$(REPORT_PASSWORD)
                - --org-id=1
                - --output-dir=/output
              env:
                - name: REPORT_EMAIL
                  valueFrom:
                    secretKeyRef:
                      name: kitamanager-report-creds
                      key: email
                - name: REPORT_PASSWORD
                  valueFrom:
                    secretKeyRef:
                      name: kitamanager-report-creds
                      key: password
              volumeMounts:
                - name: output
                  mountPath: /output
          volumes:
            - name: output
              persistentVolumeClaim:
                claimName: kitamanager-reports
```

Pair with a sidecar (or a follow-on job) that uploads to S3 / sends email with the resulting PDF.

## Requirements

- KitaManager API server running and accessible
- KitaManager frontend running and accessible
- Playwright-compatible environment (Chromium is installed automatically)
