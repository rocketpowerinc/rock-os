---
name: dashboard-new
description: Use when the user invokes /dashboard-new or asks to create or scaffold an ordinary Rock-OS dashboard under Website/ENCRYPTED/dashboards outside the Profiles category. Follow Rock-OS dashboard conventions, gather the dashboard name, category, local icon plan, and session visibility requirements, then create the standard dashboard files. Use /profile-new instead for a profile workspace.
---

# Dashboard New

Create an ordinary Rock-OS dashboard under `Website/ENCRYPTED/dashboards/<Category>/<DashboardName>/`.

## Required Inputs

Ask briefly for any missing values:

- Dashboard name. Prefer one word; ask before using a multi-word name.
- Dashboard category. Do not use `Profiles`; hand profile requests to `/profile-new`.
- Icon source or visual direction. Keep the final asset local inside the dashboard.
- Sessions that should not see the dashboard. Accept `none`.

## Folder Convention

```text
Website/ENCRYPTED/dashboards/<Category>/<DashboardName>/
  index.html
  Overview.md
  dashboard.json
  widgets.txt
  assets/
    <local-icon>
```

## Workflow

1. Read `AGENTS.md`, confirm `Website/ENCRYPTED/` is unlocked, and inspect a similar existing dashboard.
2. Reject `Profiles` as a dashboard category and use `/profile-new` for that request.
3. Inspect `Website/Sessions/sessions.json` and `cmd/rock-os/sessions.go` before changing visibility.
4. Create the dashboard files by adapting the current dashboard convention.
5. Keep root-relative website paths and the shared module:

   ```html
   <script type="module" src="/js/profiles.js"></script>
   ```

6. Keep dashboard-specific assets under the dashboard's `assets/` folder. Do not use remote assets at runtime or place them in `Website/assets/`.
7. Add the dashboard avatar and landing-card icon rules to `Website/css/style.css`.
8. Apply requested session visibility server-side. If the current session model cannot express it, update the filtering code and focused tests instead of only hiding a frontend card.
9. Update `documentation/Widgets.md` only when a widget type or supported field changes.
10. Run relevant checks:
    - `node --check Website\js\profiles.js` when JavaScript changes.
    - `go test ./...` from `cmd/rock-os` when server behavior changes.
    - `git diff --check`.

## Conventions

- Use `index.html`, `Overview.md`, `dashboard.json`, and `widgets.txt`.
- Preserve exact casing and punctuation when the user specifies it.
- Landing cards show only the icon and title.
- The Dashboards landing kicker remains `ENCRYPTED DASHBOARDS`.
- Internal links open in the same tab; external links follow the existing new-tab behavior.
- Do not update `README.md` for an ordinary dashboard addition unless it introduces a new project convention.

## Safety

- Do not edit locked `git-crypt` content.
- Do not stage, commit, stash, or push.
- Do not remove existing dashboards or assets unless explicitly requested.
