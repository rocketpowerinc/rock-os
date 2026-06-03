---
name: release-new
description: Use when the user invokes /release-new, says "Release new", or asks to publish a new Rock-OS server release. Run the project's Windows release helper in publish mode so it asks the user for the version, validates changes, creates the release commit, builds binaries, pushes, and publishes the GitHub release.
---

# Release New

Publish a reviewed Rock-OS server release through the repo-owned helper. This is
the only Rock-OS workflow allowed to stage, commit, and push without a separate
confirmation.

## Workflow

1. Work from the Rock-OS repo root.
2. Inspect `git status --short` and `git diff --check`.
3. Confirm the pending work includes a meaningful server-side change under
   `cmd/rock-os/`, or that the user explicitly wants a release despite only
   web/content changes.
4. Run the release helper outside the sandbox in a visible PowerShell window and
   wait for it to finish. Use `-NoProfile` so user profile startup scripts do
   not break the release helper, and use `-WindowStyle Normal` so the version
   prompt is actually visible:

```powershell
Start-Process `
  -FilePath 'C:\Program Files\PowerShell\7\pwsh.exe' `
  -ArgumentList '-NoProfile', '-NoExit', '-ExecutionPolicy', 'Bypass', '-File', '.\dev\windows-create-release.ps1', '-Publish' `
  -WorkingDirectory (Get-Location) `
  -WindowStyle Normal `
  -Wait
```

5. If Codex requires approval to launch that visible PowerShell process outside
   the sandbox, request it and explain that the release helper needs a visible
   version prompt and normal Go/PowerShell cache access.
6. Let the script ask the user for the version number. Do not ask additional
   release questions in chat.
7. After the PowerShell window closes, inspect `git status --short --branch` and
   report the release result.

## Script Contract

`dev/windows-create-release.ps1 -Publish` owns the release transaction:

- Prompt for the version.
- Refuse pre-existing staged changes.
- Stage pending source changes.
- Reject secrets and generated artifacts.
- Run whitespace checks and Go tests.
- Commit with `release: prepare vX.Y`.
- Build six cross-platform binaries and checksums under ignored `.release/`.
- Push the current branch and publish the GitHub release.

## Safety

- Do not manually stage, commit, push, or create a release outside the helper.
- Do not pass `-Version`; keep the version prompt inside the visible script.
- Do not move, rename, or restructure locked `Website/ENCRYPTED/` ciphertext.
- Stop and report any helper failure. Do not bypass its checks.
