import { createMarkdownTabApp } from './wiki/markdown-tab.js';

const isDashboardsMode =
    window.location.pathname.includes('/dashboards/') ||
    window.location.pathname.endsWith('/dashboards.html');

const appMode = isDashboardsMode
    ? {
        rootDir: 'dashboards',
        indexFile: 'dashboards-index.json',
        apiRoot: 'dashboards',
        mainPage: 'dashboards.html',
        pageTitle: 'Dashboards',
        landingKicker: 'Local Dashboards',
        landingDescription: 'Choose a dashboard. Each dashboard can keep its own widgets, markdown tree, search, favorites, and document view.',
        cardDescription: 'Open local dashboard',
        emptyLabel: 'dashboard files',
        defaultSelectText: 'Select a dashboard document.',
        viewNotesText: 'View Dashboard Notes',
        searchPlaceholder: 'Search dashboards',
        documentTitlePrefix: 'Rock-OS'
    }
    : {
        rootDir: 'profiles',
        indexFile: 'profiles-index.json',
        apiRoot: 'profiles',
        mainPage: 'profiles.html',
        pageTitle: 'Profiles',
        landingKicker: 'Encrypted Profiles',
        landingDescription: 'Choose a profile dashboard. Each profile keeps its own private markdown tree, search, favorites, and document view.',
        cardDescription: 'Open private dashboard',
        emptyLabel: 'profile files',
        defaultSelectText: 'Select a profile document.',
        viewNotesText: 'View Private Notes',
        searchPlaceholder: 'Search profiles',
        documentTitlePrefix: 'Rock-OS'
    };

function escapeHtml(value) {

    return String(value)
        .replaceAll('&', '&amp;')
        .replaceAll('<', '&lt;')
        .replaceAll('>', '&gt;')
        .replaceAll('"', '&quot;')
        .replaceAll("'", '&#039;');
}

function linkTargetAttrs(href) {
    try {
        const url = new URL(href, window.location.origin);

        if (
            (url.protocol === 'http:' || url.protocol === 'https:') &&
            url.host !== window.location.host
        ) {
            return ' target="_blank" rel="noopener noreferrer"';
        }
    }
    catch {
        return '';
    }

    return '';
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

function renderDashboardError(message) {

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
                <div class="profiles-lock-badge">Unavailable</div>
                <h1>${escapeHtml(appMode.pageTitle)} Unavailable</h1>
                <p>${escapeHtml(message)}</p>
            </section>
        `;
    }
}

function currentProfileName() {
    const params = new URLSearchParams(window.location.search);
    let profile = params.get('profile') || params.get('dashboard') || '';
    if (!profile) {
        const parts = window.location.pathname.split('/').filter(Boolean);
        const filename = parts.pop() || '';
        if (filename && filename !== appMode.mainPage && filename.endsWith('.html')) {
            const name = filename === 'index.html' && parts.length > 0
                ? parts[parts.length - 1]
                : filename.substring(0, filename.length - 5);
            profile = name.charAt(0).toUpperCase() + name.slice(1);
        } else if (filename && parts.includes(appMode.rootDir)) {
            profile = filename.charAt(0).toUpperCase() + filename.slice(1);
        }
    }
    return profile;
}

function profileNameFromPath(path) {

    const match =
        path.match(new RegExp(`^${appMode.rootDir}/([^/]+)/`));

    return match
        ? decodeURIComponent(match[1])
        : '';
}

function profileUrl(profile) {
    return `/${appMode.rootDir}/${profile}/`;
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
            <p class="wiki-error-kicker">${escapeHtml(appMode.landingKicker)}</p>
            <h1>${escapeHtml(appMode.pageTitle)}</h1>
            <p>${escapeHtml(appMode.landingDescription)}</p>
            <div class="profiles-card-grid">
                ${profiles.map(profile => `
                    <a class="profiles-card" href="${escapeHtml(profileUrl(profile))}" data-profile="${escapeHtml(profile)}">
                        <div class="profile-card-icon"></div>
                        <div class="profiles-card-info">
                            <span>${escapeHtml(profile)}</span>
                            <small>${escapeHtml(appMode.cardDescription)}</small>
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
            await fetch(`/${appMode.indexFile}?nocache=` + Date.now());

        if (!response.ok) {
            throw new Error(`${appMode.pageTitle} index failed with HTTP ${response.status}`);
        }

        const files =
            await response.json();

        renderProfilesLanding(
            Array.isArray(files) ? files : []
        );
    }
    catch (err) {
        console.warn(err);
        if (isDashboardsMode) {
            renderDashboardError('The dashboards index could not be loaded. Restart Rock-OS from the latest server source or release binary.');
            return;
        }
        renderLockedProfiles();
    }
}

