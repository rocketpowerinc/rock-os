# Git-Crypt Cheat Sheet

`git-crypt` encrypts selected files in a Git repository. People with the key see normal readable files after unlocking the repo. People without the key see encrypted blobs.

It is useful when you want to keep some private files in a repo without making the whole repo private.

## What Git-Crypt Protects

`git-crypt` protects file contents.

It does not hide:

- File names
- Folder names
- Commit messages
- Branch names
- The fact that encrypted files exist

Use boring names for sensitive files. A file named `Personal.md` leaks less than a file named `Bank-Passwords-And-Server-Keys.md`.

## Install Git-Crypt

Windows with Scoop:

```powershell
scoop install git-crypt
```

Linux:

```bash
sudo apt install git-crypt
sudo dnf install git-crypt
sudo pacman -S git-crypt
```

macOS with Homebrew:

```bash
brew install git-crypt
```

Check the install:

```bash
git-crypt --version
```

## Initialize A Repo

Run this once from the root of a Git repository:

```bash
git-crypt init
```

This creates the encryption setup inside `.git`. Your current clone can now encrypt and decrypt files, but a fresh clone will still need an exported key.

## Choose Files To Encrypt

`git-crypt` uses `.gitattributes` to decide what gets encrypted.

Example: encrypt everything in a `private` folder:

```gitattributes
private/** filter=git-crypt diff=git-crypt
```

Example: encrypt all `.secret` files:

```gitattributes
*.secret filter=git-crypt diff=git-crypt
```

Example: encrypt one specific file:

```gitattributes
notes/private.md filter=git-crypt diff=git-crypt
```

Important: these lines go inside `.gitattributes`. Do not run them as terminal commands.

## Add And Commit Encrypted Files

After updating `.gitattributes`, add the rule and the protected files:

```bash
git add .gitattributes private/
git commit -m "Add encrypted private files"
```

Check what `git-crypt` thinks is encrypted:

```bash
git-crypt status
```

If a file was already tracked before you added the encryption rule, re-add it so Git stores the encrypted version:

```bash
git add private/
git commit -m "Re-add private files through git-crypt"
```

## Export The Key

Export a key from a trusted unlocked clone:

```bash
git-crypt export-key my-repo-git-crypt.key
```

Keep this key outside the repo.

Good places:

- Encrypted USB drive
- Password manager with file attachments
- Encrypted backup drive
- Another secure offline backup

Bad places:

- The Git repository
- A public cloud folder
- The same folder as the encrypted files
- Anywhere you will forget exists until five minutes after disaster strikes

## Unlock A Fresh Clone

Clone the repo:

```bash
git clone https://github.com/example/example-repo.git
cd example-repo
```

Unlock it with the exported key:

```bash
git-crypt unlock /path/to/my-repo-git-crypt.key
```

After unlocking, encrypted files become readable in the working tree.

Check status:

```bash
git-crypt status
```

## Everyday Workflow

Once unlocked, work normally:

```bash
git pull
```

Edit protected files, then commit:

```bash
git add private/
git commit -m "Update private notes"
git push
```

The files are readable locally but encrypted in Git.

## Verify What GitHub Sees

After pushing, check the file on the remote hosting site. It should not render as readable text. It should look like encrypted binary or unreadable data.

You can also test from a fresh clone without unlocking:

```bash
git clone https://github.com/example/example-repo.git test-locked-clone
cd test-locked-clone
cat private/example.md
```

Without the key, the file should not be readable.

## Important Git History Warning

`git-crypt` only protects files after encryption is configured.

If a secret was committed before the `.gitattributes` rule existed, the plaintext may still exist in Git history. Removing the file from the latest commit is not enough.

If that happens:

- Treat the exposed secret as compromised
- Rotate passwords, tokens, or keys
- Consider cleaning Git history with a tool like `git filter-repo`

## Quick Reference

Initialize:

```bash
git-crypt init
```

Encrypt a folder:

```gitattributes
private/** filter=git-crypt diff=git-crypt
```

Check status:

```bash
git-crypt status
```

Export key:

```bash
git-crypt export-key my-repo-git-crypt.key
```

Unlock:

```bash
git-crypt unlock /path/to/my-repo-git-crypt.key
```

## Recovery Rule

Keep at least two secure copies of the exported key.

One unlocked laptop is convenient. One exported key is survival. Two exported keys, stored safely, is how you avoid future-you staring at encrypted files and learning new vocabulary.
