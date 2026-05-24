# AnduinOS Bootstrap

### Update

```bash
sudo apt update && sudo apt upgrade
```

```bash
sudo do_anduinos_upgrade
```

### Timeshift - Create First System Backup

```bash
sudo apt update
sudo apt install timeshift -y

# Initial setup
sudo timeshift --setup

# Enable Schedule
sudo nano /etc/timeshift/timeshift.json

# Create first snapshot
sudo timeshift --create --comments "First snapshot"
```

### Restic - Create First $HOME Data Backup

```bash
sudo apt update
sudo apt install restic -y

mkdir -p ~/Backups/restic


# Initialize repository (one-time)
restic -r ~/Backups/restic init

# Backup
restic -r ~/Backups/restic backup ~ \
  --exclude ~/Backups \
  --exclude ~/Downloads \
  --exclude ~/Docker

# Retention policy + cleanup
restic -r ~/Backups/restic forget \
  --keep-last 5 \
  --keep-daily 7 \
  --keep-weekly 4 \
  --keep-monthly 6 \
  --prune
```

### Gnome Tweaks

```bash
# Dock Icons
gsettings set org.gnome.shell favorite-apps "['org.gnome.Settings.desktop', 'org.gnome.Ptyxis.desktop', 'org.gnome.Nautilus.desktop', 'firefox-esr.desktop', 'org.gnome.Software.desktop']"
```

- Ptyxis Terminal
  - Set Shortcut keys to use `ctrl + c` and `ctrl + v` to copy/paste

### Go-PWR - Setup (From Public Google Drive Link)

```bash
sudo apt update && sudo apt install -y git gh jq make bat tmux curl wget glow gum
wget -O go-pwr "https://docs.google.com/uc?export=download&id=1LaGHTYWZmsJ_nmjw_S-0W7l4BTEyp-MJ" && \
chmod +x go-pwr && \
sudo mv go-pwr /usr/bin/go-pwr
```

### Go-PWR

- With Go-PWR Install 
  - Bashrc
  - AnduinOS Justfile + Yad Toolkit
  - Wallpapers

### Docker - Install

- `curl -fsSL https://get.docker.com -o get-docker.sh && sudo sh get-docker.sh`

### Users

- set rocket user avatar

```bash
IMG_URL="https://cdn-icons-png.flaticon.com/512/9434/9434459.png"
USER_NAME=$(whoami)

sudo curl -L "$IMG_URL" -o /var/lib/AccountsService/icons/$USER_NAME &&
sudo chmod 644 /var/lib/AccountsService/icons/$USER_NAME &&

sudo bash -c "cat > /var/lib/AccountsService/users/$USER_NAME <<EOF
[User]
Icon=/var/lib/AccountsService/icons/$USER_NAME
EOF" &&

sudo systemctl restart accounts-daemon
```
### Install Essential Cli-Tools