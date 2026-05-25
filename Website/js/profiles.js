import { createMarkdownTabApp } from './wiki/markdown-tab.js';

function escapeHtml(value) {

    return String(value)
        .replaceAll('&', '&amp;')
        .replaceAll('<', '&lt;')
        .replaceAll('>', '&gt;')
        .replaceAll('"', '&quot;')
        .replaceAll("'", '&#039;');
}

async function profilesAreLocked() {

    try {
        const response =
            await fetch('/api/server/status?nocache=' + Date.now());

        if (!response.ok) {
            return true;
        }

        const status =
            await response.json();

        return status?.gitCrypt !== 'unlocked';
    }
    catch {
        return true;
    }
}

function renderLockedProfiles() {

    const sidebar =
        document.getElementById('sidebar');
    const resizer =
        document.getElementById('sidebarResizer');
    const expandButton =
        document.getElementById('expandSidebarBtn');
    const toc =
        document.getElementById('wikiToc');
    const content =
        document.getElementById('content');

    if (sidebar) {
        sidebar.style.display = 'none';
    }
    if (resizer) {
        resizer.style.display = 'none';
    }
    if (expandButton) {
        expandButton.style.display = 'none';
    }
    if (toc) {
        toc.innerHTML = '';
    }
    if (content) {
        content.classList.add('fullwidth');
        content.innerHTML = `
            <section class="profiles-locked-panel" aria-live="polite">
                <div class="profiles-lock-badge">Locked</div>
                <h1>Profiles Locked</h1>
                <p>Encrypted profile notes are locked with git-crypt. Unlock the repository to view Rocket, Kids, Prepper, and any future profiles.</p>
                <div class="profiles-lock-fake-button" aria-hidden="true">Profiles Locked</div>
                <pre><code>START-HERE\\Windows\\unlock-git-crypt.cmd</code></pre>
            </section>
        `;
    }
}

function currentProfileName() {

    const params =
        new URLSearchParams(window.location.search);

    return params.get('profile') || '';
}

