---
title: Enable two-factor authentication for your team
weight: 3
---

This tutorial walks you through making 2FA the default for everyone who can edit data in your KitaManager organisation. By the end, every admin and manager account has TOTP enrolled and you have a documented process for new starters.

You'll need:

- An admin account in your organisation.
- About 15 minutes of your own time, plus 5 minutes per teammate (most of which they do themselves).

## Step 1 — Enable 2FA on your own account first

Lead by example. Walk through [Enable two-factor authentication](../../how-to/use/enable-2fa/) on your own account end-to-end, including:

- Scanning the QR with the authenticator app you'll recommend (Google Authenticator, 1Password, Authy, Bitwarden — pick one and standardise).
- Saving the recovery codes somewhere you'll be able to find them in 18 months.
- Optionally adding a security key.

Sign out and sign back in to verify the prompt works as expected.

## Step 2 — Plan the rollout

For each teammate you'll need:

- Their working email and password (they already have these).
- A short window when they're at a screen and not in the middle of urgent work.
- A communication channel: in-person, video call, Slack DM. **Not email.**

Decide:

- **Which authenticator app** the team will use. Standardising means you can give one set of instructions and one trouble-shooting path.
- **Where recovery codes go.** Options: each user keeps theirs in a personal password manager (best); or printed and stored in a locked drawer (ok); or pasted to a shared password vault entry per user (acceptable but visible to whoever else has access).
- **What "lost both authenticator and codes" looks like.** Today the only recovery is to delete + recreate the account. Tell users this so they take the codes seriously.

## Step 3 — Enrol each teammate

For each user:

1. Sit with them (in person or screenshare) on **Settings → Two-factor authentication**.
2. Walk them through [Enable two-factor authentication](../../how-to/use/enable-2fa/) step by step.
3. **Verify with them** that the recovery codes are saved somewhere they'll find later. Watch them save them. If they say "I'll do it later", stay until they do.
4. Sign them out and back in to confirm the next sign-in prompts for the code.
5. Note the date they enrolled (e.g. in your team wiki) so you have a record.

## Step 4 — Document the process for new starters

Add a step to your onboarding for any new admin/manager:

> Day 1, after first sign-in:
> 1. Change your initial password.
> 2. Enable 2FA following [Enable two-factor authentication](../../how-to/use/enable-2fa/).
> 3. Save recovery codes in your password manager.
> 4. Confirm with your team lead.

## Step 5 — Periodic audit

Once a quarter, scan your **Users** list for anyone with edit rights but no enrolled factor. There's no "users without 2FA" report yet — manual scanning of `/api/v1/users/{id}/factors` for each admin/manager covers it.

If you find a stale account (enrolled, but the user no longer has access to the device), reset their password to force them to sign in again, and re-walk them through enrolment.

## You're done

Your team's accounts are protected by something the user has, in addition to something they know. A leaked password alone is no longer enough to compromise your Kita's data.

Next:

- Make sure backups are working — see [Back up and restore](../../how-to/operate/back-up-and-restore/).
- Periodically review the [audit log](../../how-to/administer/review-audit-log/) for anomalies.
- Keep the [funding rates](../../how-to/operate/update-government-funding-rates/) current so reconciliation stays accurate.
