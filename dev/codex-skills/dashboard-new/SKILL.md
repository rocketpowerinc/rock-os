---
name: dashboard-new
description: Use when the user invokes /dashboard-new or asks to create/scaffold a new Rock-OS dashboard under Website/dashboards. The skill follows Rock-OS dashboard conventions, asks for the dashboard category, dashboard name, and icon asset URL when missing, downloads the icon locally into the dashboard folder, and creates the standard dashboard files.
metadata:
  short-description: Scaffold a Rock-OS dashboard
---

# Dashboard New

Create a fresh Rock-OS dashboard under a category folder inside `Website/dashboards/` using the user's preferred conventions.

## Trigger

Use this skill when the user says `/dashboard-new`, asks to create a new dashboard, or asks to scaffold a dashboard under `Website/dashboards/`.

## Required Inputs

If missing, ask briefly for:

- Dashboard name, such as `Windows`, `Linux`, `Homelab`, or `Recovery`.
  Tell the user the name should be one word so URLs, folder names, CSS selectors,
  and dashboard routing stay clean.
- Dashboard category/directory under `Website/dashboards/`, such as `OS`,
  `Gaming`, `Homelab`, or a new one-word category. This determines which
  section title the dashboard appears under on `dashboards.html`.
- Icon/image URL to download for the dashboard icon.

If any of these are missing, ask for only the missing pieces before editing files.

## Folder Convention

Create this shape:

```text
Website/dashboards/<Category>/<DashboardName>/
  index.html
  Overview.md
  dashboard.json
  widgets.txt
  assets/
    <safe-icon-name>.<ext>
```

Rules:

- Use PascalCase or clean title case for `<DashboardName>` unless the user specifies exact casing.
- Prefer one-word dashboard names. If the user provides multiple words, warn
  that one word is preferred and ask whether they want a one-word version before
  scaffolding.
- Prefer one-word category names too. Existing categories are usually best when
  they fit.
- Keep dashboard-specific assets inside `Website/dashboards/<Category>/<DashboardName>/assets/`.
- Do not place dashboard icons under `Website/assets/`.
- Shared widget/feed fallback icons live under `Website/assets/widget-icons/` and are not dashboard-specific.
- Internal Rock-OS links open in the same tab. External links open in a new tab through existing app behavior.
- Dashboard/Profile landing cards should show only the item title and icon. Do not add subtitle text such as `Open local dashboard`.
- The Dashboards landing kicker should read `UNENCRYPTED DASHBOARDS` in all caps.
- Do not add descriptive paragraph text below the Profiles or Dashboards landing headings.
- Do not update `README.md` for every new dashboard unless the dashboard introduces a new convention or feature.

## Workflow

1. Inspect an existing dashboard, usually `Website/dashboards/OS/Windows/`, before editing.
2. Create the dashboard folder and `assets/` folder.
3. Download the icon URL into the dashboard `assets/` folder.
   - Prefer the original extension when obvious.
   - Use a safe lowercase filename such as `icon.png`, `windows.png`, or `<dashboard-name>.png`.
   - If network access is blocked, request approval to download the asset.
4. Create `index.html` by adapting the current dashboard page convention.
   - Use root-relative paths like `/css/style.css`, `/js/theme.js`, and `/js/profiles.js`.
   - Keep the same navbar and sidebar structure as other dashboard pages.
5. Create `Overview.md` with a short practical intro and a few useful sections.
6. Create `dashboard.json` with:
   - `title`: `<DashboardName> Command Center` unless user asks otherwise.
   - `subtitle`: a concise description.
   - `avatarClass`: a dashboard-specific class, for example `windows-dashboard-avatar-display`.
   - one starter `bookmarks` widget with useful internal links.
7. Create `widgets.txt` with comments explaining that it can override or add widgets later.
8. Update `Website/css/style.css`:
   - Add the avatar class pointing to `../dashboards/<Category>/<DashboardName>/assets/<icon>`.
   - Add/confirm a landing-card icon rule for `.profiles-card[data-profile="<DashboardName>"]` pointing to the same icon.
9. Run sanity checks:
   - `node --check Website\js\profiles.js` if JS changed.
   - `git diff --check`.
   - If Go server code changes, run `go test ./...` from `cmd/rock-os-wiki` and remind the user a new release binary is needed.

## Dashboard HTML Template Notes

Use the existing `Website/dashboards/OS/Windows/index.html` as the source pattern when it exists. Change only dashboard-specific text:

- `<title>Rock-OS <DashboardName> Dashboard</title>`
- sidebar heading
- search placeholder and aria label
- initial `<h1>`

Keep the script module as:

```html
<script type="module" src="/js/profiles.js"></script>
```

The shared `profiles.js` module detects dashboard mode automatically.

## Safety

- Do not put private or sensitive notes in Dashboards. Use Profiles for encrypted/private content.
- Do not stage, commit, stash, or push unless the user explicitly asks.
- Do not remove existing dashboards or assets unless the user explicitly asks.
