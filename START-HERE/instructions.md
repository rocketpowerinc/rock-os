# START-HERE Instructions

This folder is the human-friendly control panel for Rock-OS.

Pick the folder for your operating system:

```text
START-HERE/
  Windows/
  Linux/
  Mac/
```

Use the scripts inside your platform folder. The scripts in each folder do the
same jobs, but use the correct file type and terminal behavior for that system.

## Quick Picks

| What you want to do                                             | Windows                                     | Linux                                    | Mac                                    |
| --------------------------------------------------------------- | ------------------------------------------- | ---------------------------------------- | -------------------------------------- |
| Install Rock-OS and create the `rock-os` command + Desktop Icon | `Windows/install-rock-os.ps1`               | `Linux/install-rock-os.sh`               | `Mac/install-rock-os.sh`               |
| Start Rock-OS normally                                          | `Windows/start-rock-os.cmd`                 | `Linux/start-rock-os.sh`                 | `Mac/start-rock-os.sh`                 |
| Start Rock-OS from Go source                                    | `Windows/start-rock-os-from-source.cmd`     | `Linux/start-rock-os-from-source.sh`     | `Mac/start-rock-os-from-source.sh`     |
| Start from Go source in LAN mode                                | `Windows/start-rock-os-from-source-lan.cmd` | `Linux/start-rock-os-from-source-lan.sh` | `Mac/start-rock-os-from-source-lan.sh` |
| Stop Rock-OS on port 8000                                       | `Windows/stop-rock-os.cmd`                  | `Linux/stop-rock-os.sh`                  | `Mac/stop-rock-os.sh`                  |
| Check repo and private markdown status                          | `Windows/repo-status.cmd`                   | `Linux/repo-status.sh`                   | `Mac/repo-status.sh`                   |
| Unlock private markdown                                         | `Windows/unlock-git-crypt.cmd`              | `Linux/unlock-git-crypt.sh`              | `Mac/unlock-git-crypt.sh`              |
| Re-lock private markdown                                        | `Windows/lock-git-crypt.cmd`                | `Linux/lock-git-crypt.sh`                | `Mac/lock-git-crypt.sh`                |

## Install Scripts

### `install-rock-os`

Use this when you are setting up Rock-OS on a computer.

What it does:

- Checks that Git is installed.
- Clones the Rock-OS repo into your home folder if it is not already there.
- If the repo already exists, updates it with a safe fast-forward pull.
- Creates a `rock-os` terminal command.
- Creates a desktop launcher icon where the operating system supports it.
- Starts Rock-OS when the install finishes.

Run it from the internet with the one-liners in the main README, or run the
local platform script after cloning the repo.

## Start Scripts

### `start-rock-os`

Use this for normal everyday startup.

What it does:

- Checks that you are in a real Git clone, not a GitHub ZIP download.
- Pulls safe repo updates with `git pull --ff-only` and shows the real Git
  output so you can see what changed.
- Detects your operating system and CPU architecture.
- Downloads the latest matching release binary into `Website/` when internet is
  available.
- Starts the release binary when it exists.
- Falls back to Go source only if no release binary is available.
- Checks whether `git-crypt` is installed.
- Reports whether the Private markdown folder appears locked or unlocked.

After launch, the Go server prints a colored startup checklist and request log
in the terminal window. That output is normal and useful when checking LAN
access or debugging a page.

Default mode is local-only. Other computers cannot connect unless you start LAN
mode.

To start in LAN mode, pass `lan`:

```bash
./START-HERE/Linux/start-rock-os.sh lan
```

On Windows:

```powershell
.\START-HERE\Windows\start-rock-os.cmd lan
```

Only use LAN mode on a trusted private network.

### `start-rock-os-from-source`

Use this for development or troubleshooting when you want to skip release
binaries and run the current Go source.

What it does:

- Builds the Go server from `cmd/rock-os-wiki/`.
- Places a temporary dev binary in `Website/`.
- Starts that dev binary against the real `Website/` folder.

This is useful after changing Go server code and before making a new release.
It requires Go to be installed.

### `start-rock-os-from-source-lan`

This is the LAN version of `start-rock-os-from-source`.

Use it only when you intentionally want other trusted devices on your home
network to connect to the source-built server. It is not for public Wi-Fi,
guest networks, hotels, schools, or coffee shops.

## Stop Scripts

### `stop-rock-os`

Use this when Rock-OS is already running and you want to stop it.

What it does:

- Looks for a process listening on port `8000`.
- Stops that process.
- Accepts a different port number if you started Rock-OS on another port.

Example:

```bash
./START-HERE/Linux/stop-rock-os.sh 8001
```

On Windows:

```powershell
.\START-HERE\Windows\stop-rock-os.cmd 8001
```

## Status Scripts

### `repo-status`

Use this when you want one quick health report.

What it checks:

- Git version, branch, current commit, and working tree changes.
- Whether a release binary exists.
- Whether Go is installed.
- Whether port `8000` is already in use.
- Whether `git-crypt` is installed.
- Whether Private markdown appears locked or unlocked.
- Whether a `.key` file is sitting in the repo root.
- Full `git-crypt status` output.

This script does not commit, stash, or fix anything. It only reports.

## Private Markdown Scripts

Private markdown lives here:

```text
Website/markdown/Private/
```

That folder is intended to be encrypted with `git-crypt`.

### `unlock-git-crypt`

Use this after cloning the repo on a trusted computer when Private markdown is
still encrypted.

How it works:

- Looks for exactly one `.key` file in the repo root.
- Copies the key to a temporary system folder.
- Removes the root key copy before unlocking so the working tree stays clean
  enough for `git-crypt unlock`.
- Runs `git-crypt unlock`.
- Copies the key back to the repo root afterward.

The restored `.key` file is ignored by Git. Keep it private and never commit it.

### `lock-git-crypt`

Use this when you want to re-lock Private markdown.

What it does:

- Runs `git-crypt lock` from the repo root.
- Leaves the repo locked again so Private markdown is encrypted on disk.

If locking fails, close open Private markdown files and make sure you do not
have pending private edits that block `git-crypt`.

## Folder Layout

The top level of `START-HERE/` is intentionally kept clean. Use the folder that
matches your operating system:

```text
START-HERE/Windows/
START-HERE/Linux/
START-HERE/Mac/
```

## Notes For Linux And Mac

Shell scripts should be executable in normal Git clones. If they are not,
repair permissions with:

```bash
chmod +x ./START-HERE/Linux/*.sh ./START-HERE/Mac/*.sh
```

This avoids chmod-only Git changes that can dirty the repo and get in the way of
`git-crypt unlock`.