const REDDIT_PLACEHOLDER = '/assets/widget-icons/reddit.png';

const YOUTUBE_PLACEHOLDER = 'data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCA4MCA1MCIgd2lkdGg9IjgwIiBoZWlnaHQ9IjUwIj48cmVjdCB3aWR0aD0iODAiIGhlaWdodD0iNTAiIHJ4PSI0IiBmaWxsPSIjMWExYTI0IiBzdHJva2U9IiMzZTRhNTYiIHN0cm9rZS13aWR0aD0iMSIvPjxwb2x5Z29uIHBvaW50cz0iMzUsMTggNTAsMjUgMzUsMzIiIGZpbGw9IiNmZjAwMDAiLz48L3N2Zz4=';
const PODCAST_PLACEHOLDER = 'data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCA4MCA1MCIgd2lkdGg9IjgwIiBoZWlnaHQ9IjUwIj48cmVjdCB3aWR0aD0iODAiIGhlaWdodD0iNTAiIHJ4PSI0IiBmaWxsPSIjMWExYTI0IiBzdHJva2U9IiMzZTRhNTYiIHN0cm9rZS13aWR0aD0iMSIvPjxjaXJjbGUgY3g9IjQwIiBjeT0iMjAiIHI9IjYiIGZpbGw9Im5vbmUiIHN0cm9rZT0iIzQ2ODJCNCIgc3Ryb2tlLXdpZHRoPSIyIi8+PHJlY3QgeD0iMzciIHk9IjIwIiB3aWR0aD0iNiIgaGVpZ2h0PSI4IiByeD0iMyIgZmlsbD0iIzQ2ODJCNCIvPjxwYXRoIGQ9Ik0gMzQgMjIgQSA4IDggMCAwIDAgNDYgMjIiIGZpbGw9Im5vbmUiIHN0cm9rZT0iIzQ2ODJCNCIgc3Ryb2tlLXdpZHRoPSIyIi8+PGxpbmUgeDE9IjQwIiB5MT0iMzAiIHgyPSI0MCIgeTI9IjM2IiBzdHJva2U9IiM0NjgyQjQiIHN0cm9rZS13aWR0aD0iMiIvPjxsaW5lIHgxPSIzNSIgeTE9IjM2IiB4Mj0iNDUiIHkyPSIzNiIgc3Ryb2tlPSIjNDY4MkI0IiBzdHJva2Utd2lkdGg9IjIiLz48L3N2Zz4=';
const SPOTIFY_PLACEHOLDER = 'data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCA4MCA1MCIgd2lkdGg9IjgwIiBoZWlnaHQ9IjUwIj48cmVjdCB3aWR0aD0iODAiIGhlaWdodD0iNTAiIHJ4PSI0IiBmaWxsPSIjMWExYTI0IiBzdHJva2U9IiMzZTRhNTYiIHN0cm9rZS13aWR0aD0iMSIvPjxjaXJjbGUgY3g9IjQwIiBjeT0iMjUiIHI9IjEyIiBmaWxsPSIjMWRiOTU0Ii8+PHBhdGggZD0iTSAzMiAyNCBDIDM3IDIxIDQzIDIxIDQ4IDI0IE0gMzQgMjcgQyAzOCAyNSA0MiAyNSA0NiAyNyBNIDM2IDMwIEMgMzkgMjkgNDEgMjkgNDQgMzAiIGZpbGw9Im5vbmUiIHN0cm9rZT0iIzEyMTIxMiIgc3Ryb2tlLXdpZHRoPSIxLjUiIHN0cm9rZS1saW5lY2FwPSJyb3VuZCIvPjwvc3ZnPg==';
const NEWS_PLACEHOLDER = 'data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCA4MCA1MCIgd2lkdGg9IjgwIiBoZWlnaHQ9IjUwIj48cmVjdCB3aWR0aD0iODAiIGhlaWdodD0iNTAiIHJ4PSI0IiBmaWxsPSIjMWExYTI0IiBzdHJva2U9IiMzZTRhNTYiIHN0cm9rZS13aWR0aD0iMSIvPjxyZWN0IHg9IjE1IiB5PSIxNSIgd2lkdGg9IjgwIiBoZWlnaHQ9IjIwIiByeD0iMiIgZmlsbD0ibm9uZSIgc3Ryb2tlPSIjNDZCOEQzIiBzdHJva2Utd2lkdGg9IjIiLz48bGluZSB4MT0iMjAiIHkxPSIyMCIgeDI9Ijg1IiB5Mj0iMjAiIHN0cm9rZT0iIzQ2QjhEMyIgc3Ryb2tlLXdpZHRoPSIyIi8+PGxpbmUgeDE9IjIwIiB5MT0iMjUiIHgyPSI2MCIgeTI9IjI1IiBzdHJva2U9IiM0NkI4RDMiIHN0cm9rZS13aWR0aD0iMiIvPjxsaW5lIHgxPSIyMCIgeTE9IjMwIiB4Mj0iNTUiIHkyPSIzMCIgc3Ryb2tlPSIjNDZCOEQzIiBzdHJva2Utd2lkdGg9IjIiLz48L3N2Zz4=';
const GOOGLE_NEWS_PLACEHOLDER = '/assets/widget-icons/google-news.png';

