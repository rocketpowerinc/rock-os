# **Strengthening Your Linux System: Building a Resilient Foundation**

## **1. Booting and Updating For the first time**

## **1. Network-Level Hardening**

A secure Linux system begins with a secure network. Implement the following best practices to reduce exposure and limit attack surfaces:

- Change the router’s **default administrator password** and **default SSID** immediately after setup
- Ensure **automatic firmware updates** are enabled and the router is running the latest version
- Configure the router’s DNS to a trusted provider such as **Quad9 (9.9.9.9)**
- Keep IoT and smart devices isolated on the **guest network**, unless your router supports **VLANs** for proper network segmentation

---

## **2. Operating System-Level Hardening**

Once the network is secured, reinforce the Linux OS itself with these essential measures:

- Enable and configure the system firewall (e.g., **UFW**)
- Use a privacy‑focused DNS resolver such as **NextDNS**
- Install a network monitoring tool like **Little Snitch for Linux** to observe outbound and inbound traffic
  - https://obdev.at/products/littlesnitch-linux/index.html

---

## **Notable Mentions**

Additional tools, practices, and concepts worth exploring as you continue to harden your system:

- **Fail2ban** for SSH brute‑force protection
- **CrowdSec** for collaborative, behavior‑based intrusion detection and automated blocking
- **AdGuard Home** for network‑wide ad blocking, DNS filtering, and privacy protection
- **AppArmor** or **SELinux** for mandatory access control
- **Regular system updates** and unattended‑upgrade configurations
- **Encrypted storage** (LUKS) for protecting data at rest
- **Linux Hosts File Hardening** — block telemetry and enforce DNS filtering at the OS level
  - `sudo nano /private/etc/hosts`
