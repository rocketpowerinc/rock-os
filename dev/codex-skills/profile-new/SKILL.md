---
name: profile-new
description: Use when the user invokes /profile-new or asks to create or scaffold a new Rock-OS profile. Create a special profile workspace under Website/ENCRYPTED/Profiles with its own Dashboards, Bookmarks, Cheatsheets, Dotfiles, Bootstraps, Scripts, and Wiki workspace, local theme assets, and server-enforced session visibility.
---

# Profile New

Create a Rock-OS profile workspace under `Website/ENCRYPTED/Profiles/<ProfileName>/`.

## Required Inputs

Ask briefly for any missing values:

- Profile name. Prefer one word; ask before using a multi-word name.
- Visual direction or asset source for the profile's Steel, Rugged, Cyberpunk, and Blue-Grass icons. Example: "make one in the same low-poly style" or "use this local image."
- Sessions that should or should not see the profile. Accept the current default session behavior when the user has no special requirement.

## Folder Convention

```text
Website/ENCRYPTED/Profiles/<ProfileName>/
  index.html
  Overview.md
  dashboard.json
  widgets.txt
  assets/
    <ProfileName>-Steel.<ext>
    <ProfileName>-Rugged.<ext>
    <ProfileName>-Cyberpunk.<ext>
    <ProfileName>-Blue-Grass.<ext>
  dashboards/
    .gitkeep
  bookmarks/
    .gitkeep
  cheatsheets/
    .gitkeep
  dotfiles/
    .gitkeep
  bootstraps/
    .gitkeep
  scripts/
    .gitkeep
  wiki/
    .gitkeep
```

## Workflow

1. Read `AGENTS.md`, confirm `Website/ENCRYPTED/` is unlocked, and inspect a similar existing profile.
2. Inspect `Website/Sessions/sessions.json`, `cmd/rock-os/sessions.go`, `Website/js/index.js`, and `Website/js/profiles.js` before changing visibility or ordering.
3. Create the profile dashboard files by adapting the current profile page convention.
4. Create all seven workspace folders, even when empty: Dashboards, Bookmarks, Cheatsheets, Dotfiles, Bootstraps, Scripts, and Wiki. Use **Bootstraps** for setup playbooks.
5. Keep the profile page on the shared dashboard module:

   ```html
   <script type="module" src="/js/profiles.js"></script>
   ```

   Do not hardcode the horizontal workspace navigation; `profiles.js` injects it through `profile-workspace.js`.
6. Keep every profile asset local inside the profile's `assets/` folder. Match the established theme-aware profile icon style and do not depend on remote assets at runtime.
7. Add the profile avatar and landing-card icon rules to `Website/css/style.css`.
8. Apply requested session visibility server-side so the home profile cards, Dashboards landing page, profile files, markdown APIs, and scripts agree. Add or update focused Go tests when filtering changes.
9. Keep profile-owned scripts under the profile's `scripts/` folder and only use supported script extensions: `.cmd`, `.bat`, `.sh`, `.ps1`.
10. Update `documentation/Widgets.md` only when a widget type or supported field changes.
11. Run relevant checks:
    - `node --check Website\js\profiles.js` and `node --check Website\js\index.js` when JavaScript changes.
    - `go test ./...` from `cmd/rock-os` when server behavior changes.
    - `git diff --check`.

## Conventions

- Profiles are special workspaces, not ordinary dashboard categories. Each profile owns its own `dashboards/` folder.
- Preserve exact casing and punctuation when the user specifies it.
- Landing cards show only the icon and title.
- Internal links open in the same tab; external links follow the existing new-tab behavior.
- Do not add a global Menu button.
- Do not update `README.md` for an ordinary profile addition unless it introduces a new project convention.

## Safety

- Do not edit or restructure locked `git-crypt` content.
- Do not stage, commit, stash, or push.
- Do not remove existing profiles or assets unless explicitly requested.
