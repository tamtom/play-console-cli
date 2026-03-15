# Play Console Tracks

## Overview

Google Play Console uses "tracks" to manage different release channels. Each track represents a stage in your release pipeline.

## Built-in Tracks

### Internal Testing
- **Track name**: `internal`
- **Purpose**: Quick internal testing with up to 100 testers
- **Review**: No Google review required
- **Availability**: Available to internal testers within minutes

### Closed Testing (Alpha)
- **Track name**: `alpha`
- **Purpose**: Testing with a limited group of testers
- **Review**: Subject to Google review
- **Availability**: Available to opted-in testers

### Open Testing (Beta)
- **Track name**: `beta`
- **Purpose**: Public beta testing
- **Review**: Subject to Google review
- **Availability**: Available to anyone who opts in

### Production
- **Track name**: `production`
- **Purpose**: Public release to all users
- **Review**: Subject to Google review
- **Availability**: Available to everyone

## Custom Tracks

You can create custom closed testing tracks in Play Console for more granular testing.

## Release Statuses

| Status | Description |
|--------|-------------|
| `draft` | Release is being prepared, not yet sent for review |
| `inProgress` | Release is actively rolling out |
| `halted` | Rollout has been paused |
| `completed` | Release is fully rolled out |

## Staged Rollouts

Production and open testing tracks support staged rollouts:

```bash
# Start with 1% rollout
gplay release --track production --bundle app.aab --rollout 0.01

# Increase to 10%
gplay rollout update --track production --rollout 0.1

# Increase to 50%
gplay rollout update --track production --rollout 0.5

# Complete rollout (100%)
gplay rollout complete --track production
```

## Common Commands

```bash
# List all tracks
gplay tracks list --package com.example.app

# Get a specific track
gplay tracks get --package com.example.app --track production

# Promote between tracks
gplay promote --package com.example.app --from internal --to beta
```