function newsPlaceholderForSource(source) {
    const normalized = String(source || '').toLowerCase();

    if (normalized.includes('google')) {
        return GOOGLE_NEWS_PLACEHOLDER;
    }

    return NEWS_PLACEHOLDER;
}

async function fetchRedditFeed(subreddit, urls, fallback) {
    try {
        let urlStr = '/api/feeds/reddit';
        if (urls && urls.length > 0) {
            urlStr += `?url=${encodeURIComponent(urls[0])}`;
        } else {
            urlStr += `?subreddit=${encodeURIComponent(subreddit)}`;
        }
        const res = await fetch(urlStr);
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

async function fetchSpotifyFeed(urls, limit, fallback) {
    try {
        const url = new URL('/api/feeds/spotify', window.location.origin);
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
            date: item.created || 'Spotify',
            thumbnail: item.thumbnail || SPOTIFY_PLACEHOLDER
        }));
    } catch (e) {
        console.warn(`Could not fetch live Spotify feed. Loading fallback.`);
        return fallback;
    }
}

async function fetchNewsFeed(urls, limit, fallback) {
    try {
        const url = new URL('/api/feeds/news', window.location.origin);
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
            date: item.created || 'News',
            source: item.source || 'News',
            thumbnail: item.thumbnail || newsPlaceholderForSource(item.source)
        }));
    } catch (e) {
        console.warn(`Could not fetch live News feed. Loading fallback.`);
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
                        <span>📄</span> ${escapeHtml(appMode.viewNotesText)}
                    </button>
                </div>
            </div>
            <div class="glance-dashboard">
                ${config.widgets.map((w, idx) => {
                    if (w.type === 'featuring') {
                        return `
                            <div class="glance-card widget-featuring card-size-${w.card_size}">
                                <div class="glance-card-header">
                                    <h2>${escapeHtml(w.title)}</h2>
                                    <span class="glance-badge">${escapeHtml(w.badge)}</span>
                                </div>
                                <div class="glance-featuring-grid">
                                    ${w.bookmarks.flatMap(section => section.items).map(item => {
                                        const initials = item.name ? item.name.split(' ').map(n => n[0]).slice(0, 2).join('').toUpperCase() : '★';
                                        return `
                                            <a class="glance-featuring-banner" href="${escapeHtml(item.url)}"${linkTargetAttrs(item.url)}>
                                                <div class="featuring-banner-visual">
                                                    <span>${escapeHtml(initials)}</span>
                                                </div>
                                                <div class="featuring-banner-details">
                                                    <span class="featuring-banner-title">${escapeHtml(item.name)}</span>
                                                    <p class="featuring-banner-description">${escapeHtml(item.desc)}</p>
                                                    <div class="featuring-banner-action">
                                                        <span>Explore Now</span>
                                                        <span class="featuring-banner-arrow">➔</span>
                                                    </div>
                                                </div>
                                            </a>
                                        `;
                                    }).join('')}
                                </div>
                            </div>
                        `;
                    }
                    if (w.type === 'bookmarks') {
                        const layout = w.layout || 'vertical';
                        const size = w.size || 'medium';
                        if (layout === 'horizontal' || layout === 'banners') {
                            return `
                                <div class="glance-card widget-bookmarks banners-layout card-size-${w.card_size} link-size-${w.link_size}">
                                    <div class="glance-card-header">
                                        <h2>${escapeHtml(w.title)}</h2>
                                        <span class="glance-badge">${escapeHtml(w.badge)}</span>
                                    </div>
                                    <div class="glance-bookmarks-banners-grid link-size-${w.link_size}">
                                        ${w.bookmarks.flatMap(section => section.items).map(item => {
                                            const initials = item.name ? item.name.split(' ').map(n => n[0]).slice(0, 2).join('').toUpperCase() : '★';
                                            return `
                                                <a class="glance-bookmark-banner link-size-${w.link_size}" href="${escapeHtml(item.url)}"${linkTargetAttrs(item.url)}>
                                                    <div class="bookmark-banner-accent">
                                                        <span>${escapeHtml(initials)}</span>
                                                    </div>
                                                    <div class="bookmark-banner-info">
                                                        <span class="bookmark-banner-name">${escapeHtml(item.name)}</span>
                                                        <span class="bookmark-banner-desc">${escapeHtml(item.desc)}</span>
                                                    </div>
                                                    <span class="bookmark-banner-arrow">➔</span>
                                                </a>
                                            `;
                                        }).join('')}
                                    </div>
                                </div>
                            `;
                        }
                        return `
                            <div class="glance-card widget-bookmarks card-size-${w.card_size} link-size-${w.link_size}">
                                <div class="glance-card-header">
                                    <h2>${escapeHtml(w.title)}</h2>
                                    <span class="glance-badge">${escapeHtml(w.badge)}</span>
                                </div>
                                ${w.bookmarks.map(section => `
                                    <div class="glance-bookmark-sec" style="${section !== w.bookmarks[0] ? 'margin-top: 8px;' : ''}">
                                        <div class="glance-bookmark-title">${escapeHtml(section.section)}</div>
                                        ${section.items.map(item => `
                                            <a class="glance-bookmark-item link-size-${w.link_size}" href="${escapeHtml(item.url)}"${linkTargetAttrs(item.url)}>
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
                        <div class="glance-card widget-${w.type} card-size-${w.card_size} link-size-${w.link_size}" id="widget-${idx}">
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

            const layout = w.layout || 'vertical';

            if (w.type === 'reddit') {
                fetchRedditFeed(w.subreddit, w.urls, w.fallback).then(posts => {
                    const linkSize = w.link_size || 'medium';
                    container.innerHTML = `
                        <ul class="glance-feed-list layout-${layout} link-size-${linkSize}">
                            ${posts.map(post => `
                                <li class="glance-feed-item layout-${layout} link-size-${linkSize}">
                                    <img class="glance-feed-thumb layout-${layout} link-size-${linkSize}" src="${escapeHtml(post.thumbnail)}" onerror="this.src='${REDDIT_PLACEHOLDER}';" alt="Reddit Thumbnail">
                                    <div class="glance-feed-content layout-${layout} link-size-${linkSize}">
                                        <a class="glance-feed-title layout-${layout} link-size-${linkSize}" href="${escapeHtml(post.url)}"${linkTargetAttrs(post.url)}>${escapeHtml(post.title)}</a>
                                        <div class="glance-feed-meta layout-${layout} link-size-${linkSize}">
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
                fetchYouTubeFeed(w.channels, w.playlists, w.urls, w.limit, w.fallback).then(videos => {
                    const linkSize = w.link_size || 'medium';
                    container.innerHTML = `
                        <ul class="glance-feed-list layout-${layout} link-size-${linkSize}">
                            ${videos.map(video => `
                                <li class="glance-feed-item layout-${layout} link-size-${linkSize}">
                                    <img class="glance-feed-thumb layout-${layout} link-size-${linkSize}" src="${escapeHtml(video.thumbnail)}" onerror="this.src='${YOUTUBE_PLACEHOLDER}';" alt="YouTube Thumbnail">
                                    <div class="glance-feed-content layout-${layout} link-size-${linkSize}">
                                        <a class="glance-feed-title layout-${layout} link-size-${linkSize}" href="${escapeHtml(video.url)}"${linkTargetAttrs(video.url)}>${escapeHtml(video.title)}</a>
                                        <div class="glance-feed-meta layout-${layout} link-size-${linkSize}">
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
                const feedUrl = (w.urls && w.urls.length > 0) ? w.urls[0] : w.feedUrl;
                fetchPodcastFeed(feedUrl, w.limit, w.fallback).then(episodes => {
                    const linkSize = w.link_size || 'medium';
                    container.innerHTML = `
                        <ul class="glance-feed-list layout-${layout} link-size-${linkSize}">
                            ${episodes.map(episode => `
                                <li class="glance-feed-item layout-${layout} link-size-${linkSize}">
                                    <img class="glance-feed-thumb layout-${layout} link-size-${linkSize}" src="${escapeHtml(episode.thumbnail)}" onerror="this.src='${PODCAST_PLACEHOLDER}';" alt="Podcast Art">
                                    <div class="glance-feed-content layout-${layout} link-size-${linkSize}">
                                        <a class="glance-feed-title layout-${layout} link-size-${linkSize}" href="${escapeHtml(episode.url)}"${linkTargetAttrs(episode.url)}>${escapeHtml(episode.title)}</a>
                                        <div class="glance-feed-meta layout-${layout} link-size-${linkSize}">
                                            <span class="glance-badge">${escapeHtml(w.badge)}</span>
                                            <span>${escapeHtml(episode.date)}</span>
                                        </div>
                                    </div>
                                </li>
                            `).join('')}
                        </ul>
                    `;
                });
            } else if (w.type === 'spotify') {
                fetchSpotifyFeed(w.urls, w.limit, w.fallback).then(tracks => {
                    const linkSize = w.link_size || 'medium';
                    container.innerHTML = `
                        <ul class="glance-feed-list layout-${layout} link-size-${linkSize}">
                            ${tracks.map(track => `
                                <li class="glance-feed-item layout-${layout} link-size-${linkSize}">
                                    <img class="glance-feed-thumb layout-${layout} link-size-${linkSize}" src="${escapeHtml(track.thumbnail)}" onerror="this.src='${SPOTIFY_PLACEHOLDER}';" alt="Spotify Art">
                                    <div class="glance-feed-content layout-${layout} link-size-${linkSize}">
                                        <a class="glance-feed-title layout-${layout} link-size-${linkSize}" href="${escapeHtml(track.url)}"${linkTargetAttrs(track.url)}>${escapeHtml(track.title)}</a>
                                        <div class="glance-feed-meta layout-${layout} link-size-${linkSize}">
                                            <span class="glance-badge">${escapeHtml(w.badge)}</span>
                                            <span>${escapeHtml(track.date)}</span>
                                        </div>
                                    </div>
                                </li>
                            `).join('')}
                        </ul>
                    `;
                });
            } else if (w.type === 'news') {
                fetchNewsFeed(w.urls, w.limit, w.fallback).then(headlines => {
                    const linkSize = w.link_size || 'medium';
                    container.innerHTML = `
                        <ul class="glance-feed-list layout-${layout} link-size-${linkSize}">
                            ${headlines.map(headline => `
                                <li class="glance-feed-item layout-${layout} link-size-${linkSize}">
                                    <img class="glance-feed-thumb layout-${layout} link-size-${linkSize}" src="${escapeHtml(headline.thumbnail)}" onerror="this.src='${NEWS_PLACEHOLDER}';" alt="News Thumbnail">
                                    <div class="glance-feed-content layout-${layout} link-size-${linkSize}">
                                        <a class="glance-feed-title layout-${layout} link-size-${linkSize}" href="${escapeHtml(headline.url)}"${linkTargetAttrs(headline.url)}>${escapeHtml(headline.title)}</a>
                                        <div class="glance-feed-meta layout-${layout} link-size-${linkSize}">
                                            <span class="glance-badge">${escapeHtml(headline.source || w.badge)}</span>
                                            <span>${escapeHtml(headline.date)}</span>
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
            <p>${escapeHtml(appMode.defaultSelectText)}</p>
        `;
    }

    createMarkdownTabApp({
        key: `${appMode.apiRoot}-${profile}`,
        label: profile,
        emptyLabel: appMode.emptyLabel,
        searchStatusId: 'profilesSearchStatus',
        searchInputId: 'profilesSearchInput',
        refreshButtonId: 'refreshProfilesBtn',
        indexUrl: `/${appMode.indexFile}?profile=${encodeURIComponent(profile)}`,
        docApiUrl: `/api/${appMode.apiRoot}/doc`,
        searchApiUrl: `/api/${appMode.apiRoot}/search?profile=${encodeURIComponent(profile)}`,
        pathPrefix: `${appMode.rootDir}/${profile}`,
        directOpenPageName: appMode.mainPage
    });
}

async function loadWidgetsConfig(profile) {
    try {
        const res = await fetch(`/${appMode.rootDir}/${encodeURIComponent(profile)}/widgets.txt?nocache=${Date.now()}`);
        if (!res.ok) return [];
        const text = await res.text();
        const widgets = [];
        let currentWidget = null;
        const lines = text.split('\n');
        for (let line of lines) {
            line = line.trim();
            if (!line || line.startsWith('#') || line.startsWith(';')) continue;
            if (line.startsWith('[') && line.endsWith(']')) {
                if (currentWidget) {
                    widgets.push(currentWidget);
                }
                const name = line.slice(1, -1).trim();
                currentWidget = {
                    feedKey: name,
                    urls: []
                };
            } else if (currentWidget && line.includes('=')) {
                const parts = line.split('=');
                const key = parts[0].trim().toLowerCase();
                const value = parts.slice(1).join('=').trim();
                if (key === 'url') {
                    currentWidget.urls.push(value);
                } else if (key === 'limit') {
                    currentWidget.limit = parseInt(value, 10) || 5;
                } else {
                    currentWidget[key] = value;
                }
            }
        }
        if (currentWidget) {
            widgets.push(currentWidget);
        }
        return widgets;
    } catch (e) {
        console.warn(`Could not load or parse ${appMode.rootDir}/${profile}/widgets.txt`, e);
        return [];
    }
}

async function startProfiles() {
    if (!isDashboardsMode && await profilesAreLocked()) {
        renderLockedProfiles();
        return;
    }

    const profile = currentProfileName();
    if (!profile) {
        await loadProfilesLanding();
        return;
    }

    document.title = `${appMode.documentTitlePrefix} ${profile}`;

    const params = new URLSearchParams(window.location.search);
    if (params.get('view') === 'notes') {
        renderNotesViewer(profile);
        return;
    }

    // Try loading item-specific dashboard config dynamically
    try {
        const res = await fetch(`/${appMode.rootDir}/${encodeURIComponent(profile)}/dashboard.json?nocache=${Date.now()}`);
        if (res.ok) {
            const config = await res.json();
            const feeds = await loadWidgetsConfig(profile); // Array of parsed widget objects from widgets.txt

            if (feeds.length > 0) {
                // Generate dynamic widgets
                const dynamicWidgets = feeds.map(f => {
                    let targetType = f.type;
                    if (f.type === 'videos' || f.type === 'music') {
                        targetType = 'youtube';
                    }
                    if (targetType === 'bookmarks' || targetType === 'featuring') {
                        const items = (f.urls || []).map(uStr => {
                            const parts = uStr.split('|');
                            if (parts.length >= 3) {
                                return {
                                    name: parts[0].trim(),
                                    desc: parts[1].trim(),
                                    url: parts.slice(2).join('|').trim()
                                };
                            } else if (parts.length === 2) {
                                return {
                                    name: parts[0].trim(),
                                    desc: '',
                                    url: parts[1].trim()
                                };
                            } else {
                                const rawUrl = parts[0].trim();
                                let name = rawUrl;
                                try {
                                    name = new URL(rawUrl).hostname;
                                } catch {}
                                return {
                                    name: name,
                                    desc: '',
                                    url: rawUrl
                                };
                            }
                        });
                        return {
                            type: targetType,
                            feedKey: f.feedKey,
                            title: f.title || f.feedKey,
                            badge: f.badge || (targetType === 'featuring' ? 'Featured' : 'Links'),
                            layout: f.layout || 'horizontal',
                            size: f.size || 'medium',
                            card_size: f.card_size || f.size || 'medium',
                            link_size: f.link_size || f.size || 'medium',
                            bookmarks: [
                                {
                                    section: f.title || f.feedKey,
                                    items: items
                                }
                            ]
                        };
                    }
                    return {
                        type: targetType,
                        feedKey: f.feedKey,
                        title: f.title || f.feedKey,
                        badge: f.badge || (targetType === 'news' ? 'News' : 'RSS'),
                        limit: f.limit || 5,
                        layout: f.layout || 'vertical',
                        size: f.size || 'medium',
                        card_size: f.card_size || f.size || 'medium',
                        link_size: f.link_size || f.size || 'medium',
                        channels: [],
                        playlists: [],
                        urls: f.urls || [],
                        fallback: []
                    };
                });

                // Append bookmarks if it was in the original JSON config and not overridden by widgets.txt
                const bookmarksWidget = config.widgets.find(w => w.type === 'bookmarks');
                const hasWidgetsTxtBookmarks = dynamicWidgets.some(w => w.type === 'bookmarks');
                if (bookmarksWidget && !hasWidgetsTxtBookmarks) {
                    dynamicWidgets.push(bookmarksWidget);
                }
                config.widgets = dynamicWidgets;
            }

            renderDashboard(profile, config);
            return;
        }
    } catch (e) {
        console.warn(`No dashboard configuration found for "${profile}". Falling back to notes viewer.`, e);
    }

    // Default: notes viewer
    renderNotesViewer(profile);
}

startProfiles();
