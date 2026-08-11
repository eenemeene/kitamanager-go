---
title: First day in KitaManager
weight: 1
aliases:
  - /docs/user-guide/
---

This tutorial takes you from "I just got an account" to "I can use KitaManager confidently for my Kita's daily work" in about 30 minutes. You'll log in, walk the dashboard, record an attendance day, look at a child's contract, upload a fake ISBJ bill, and read your first report.

You'll need:

- A running KitaManager instance — if you don't have one, do [Deploy KitaManager](../deploy-kitamanager/) first.
- An admin or manager account in an organisation that has the **Kita Sonnenschein** seed data loaded (any local-dev install does, by default).
- About 30 minutes.

When you're done you should be able to find your way around without referring to docs, and you should know which [how-to guide](../../how-to/use/) to consult when a specific task comes up.

## Step 1 — Log in

1. Open KitaManager in your browser.
2. Enter your email and password and click **Sign in**.

If your account has two-factor authentication enabled, enter the code from your authenticator app or insert your security key when prompted. If you've never set up 2FA, the [Enable two-factor authentication](../../how-to/use/enable-2fa/) recipe walks through it; it's strongly recommended for any account that can edit data.

After signing in, you land on the **Dashboard**. This is the heart of KitaManager: every important warning shows up here.

## Step 2 — Walk the dashboard

The dashboard is organised top to bottom by urgency: stat cards (KPIs for the current month), then warning cards that only appear when there's something to fix, then routine widgets for upcoming work. Hover any card title for a one-line explanation.

**Action:** spend two minutes reading the descriptions on each card. If your dashboard has any warnings (the seeded "Kita Sonnenschein" should have a few mismatches), you've found your first task.

## Step 3 — Record an attendance day

1. Click **Attendance** in the left sidebar.
2. You'll see a weekly grid: rows are children with active care contracts, columns are weekdays.
3. Tap (or click) any cell to mark the child present, absent, or to clear the mark. The save happens automatically — no Save button.
4. Use the arrows at the top of the page to flip to a different week.

The summary at the top shows you, for the selected week, how many children were present each day. If you wanted to look at a single child's attendance history, you'd open their detail page; the Attendance widget is for the wide view.

## Step 4 — Look at a child's care contract

1. Click **Children** in the sidebar.
2. The list shows all enrolled children with their current funding amount and any billing differences. Spot a child whose name catches your eye and click them.
3. Scroll down to the **Contracts** section. You'll see one or more care contracts — each has a from/to date, a section, a Gutscheinnummer, a care type, and any supplements (NdH, QM/MSS, Integration).

Hover over a contract row to see the calculated monthly funding. If you don't recognise the supplements, [How funding works in Berlin](../../explanation/how-funding-works-in-berlin/) is the canonical explanation — read it once and you'll never have to look the abbreviations up again.

**Action:** compare what's in KitaManager with what your last paper Bescheid says. Same Gutscheinnummer? Same care type? Same supplements? If yes, this child should reconcile; if no, you've identified a fix.

## Step 5 — Open an ISBJ bill comparison

The seed data already includes a sample ISBJ Bescheid so you don't need a real Excel to see the comparison flow.

1. Click **Funding Bills** in the sidebar.
2. Click the most recent bill row.
3. The detail page shows a per-child comparison: each row is a child, with two columns for KitaManager's calculated amount and the ISBJ amount, plus a status (match / different / missing from bill / extra in bill).

The seeded data deliberately includes a few mismatches. Click any red row to see which contract property differs. The next step would be [Investigate a bill discrepancy](../../how-to/use/investigate-a-bill-discrepancy/) — but for the tutorial, just confirm you can read the comparison.

(For uploading your own monthly Bescheid, see [Upload a funding bill](../../how-to/use/upload-an-isbj-bill/).)

## Step 6 — Read your first report

Click **Statistics** in the sidebar and pick **Staffing Hours**.

The staffing-hours chart shows two lines: how many staff hours your children require (computed from per-child FTE requirements set by the funding configuration), and how many you actually have (computed from active employee contracts). If the available line is below the required line, you're understaffed for the period.

For a fuller tour of every chart, see the [report-printing recipe](../../how-to/use/print-a-report/) — you can print any report to take it to a board meeting.

## You're done

You've logged in, found the dashboard's structure, recorded attendance, inspected a contract, uploaded a bill, and read a report. The day-to-day shape of KitaManager is now familiar.

Where to go from here:

- For a specific task, jump to the matching [how-to guide](../../how-to/use/).
- To understand *why* funding works the way it does, [How funding works in Berlin](../../explanation/how-funding-works-in-berlin/) is the most useful 15 minutes of reading you'll spend.
- If you're an admin and need to add a colleague's account, see [Create a user](../../how-to/administer/create-a-user/).
