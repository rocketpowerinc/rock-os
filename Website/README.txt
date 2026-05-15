ROCKOS WIKI
===========

FEATURES
========
- Simple markdown files rendered as a website wiki
- Automatic sidebar tree from markdown folders
- Recursive subdirectories
- Markdown-it rendering in the browser
- Copy buttons on code blocks
- Cross-platform Go server for Windows, Linux, and macOS

REQUIREMENTS
============
Install Go:

https://go.dev/dl/

RUNNING
=======
From the Website folder:

go run .

Windows helper:

start-rock-os.cmd

Linux/macOS helper:

sh start-rock-os.sh

The server opens:

http://YOUR_LOCAL_IP:8000

By default, the server binds to all network interfaces and opens your best
detected local network IP so other devices on the same network can open the
wiki. If your computer has multiple network adapters, the server prints the
other detected local URLs too.

LAN MODE
========
This is the default mode:

go run . --host local

You can also bind all network interfaces manually:

go run . --host 0.0.0.0

The server will still print the preferred local IP URL to open, such as:

http://192.168.1.2:8000

LOCALHOST ONLY
==============
To serve only on the current computer:

go run . --host 127.0.0.1

CUSTOM PORT
===========
Example:

go run . --port 9000

BUILD INDEX ONLY
================
To rebuild markdown-index.json without running the server:

go run . --build-index

HOW THE WIKI INDEX WORKS
========================
The Go server scans:

markdown/

Every 2 seconds and writes:

markdown-index.json

The browser reads that JSON file, builds the sidebar tree, fetches the selected
markdown file, and renders it into the page.

EXAMPLE
=======
markdown/
  Linux/
    AnduinOS/
      Bootstrap.md

SUPPORTED MARKDOWN
==================
- images
- links
- videos
- tables
- code blocks
- html embeds
- lists
