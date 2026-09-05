# Monitoring & Operations

Vitals, reviews, reports, team management, and observability.

## Vitals & quality

```bash
# Crash reports
gplay vitals crashes clusters --package com.example.app
gplay vitals crashes reports --package com.example.app

# Performance metrics
gplay vitals performance startup --package com.example.app
gplay vitals performance rendering --package com.example.app
gplay vitals performance battery --package com.example.app

# ANR / error tracking (defaults to the last 24 hours)
gplay vitals errors issues --package com.example.app
gplay vitals errors reports --package com.example.app

# Error tracking over a date range (UTC, inclusive)
gplay vitals errors issues --package com.example.app --from 2025-01-01 --to 2025-01-31
```

## Reviews

```bash
# List and filter reviews
gplay reviews list --package com.example.app
gplay reviews list --package com.example.app --paginate

# Reply to reviews
gplay reviews get --package com.example.app --review-id <id>
gplay reviews reply --package com.example.app --review-id <id> --text "Thank you!"
```

## Financial & statistics reports

Reports are stored as CSV/ZIP files in Google Cloud Storage buckets (`pubsite_prod_rev_<developer_id>`). The service account must have access to the GCS bucket (granted automatically when added to Play Console).

> **Important:** The `--developer` ID for reports is **not** the developer ID in your Play Console URL. To find the correct ID, go to **Play Console > Download reports > Copy Cloud Storage URI**. The URI looks like `gs://pubsite_prod_rev_XXXX/` — the number after `pubsite_prod_rev_` is your developer ID.

```bash
# Financial reports (earnings, sales, payouts)
gplay reports financial list --developer <id>
gplay reports financial list --developer <id> --type earnings --from 2026-01 --to 2026-06
gplay reports financial download --developer <id> --from 2026-01 --type earnings --dir ./reports

# Statistics reports (installs, ratings, crashes, store_performance, subscriptions)
gplay reports stats list --developer <id>
gplay reports stats list --developer <id> --package com.example.app --type installs
gplay reports stats download --developer <id> --package com.example.app --from 2026-01 --type installs --dir ./reports
```

## Weekly and daily insights

`gplay insights` compares official report files locally. It does not resolve
credentials, contact Google, or mutate an account. Google documents the CSV
fields and monthly Cloud Storage layout in [Download and export monthly
reports](https://support.google.com/googleplay/android-developer/answer/6135870).

Supply one breakdown file per report type so dimension totals are not counted
twice. UTF-8 and Google Play's UTF-16 exports are supported. A missing source,
date window, or metric column is returned as `unavailable` rather than inferred.

```bash
# Monday through Sunday compared with the preceding week
gplay insights weekly \
  --package com.example.app \
  --week 2026-08-17 \
  --installs-file ./reports/installs_com.example.app_202608_country.csv \
  --crashes-file ./reports/crashes_com.example.app_202608_device.csv \
  --store-performance-file ./reports/store_performance_com.example.app_202608_country.csv

# One UTC report date compared with the preceding date
gplay insights daily \
  --package com.example.app \
  --date 2026-08-24 \
  --crashes-file ./reports/crashes_com.example.app_202608_device.csv \
  --output table
```

## Team & permissions

```bash
# Manage developer account users
gplay users list --developer <id>
gplay users create --developer <id> --email user@example.com --json @permissions.json
gplay users delete --developer <id> --email user@example.com --confirm

# Manage per-app grants
gplay grants create --developer <id> --email user@example.com --package com.example.app --json @grant.json
gplay grants update --developer <id> --email user@example.com --package com.example.app --json @grant.json
gplay grants delete --developer <id> --email user@example.com --package com.example.app --confirm
```

## Notifications

```bash
# Send webhook notifications (Slack, Discord, generic)
gplay notify send --webhook-url https://hooks.slack.com/... --message "Deploy complete" --format slack
gplay notify send --webhook-url https://discord.com/... --message "New release" --format discord
```

## Diagnostics & observability

```bash
# Full environment health check (16 checks: gcloud, config, SA, DNS, disk, clock, ...)
gplay doctor
gplay doctor --output json --pretty

# Local audit log of every invocation (auto-written to ~/.gplay/audit.log)
gplay audit list --limit 50
gplay audit search --command vitals --status error
gplay audit clear --confirm

# API quota usage (derived from audit log)
gplay quota status                 # daily + per-minute windows
gplay quota status --top 10
```
