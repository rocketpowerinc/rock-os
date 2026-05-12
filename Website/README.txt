ROCKOS LIVE RELOAD EDITION

FEATURES
========
- Live markdown detection
- Auto sidebar refresh
- Recursive subdirectories
- Markdown-it rendering
- No server restart required
- Cyberpunk UI

REQUIREMENTS
============
Download static-web-server.exe from:

https://github.com/static-web-server/static-web-server/releases

Place beside:
- start.bat

RUNNING
========
Double click:

start.bat

HOW LIVE RELOAD WORKS
=====================
A PowerShell watcher continuously scans:

markdown/

Every 2 seconds.

Any new:
- folders
- markdown files

automatically appear in the sidebar.

EXAMPLE
=======

markdown/
├── ai/
│   └── agents.md
│
├── guides/
│   └── setup.md

No edits required.

SUPPORTED MARKDOWN
==================
- images
- links
- videos
- tables
- code blocks
- html embeds
- lists
