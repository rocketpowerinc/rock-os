---
name: dashboard-new
description: Use when the user invokes /dashboard-new or asks to create or scaffold an ordinary Rock-OS dashboard inside a profile-owned dashboards folder. Follow Rock-OS dashboard conventions, gather the owning profile, dashboard name, category, and local icon plan, then create the standard dashboard files. Use /profile-new instead for a profile workspace.
---

# Dashboard New

Create an ordinary Rock-OS dashboard under `Website/ENCRYPTED/Sessions/<SessionName>/Profiles/<ProfileName>/dashboards/<Category>/<DashboardName>/`.

## Required Inputs

Ask briefly for any missing values:

- Dashboard name. Prefer one word; ask before using a multi-word name.
- Owning session and profile. If missing, stop and ask; do not assume a default owner.
- Dashboard category. Do not use `Profiles`; hand profile requests to `/profile-new`.
- Icon source or visual direction. Keep the final asset local inside the dashboard.
- Dashboard theme or vibe. Ask what visual direction the dashboard should use, such as `Professional`, `Manly`, `Feminine`, `Nature`, `Cyberpunk`, `Construction`, `Space`, `Cozy`, `Minimal`, `Military`, `Retro`, or `Luxury`.

## Folder Convention

```text
Website/ENCRYPTED/Sessions/<SessionName>/Profiles/<ProfileName>/dashboards/<Category>/<DashboardName>/
  index.html
  Dashboard-Overview.md
  dashboard.json
  widgets.txt
  assets/
    <local-icon>
```

## Workflow

1. Read `AGENTS.md`, confirm `Website/ENCRYPTED/` is unlocked, and inspect a similar existing dashboard.
2. Reject `Profiles` as a dashboard category and use `/profile-new` for that request.
3. Inspect `Website/Sessions-State/sessions.json` and `cmd/rock-os/sessions.go` only if the user explicitly asks for dashboard-specific visibility rules.
4. Create the dashboard files by adapting the current dashboard convention.
5. Keep root-relative website paths and the shared module:

   ```html
   <script type="module" src="/js/profiles.js"></script>
   ```

6. Keep dashboard-specific assets under the dashboard's `assets/` folder. Do not use remote assets at runtime or place them in `Website/assets/`.
7. Add dashboard-specific visual theme rules to `Website/css/style.css` when requested. Model the structure after the Boys/Girls visual skins when useful: themed border, buttons, panel styling, background treatment, and local CSS assets. Keep the theme scoped to the dashboard page.
8. Add the dashboard avatar and landing-card icon rules to `Website/css/style.css`.
9. Dashboards inherit the owning profile's session visibility by default. If the user explicitly asks for dashboard-specific visibility rules, apply them server-side and add focused tests instead of only hiding a frontend card.
10. Update `documentation/Widgets.md` only when a widget type or supported field changes.
11. Run relevant checks:
    - `node --check Website\js\profiles.js` when JavaScript changes.
    - `go test ./...` from `cmd/rock-os` when server behavior changes.
    - `git diff --check`.

## Conventions

- Use `index.html`, `Dashboard-Overview.md`, `dashboard.json`, and `widgets.txt`.
- Preserve exact casing and punctuation when the user specifies it.
- Landing cards show only the icon and title.
- Dashboards are profile-owned. The profile workspace bar links to `dashboards.html?profile=<SessionName>/Profiles/<ProfileName>`.
- Dashboard themes are visual skins only. Do not create profile-like workspace tabs for dashboards; the top tabs remain the owning profile's standard workspace nav.
- Internal links open in the same tab; external links follow the existing new-tab behavior.
- Do not update `README.md` for an ordinary dashboard addition unless it introduces a new project convention.

## Safety

- Do not edit locked `git-crypt` content.
- Do not stage, commit, stash, or push.
- Do not remove existing dashboards or assets unless explicitly requested.