function profileNameFromPath(path) {

    const match =
        path.match(/^profiles\/([^/]+)\//);

    return match
        ? decodeURIComponent(match[1])
        : '';
}

function profileUrl(profile) {

    const url =
        new URL('profiles.html', window.location.href);

    url.searchParams.set('profile', profile);

    return `${url.pathname}${url.search}`;
}

function renderProfilesLanding(files) {

    const sidebar =
        document.getElementById('sidebar');
    const resizer =
        document.getElementById('sidebarResizer');
    const expandButton =
        document.getElementById('expandSidebarBtn');
    const toc =
        document.getElementById('wikiToc');
    const content =
        document.getElementById('content');

    if (sidebar) {
        sidebar.style.display = 'none';
    }
    if (resizer) {
        resizer.style.display = 'none';
    }
    if (expandButton) {
        expandButton.style.display = 'none';
    }
    if (toc) {
        toc.innerHTML = '';
    }
    if (!content) {
        return;
    }

    const profiles =
        Array.from(
            new Set(
                files
                    .map(file => profileNameFromPath(file.path || file))
                    .filter(Boolean)
            )
        )
            .sort((a, b) =>
                a.toLowerCase().localeCompare(b.toLowerCase())
            );

    content.classList.add('fullwidth');
    content.innerHTML = `
        <section class="profiles-dashboard">
            <p class="wiki-error-kicker">Encrypted Profiles</p>
            <h1>Profiles</h1>
            <p>Choose a profile dashboard. Each profile keeps its own private markdown tree, search, favorites, and document view.</p>
            <div class="profiles-card-grid">
                ${profiles.map(profile => `
                    <a class="profiles-card" href="${escapeHtml(profileUrl(profile))}" data-profile="${escapeHtml(profile)}">
                        <div class="profile-card-icon"></div>
                        <div class="profiles-card-info">
                            <span>${escapeHtml(profile)}</span>
                            <small>Open private dashboard</small>
                        </div>
                    </a>
                `).join('')}
            </div>
        </section>
    `;
}

async function loadProfilesLanding() {

    try {
        const response =
            await fetch('profiles-index.json?nocache=' + Date.now());

        if (!response.ok) {
            throw new Error(`Profiles index failed with HTTP ${response.status}`);
        }

        const files =
            await response.json();

        renderProfilesLanding(
            Array.isArray(files) ? files : []
        );
    }
    catch (err) {
        console.warn(err);
        renderLockedProfiles();
    }
}

const REDDIT_PLACEHOLDER = 'data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCA4MCA1MCIgd2lkdGg9IjgwIiBoZWlnaHQ9IjUwIj48cmVjdCB3aWR0aD0iODAiIGhlaWdodD0iNTAiIHJ4PSI0IiBmaWxsPSIjMWExYTI0IiBzdHJva2U9IiMzZTRhNTYiIHN0cm9rZS13aWR0aD0iMSIvPjxjaXJjbGUgY3g9IjQwIiBjeT0iMjUiIHI9IjEwIiBmaWxsPSJub25lIiBzdHJva2U9IiNmZjQ1MDAiIHN0cm9rZS13aWR0aD0iMiIvPjxsaW5lIHgxPSI0MCIgeTE9IjE1IiB4Mj0iNDMiIHkyPSIxMCIgc3Ryb2tlPSIjZmY0NTAwIiBzdHJva2Utd2lkdGg9IjIiLz48Y2lyY2xlIGN4PSI0MyIgY3k9IjEwIiByPSIyIiBmaWxsPSIjZmY0NTAwIi8+PGNpcmNsZSBjeD0iMzYiIGN5PSIyNSIgcj0iMS41IiBmaWxsPSIjZmY0NTAwIi8+PGNpcmNsZSBjeD0iNDQiIGN5PSIyNSIgcj0iMS41IiBmaWxsPSIjZmY0NTAwIi8+PHBhdGggZD0iTSAzNSAyOSBRIDQwIDMzIDQ1IDI5IiBmaWxsPSJub25lIiBzdHJva2U9IiNmZjQ1MDAiIHN0cm9rZS13aWR0aD0iMS41Ii8+PC9zdmc+';

const YOUTUBE_PLACEHOLDER = 'data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCA4MCA1MCIgd2lkdGg9IjgwIiBoZWlnaHQ9IjUwIj48cmVjdCB3aWR0aD0iODAiIGhlaWdodD0iNTAiIHJ4PSI0IiBmaWxsPSIjMWExYTI0IiBzdHJva2U9IiMzZTRhNTYiIHN0cm9rZS13aWR0aD0iMSIvPjxwb2x5Z29uIHBvaW50cz0iMzUsMTggNTAsMjUgMzUsMzIiIGZpbGw9IiNmZjAwMDAiLz48L3N2Zz4=';
const PODCAST_PLACEHOLDER = 'data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCA4MCA1MCIgd2lkdGg9IjgwIiBoZWlnaHQ9IjUwIj48cmVjdCB3aWR0aD0iODAiIGhlaWdodD0iNTAiIHJ4PSI0IiBmaWxsPSIjMWExYTI0IiBzdHJva2U9IiMzZTRhNTYiIHN0cm9rZS13aWR0aD0iMSIvPjxjaXJjbGUgY3g9IjQwIiBjeT0iMjAiIHI9IjYiIGZpbGw9Im5vbmUiIHN0cm9rZT0iIzQ2ODJCNCIgc3Ryb2tlLXdpZHRoPSIyIi8+PHJlY3QgeD0iMzciIHk9IjIwIiB3aWR0aD0iNiIgaGVpZ2h0PSI4IiByeD0iMyIgZmlsbD0iIzQ2ODJCNCIvPjxwYXRoIGQ9Ik0gMzQgMjIgQSA4IDggMCAwIDAgNDYgMjIiIGZpbGw9Im5vbmUiIHN0cm9rZT0iIzQ2ODJCNCIgc3Ryb2tlLXdpZHRoPSIyIi8+PGxpbmUgeDE9IjQwIiB5MT0iMzAiIHgyPSI0MCIgeTI9IjM2IiBzdHJva2U9IiM0NjgyQjQiIHN0cm9rZS13aWR0aD0iMiIvPjxsaW5lIHgxPSIzNSIgeTE9IjM2IiB4Mj0iNDUiIHkyPSIzNiIgc3Ryb2tlPSIjNDY4MkI0IiBzdHJva2Utd2lkdGg9IjIiLz48L3N2Zz4=';

async function fetchRedditFeed(subreddit, fallback) {
    try {
        const res = await fetch(`/api/feeds/reddit?subreddit=${encodeURIComponent(subreddit)}`);
        if (!res.ok) throw new Error();
        const items = await res.json();
        if (!Array.isArray(items) || items.length === 0) throw new Error();
        return items.map(item => ({
            title: item.title,
            url: item.url,
            created: item.created || 'Today',
            author: item.author || 'u/anonymous',
            thumbnail: item.thumbnail || REDDIT_PLACEHOLDER
        }));
    } catch (e) {
        console.warn(`Could not fetch real-time Reddit r/${subreddit} feed. Loading fallback cached feed.`);
        return fallback;
    }
}

async function fetchYouTubeFeed(channels, playlists, urls, limit, fallback) {
    try {
        const url = new URL('/api/feeds/youtube', window.location.origin);
        if (Array.isArray(channels)) {
            channels.forEach(id => url.searchParams.append('channel_id', id));
        } else if (channels) {
            url.searchParams.append('channel_id', channels);
        }
        if (Array.isArray(playlists)) {
            playlists.forEach(id => url.searchParams.append('playlist_id', id));
        } else if (playlists) {
            url.searchParams.append('playlist_id', playlists);
        }
        if (Array.isArray(urls)) {
            urls.forEach(u => url.searchParams.append('url', u));
        } else if (urls) {
            url.searchParams.append('url', urls);
        }
        if (limit) {
            url.searchParams.append('limit', limit);
        }

        const res = await fetch(url);
        if (!res.ok) throw new Error();
        const items = await res.json();
        if (!Array.isArray(items) || items.length === 0) throw new Error();
        return items.map(item => ({
            title: item.title,
            url: item.url,
            date: item.created || 'Today',
            thumbnail: item.thumbnail || YOUTUBE_PLACEHOLDER
        }));
    } catch (e) {
        console.warn(`Could not fetch live YouTube feed. Loading fallback cached videos.`);
        return fallback;
    }
}

async function fetchPodcastFeed(feedUrl, limit, fallback) {
    try {
        const url = `/api/feeds/podcast?url=${encodeURIComponent(feedUrl)}&limit=${limit || 5}`;
        const res = await fetch(url);
        if (!res.ok) throw new Error();
        const items = await res.json();
        if (!Array.isArray(items) || items.length === 0) throw new Error();
        return items.map(item => ({
            title: item.title,
            url: item.url,
            date: item.created || 'Today',
            thumbnail: item.thumbnail || PODCAST_PLACEHOLDER
        }));
    } catch (e) {
        console.warn(`Could not fetch live Podcast feed. Loading fallback.`);
        return fallback;
    }
}

function renderDashboard(profile, config, feeds) {
    if (!config || !Array.isArray(config.widgets)) return;
    feeds = feeds || {};

    const sidebar = document.getElementById('sidebar');
    const resizer = document.getElementById('sidebarResizer');
    const expandButton = document.getElementById('expandSidebarBtn');
    const toc = document.getElementById('wikiToc');
    const content = document.getElementById('content');

    if (sidebar) sidebar.style.display = 'none';
    if (resizer) resizer.style.display = 'none';
    if (expandButton) expandButton.style.display = 'none';
    if (toc) toc.style.display = 'none';

    if (content) {
        content.classList.add('fullwidth');
        content.innerHTML = `
            <div class="glance-header-card">
                <div class="glance-header-left">
                    <div class="${config.avatarClass}"></div>
                    <div class="glance-header-text">
                        <h1>${escapeHtml(config.title)}</h1>
                        <p>${escapeHtml(config.subtitle)}</p>
                    </div>
                </div>
                <div class="glance-header-actions">
                    <button id="viewNotesBtn" class="glance-btn">
                        <span>📄</span> View Private Notes
                    </button>
                </div>
            </div>
            <div class="glance-dashboard">
                ${config.widgets.map((w, idx) => {
                    if (w.type === 'bookmarks') {
                        return `
                            <div class="glance-card widget-bookmarks">
                                <div class="glance-card-header">
                                    <h2>${escapeHtml(w.title)}</h2>
                                    <span class="glance-badge">${escapeHtml(w.badge)}</span>
                                </div>
                                ${w.bookmarks.map(section => `
                                    <div class="glance-bookmark-sec" style="${section !== w.bookmarks[0] ? 'margin-top: 8px;' : ''}">
                                        <div class="glance-bookmark-title">${escapeHtml(section.section)}</div>
                                        ${section.items.map(item => `
                                            <a class="glance-bookmark-item" href="${escapeHtml(item.url)}" target="_blank">
                                                <div class="glance-bookmark-info">
                                                    <span class="glance-bookmark-name">${escapeHtml(item.name)}</span>
                                                    <span class="glance-bookmark-desc">${escapeHtml(item.desc)}</span>
                                                </div>
                                                <span class="glance-bookmark-arrow">➔</span>
                                            </a>
                                        `).join('')}
                                    </div>
                                `).join('')}
                            </div>
                        `;
                    }
                    return `
                        <div class="glance-card" id="widget-${idx}">
                            <div class="glance-card-header">
                                <h2>${escapeHtml(w.title)}</h2>
                                <span class="glance-badge">${escapeHtml(w.badge)}</span>
                            </div>
                            <div class="widget-content-container" id="widget-content-${idx}">
                                <p style="color: var(--text-muted); font-size: 0.9rem;">Syncing feed...</p>
                            </div>
                        </div>
                    `;
                }).join('')}
            </div>
        `;

        // Async load feed content for non-bookmarks widgets
        config.widgets.forEach((w, idx) => {
            if (w.type === 'bookmarks') return;

            const container = document.getElementById(`widget-content-${idx}`);
            if (!container) return;

            if (w.type === 'reddit') {
                fetchRedditFeed(w.subreddit, w.fallback).then(posts => {
                    container.innerHTML = `
                        <ul class="glance-feed-list">
                            ${posts.map(post => `
                                <li class="glance-feed-item">
                                    <img class="glance-feed-thumb" src="${escapeHtml(post.thumbnail)}" onerror="this.src='${REDDIT_PLACEHOLDER}';" alt="Reddit Thumbnail">
                                    <div class="glance-feed-content">
                                        <a class="glance-feed-title" href="${escapeHtml(post.url)}" target="_blank">${escapeHtml(post.title)}</a>
                                        <div class="glance-feed-meta">
                                            <span class="glance-badge">${escapeHtml(post.author)}</span>
                                            <span>${escapeHtml(post.created)}</span>
                                        </div>
                                    </div>
                                </li>
                            `).join('')}
                        </ul>
                    `;
                });
            } else if (w.type === 'youtube') {
                const urls = (w.feedKey && feeds[w.feedKey] && feeds[w.feedKey].length > 0) ? feeds[w.feedKey] : null;
                fetchYouTubeFeed(w.channels, w.playlists, urls, w.limit, w.fallback).then(videos => {
                    container.innerHTML = `
                        <ul class="glance-feed-list">
                            ${videos.map(video => `
                                <li class="glance-feed-item">
                                    <img class="glance-feed-thumb" src="${escapeHtml(video.thumbnail)}" onerror="this.src='${YOUTUBE_PLACEHOLDER}';" alt="YouTube Thumbnail">
                                    <div class="glance-feed-content">
                                        <a class="glance-feed-title" href="${escapeHtml(video.url)}" target="_blank">${escapeHtml(video.title)}</a>
                                        <div class="glance-feed-meta">
                                            <span class="glance-badge">${escapeHtml(w.badge)}</span>
                                            <span>${escapeHtml(video.date)}</span>
                                        </div>
                                    </div>
                                </li>
                            `).join('')}
                        </ul>
                    `;
                });
            } else if (w.type === 'podcast') {
                const feedUrl = (w.feedKey && feeds[w.feedKey] && feeds[w.feedKey][0]) ? feeds[w.feedKey][0] : w.feedUrl;
                fetchPodcastFeed(feedUrl, w.limit, w.fallback).then(episodes => {
                    container.innerHTML = `
                        <ul class="glance-feed-list">
                            ${episodes.map(episode => `
                                <li class="glance-feed-item">
                                    <img class="glance-feed-thumb" src="${escapeHtml(episode.thumbnail)}" onerror="this.src='${PODCAST_PLACEHOLDER}';" alt="Podcast Art">
                                    <div class="glance-feed-content">
                                        <a class="glance-feed-title" href="${escapeHtml(episode.url)}" target="_blank">${escapeHtml(episode.title)}</a>
                                        <div class="glance-feed-meta">
                                            <span class="glance-badge">${escapeHtml(w.badge)}</span>
                                            <span>${escapeHtml(episode.date)}</span>
                                        </div>
                                    </div>
                                </li>
                            `).join('')}
                        </ul>
                    `;
                });
            }
        });
    }

    // Toggle button handler
    const viewNotesBtn = document.getElementById('viewNotesBtn');
    if (viewNotesBtn) {
        viewNotesBtn.addEventListener('click', () => {
            renderNotesViewer(profile);
        });
    }
}

function renderNotesViewer(profile) {
    const sidebar = document.getElementById('sidebar');
    const resizer = document.getElementById('sidebarResizer');
    const expandButton = document.getElementById('expandSidebarBtn');
    const toc = document.getElementById('wikiToc');
    const content = document.getElementById('content');

    if (sidebar) sidebar.style.display = '';
    if (resizer) resizer.style.display = '';
    if (expandButton) expandButton.style.display = 'none';
    if (toc) toc.style.display = '';

    const heading = document.querySelector('.sidebar-header h3');
    const search = document.getElementById('profilesSearchInput');

    if (heading) {
        heading.textContent = profile;
    }
    if (search) {
        search.placeholder = `Search ${profile}`;
        search.setAttribute('aria-label', `Search ${profile}`);
    }
    if (content) {
        content.classList.remove('fullwidth');
        content.innerHTML = `
            <h1>${escapeHtml(profile)}</h1>
            <p>Select a profile document.</p>
        `;
    }

    createMarkdownTabApp({
        key: `profiles-${profile}`,
        label: profile,
        emptyLabel: 'profile files',
        searchStatusId: 'profilesSearchStatus',
        searchInputId: 'profilesSearchInput',
        refreshButtonId: 'refreshProfilesBtn',
        indexUrl: `profiles-index.json?profile=${encodeURIComponent(profile)}`,
        docApiUrl: '/api/profiles/doc',
        searchApiUrl: `/api/profiles/search?profile=${encodeURIComponent(profile)}`,
        pathPrefix: `profiles/${profile}`,
        directOpenPageName: 'profiles.html'
    });
}

async function loadFeedsConfig() {
    try {
        const res = await fetch(`profiles/feeds.txt?nocache=${Date.now()}`);
        if (!res.ok) return {};
        const text = await res.text();
        const sections = {};
        let currentSection = null;
        const lines = text.split('\n');
        for (let line of lines) {
            line = line.trim();
            if (!line || line.startsWith('#')) continue;
            if (line.startsWith('[') && line.endsWith(']')) {
                currentSection = line.slice(1, -1).trim();
                sections[currentSection] = [];
            } else if (currentSection) {
                sections[currentSection].push(line);
            }
        }
        return sections;
    } catch (e) {
        console.warn('Could not load or parse profiles/feeds.txt', e);
        return {};
    }
}

async function startProfiles() {
    if (await profilesAreLocked()) {
        renderLockedProfiles();
        return;
    }

    const profile = currentProfileName();
    if (!profile) {
        await loadProfilesLanding();
        return;
    }

    document.title = `Rock-OS ${profile}`;

    // Try loading profile-specific dashboard config dynamically
    try {
        const res = await fetch(`profiles/${encodeURIComponent(profile)}/dashboard.json?nocache=${Date.now()}`);
        if (res.ok) {
            const config = await res.json();
            const feeds = await loadFeedsConfig();
            renderDashboard(profile, config, feeds);
            return;
        }
    } catch (e) {
        console.warn(`No dashboard configuration found for profile "${profile}". Falling back to notes viewer.`, e);
    }

    // Default: notes viewer
    renderNotesViewer(profile);
}

startProfiles();
