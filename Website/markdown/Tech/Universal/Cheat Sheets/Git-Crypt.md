# Git-Crypt Private Markdown Notes

`git-crypt` lets you keep sensitive files inside a public Git repository while storing them encrypted in GitHub. On your own computer, after the repo is unlocked, the files look and edit like normal files.

For this project, the private markdown folder is:

```powershell
Website\markdown\Private
```

Anything in that folder is intended to be encrypted before it leaves your machine.

## The Big Rule

Do not commit your exported `git-crypt` key.

That key is what unlocks the encrypted files. Keep it somewhere private, backed up, and outside the repository. If the key is lost and you do not have another unlocked clone, the encrypted files may be impossible to recover.

## One-Time Setup

Run these commands from the root of the repository:

```powershell
cd C:\Users\rocket\Github-pwr\rock-os
```

### 1. Install git-crypt

On Windows with Scoop:

```powershell
scoop install git-crypt
```

Check that it works:

```powershell
git-crypt --version
```

On Linux or macOS, install it with your system package manager.

Linux examples:

```bash
sudo apt install git-crypt
sudo dnf install git-crypt
sudo pacman -S git-crypt
```

macOS with Homebrew:

```bash
brew install git-crypt
```

### 2. Initialize git-crypt

This only needs to be done once per repo:

```powershell
git-crypt init
```

That creates the internal git-crypt key data inside `.git`. This unlocks the repo on your current machine, but it is not enough for a future fresh clone. You still need to export a portable key.

### 3. Create the Private Folder

```powershell
New-Item -ItemType Directory -Force -Path Website\markdown\Private
```

Add private markdown files inside that folder, for example:

```powershell
Website\markdown\Private\Passwords.md
Website\markdown\Private\Personal-Notes.md
Website\markdown\Private\Offline-Plans.md
```

Use better filenames than `Passwords.md` if you can. That one is basically wearing a bright reflective vest.

### 4. Tell Git Which Files to Encrypt

Open or create this file:

```powershell
.gitattributes
```

Add this line:

```gitattributes
Website/markdown/Private/** filter=git-crypt diff=git-crypt
```


### 5. Check Encryption Status

Run:

```powershell
git-crypt status
```

Files under `Website\markdown\Private` should be listed as encrypted.

If you already added files before creating the `.gitattributes` rule, re-add them so Git stores the encrypted version:

```powershell
git add .gitattributes Website\markdown\Private
```

Then check again:

```powershell
git-crypt status
```

### 6. Commit the Encrypted Files

```powershell
git add .gitattributes Website\markdown\Private
git commit -m "Add encrypted private markdown notes"
```

When pushed to GitHub, those files should be encrypted blobs, not readable markdown.

## Export the Unlock Key

Export the key to somewhere outside the repo:

```powershell
git-crypt export-key C:\Users\rocket\Documents\rock-os-git-crypt.key
```

That exported key is what you use to unlock the encrypted files on another computer.

Good places to store it:

- An encrypted USB drive
- A password manager that supports file attachments
- An encrypted backup drive
- Another secure offline backup location

Bad places to store it:

- Inside this repository
- Inside the public GitHub project
- In the same folder as the encrypted markdown files
- Anywhere named something like `totally-not-the-secret-key.key`

## Decrypt on Another Computer

On a fresh computer, do this:

### 1. Install git-crypt

Windows with Scoop:

```powershell
scoop install git-crypt
```

Linux:

```bash
sudo apt install git-crypt
```

macOS:

```bash
brew install git-crypt
```

### 2. Clone the Repo

```powershell
git clone https://github.com/rocketpowerinc/rock-os.git
cd rock-os
```

Before unlocking, files in `Website\markdown\Private` will be encrypted and unreadable.

### 3. Copy the Exported Key to the Computer

Example location:

```powershell
C:\Users\rocket\Documents\rock-os-git-crypt.key
```

### 4. Unlock the Repo

From inside the repo:

```powershell
git-crypt unlock C:\Users\rocket\Documents\rock-os-git-crypt.key
```

After that, the private markdown files should turn back into normal readable files.

### 5. Confirm Everything Is Unlocked

```powershell
git-crypt status
```

You can also open one of the files:

```powershell
Get-Content Website\markdown\Private\Your-File.md
```

## Everyday Workflow

Once the repo is unlocked, use it like normal:

```powershell
git pull
```

Edit files in:

```powershell
Website\markdown\Private
```

Then commit:

```powershell
git add Website\markdown\Private
git commit -m "Update private notes"
git push
```

Git stores those files encrypted when they are committed.

## What GitHub Sees

GitHub will see that files exist in `Website/markdown/Private`, but the contents should look encrypted.

GitHub will still show filenames and folder names. Do not use sensitive names if the names themselves reveal too much.

Better:

```text
Website/markdown/Private/Recovery.md
Website/markdown/Private/Network.md
Website/markdown/Private/Personal.md
```

Less ideal:

```text
Website/markdown/Private/My-Bank-Passwords.md
Website/markdown/Private/Secret-Server-SSH-Key.md
```

## Important Warning About Git History

`git-crypt` protects files from the point where encryption is configured.

If a sensitive file was committed before the `.gitattributes` encryption rule existed, the plaintext may still exist in Git history. In that case, removing it from the latest commit is not enough. The history would need to be cleaned, and any exposed secret should be treated as compromised.

## Quick Command Reference

Initialize:

```powershell
git-crypt init
```

Track private markdown files:

```gitattributes
Website/markdown/Private/** filter=git-crypt diff=git-crypt
```

Check status:

```powershell
git-crypt status
```

Export key:

```powershell
git-crypt export-key C:\Users\rocket\Documents\rock-os-git-crypt.key
```

Unlock fresh clone:

```powershell
git-crypt unlock C:\Users\rocket\Documents\rock-os-git-crypt.key
```

## Recovery Rule

Keep at least two secure copies of the exported key.

One unlocked computer is convenient. One exported key is survival. Two exported keys, stored safely, is how you avoid future-you staring at encrypted files and inventing new swear words.
