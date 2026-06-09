import { pullLatestRockOS, warnLiveUpdateFailed } from './server-refresh.js';
import { renderProfileWorkspaceNav } from './profile-workspace.js';

const appMode = {
    rootDir: 'ENCRYPTED/Sessions',
    indexFile: 'dashboards-index.json',
    apiRoot: 'dashboards',
    mainPage: 'dashboards.html',
    pageTitle: 'Dashboards',
    landingKicker: 'ENCRYPTED DASHBOARDS',
    landingDescription: '',
    cardDescription: '',
    emptyLabel: 'dashboard files',
    defaultSelectText: 'Select a dashboard document.',
    viewDashboardOverviewText: 'View Dashboard Overview',
    viewHubOverviewText: 'View Hub Overview',
    searchPlaceholder: 'Search dashboards',
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

function bindReloadRefresh(buttonId) {
    const button =
        document.getElementById(buttonId);

    if (!button) {
        return;
    }

    button.addEventListener('click', async () => {
        button.disabled = true;
        button.classList.add('is-refreshing');

        try {
            await pullLatestRockOS();
        }
        catch (err) {
            warnLiveUpdateFailed(err);
        }

        window.location.reload();
    });
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
                <button id="refreshUnavailableBtn" class="command-button primary" type="button">Refresh</button>
            </section>
        `;

        bindReloadRefresh('refreshUnavailableBtn');
    }
}

function currentProfileName() {
    const params = new URLSearchParams(window.location.search);
    const isDashboardsLanding =
        window.location.pathname.toLowerCase().endsWith('/dashboards.html');
    let profile =
        isDashboardsLanding
            ? params.get('dashboard') || ''
            : params.get('dashboard') || '';
    if (!profile) {
        const parts =
            window.location.pathname
                .split('/')
                .filter(Boolean)
                .map(part => decodeURIComponent(part));
        const rootParts =
            appMode.rootDir.split('/');
        const rootIndex =
            parts.findIndex((part, index) =>
                rootParts.every((rootPart, offset) => parts[index + offset] === rootPart)
            );

        if (rootIndex >= 0) {
            const itemParts =
                parts
                    .slice(rootIndex + rootParts.length)
                    .filter(part => part && part !== 'index.html');
            const dashboardIndex =
                itemParts.indexOf('dashboards');

            if (dashboardIndex > 0 && itemParts.length >= dashboardIndex + 3) {
                profile =
                    itemParts.slice(0, dashboardIndex + 3).join('/');
            } else {
                profile =
                    itemParts.join('/');
            }
        }
    }
    return profile;
}

function ownerProfileFromPath(profile) {
    const parts =
        String(profile || '')
            .split('/')
            .filter(Boolean);
    const dashboardIndex =
        parts.indexOf('dashboards');

    return dashboardIndex > 0
        ? parts.slice(0, dashboardIndex).join('/')
        : parts.join('/');
}

function displayNameFromProfile(profile) {
    const parts =
        String(profile || '')
            .split('/')
            .filter(Boolean);

    return parts.length
        ? parts[parts.length - 1]
        : '';
}

function isDashboardProfilePath(profile) {
    return String(profile || '')
        .split('/')
        .filter(Boolean)
        .includes('dashboards');
}

function overviewDocumentName(profile) {
    return isDashboardProfilePath(profile)
        ? 'Dashboard-Overview.md'
        : 'Hub-Overview.md';
}

function overviewDisplayLabel(profile) {
    return isDashboardProfilePath(profile)
        ? 'Dashboard Overview'
        : 'Hub Overview';
}

function viewOverviewButtonText(profile) {
    return isDashboardProfilePath(profile)
        ? appMode.viewDashboardOverviewText
        : appMode.viewHubOverviewText;
}

function kidProfileTheme(profile) {
    const normalized =
        String(profile || '').toLowerCase();

    if (normalized === 'family/profiles/boys') {
        return 'boys';
    }
    if (normalized === 'family/profiles/girls') {
        return 'girls';
    }
    return '';
}

function applyKidProfileTheme(profile) {
    const body =
        document.body;
    if (!body) {
        return;
    }

    body.classList.remove('kid-profile-page', 'kid-profile-boys', 'kid-profile-girls');

    const theme =
        kidProfileTheme(profile);
    if (!theme) {
        return;
    }

    body.classList.add('kid-profile-page', `kid-profile-${theme}`);
}

function profileItemFromPath(path) {
    const parts =
        String(path || '')
            .split('/')
            .filter(Boolean);

    if (parts[0] !== 'ENCRYPTED') {
        return null;
    }

    if (parts[1] !== 'Sessions' || parts[3] !== 'Profiles' || parts.length < 5) {
        return null;
    }

    const workspaceSections =
        new Set(['dashboards', 'bookmarks', 'cheatsheets', 'dotfiles', 'bootstraps', 'scripts', 'wiki']);
    const profileParts = [];
    let sectionIndex = -1;
    for (let index = 2; index < parts.length; index++) {
        if (workspaceSections.has(parts[index])) {
            sectionIndex = index;
            break;
        }
        if (String(parts[index] || '').endsWith('.md')) {
            break;
        }
        profileParts.push(decodeURIComponent(parts[index]));
    }

    if (sectionIndex >= 0 && parts[sectionIndex] === 'dashboards' && parts.length >= sectionIndex + 4) {
        return {
            category: decodeURIComponent(parts[sectionIndex + 1]),
            name: decodeURIComponent(parts[sectionIndex + 2]),
            profile: [
                ...profileParts,
                'dashboards',
                decodeURIComponent(parts[sectionIndex + 1]),
                decodeURIComponent(parts[sectionIndex + 2])
            ].join('/'),
            rootDir: 'ENCRYPTED/Sessions'
        };
    }

    return {
        category: 'Profiles',
        name: profileParts[profileParts.length - 1] || '',
        profile: profileParts.join('/'),
        rootDir: 'ENCRYPTED/Sessions'
    };
}

function profileUrl(profile, rootDir = appMode.rootDir) {
    const path =
        String(profile || '')
            .split('/')
            .filter(Boolean)
            .map(part => encodeURIComponent(part))
            .join('/');

    return `/${rootDir}/${path}/`;
}

function profileFileUrl(profile, fileName) {
    const path =
        String(profile || '')
            .split('/')
            .filter(Boolean)
            .map(part => encodeURIComponent(part))
            .join('/');

    return `/${appMode.rootDir}/${path}/${fileName}`;
}

function uniqueProfileItems(files) {
    const seen = new Map();

    const dashboardCategoryRank = category => {
        const categoryOrder =
            ['profiles', 'os', 'mobile'];
        const index =
            categoryOrder.indexOf(String(category || '').toLowerCase());

        return index === -1
            ? categoryOrder.length
            : index;
    };

    files
        .map(file => profileItemFromPath(file.path || file))
        .filter(Boolean)
        .forEach(item => {
            if (!seen.has(item.profile)) {
                seen.set(item.profile, item);
            }
        });

    return Array.from(seen.values())
        .sort((a, b) => {
            const categoryRankCompare =
                dashboardCategoryRank(a.category) - dashboardCategoryRank(b.category);

            if (categoryRankCompare !== 0) {
                return categoryRankCompare;
            }

            const categoryCompare =
                a.category.toLowerCase().localeCompare(b.category.toLowerCase());

            if (categoryCompare !== 0) {
                return categoryCompare;
            }

            if (a.category.toLowerCase() === 'os') {
                const osOrder = ['windows', 'macos', 'linux'];
                const aIndex = osOrder.indexOf(a.name.toLowerCase());
                const bIndex = osOrder.indexOf(b.name.toLowerCase());
                const aRank = aIndex === -1 ? osOrder.length : aIndex;
                const bRank = bIndex === -1 ? osOrder.length : bIndex;

                if (aRank !== bRank) {
                    return aRank - bRank;
                }
            }

            if (a.category.toLowerCase() === 'profiles') {
                const profileOrder = [
                    'rocket',
                    'admin',
                    'parents',
                    'boys',
                    'girls',
                    'education',
                    'prepper',
                    'offline-vault'
                ];
                const profileRank = name => {
                    const index = profileOrder.indexOf(name);
                    return index === -1 ? profileOrder.length : index;
                };
                const aRank = profileRank(a.name.toLowerCase());
                const bRank = profileRank(b.name.toLowerCase());

                if (aRank !== bRank) {
                    return aRank - bRank;
                }
            }

            if (a.category.toLowerCase() === 'homelab') {
                const aIsSelfHosting = a.name.toLowerCase() === 'selfhosting';
                const bIsSelfHosting = b.name.toLowerCase() === 'selfhosting';

                if (aIsSelfHosting !== bIsSelfHosting) {
                    return aIsSelfHosting ? -1 : 1;
                }
            }

            return a.name.toLowerCase().localeCompare(b.name.toLowerCase());
        });
}

function dashboardLandingItems(files) {
    return uniqueProfileItems(files)
        .filter(item => item.category.toLowerCase() !== 'profiles');
}

function renderProfileCard(item) {
    return `
        <a class="profiles-card" href="${escapeHtml(profileUrl(item.profile, item.rootDir))}" data-profile="${escapeHtml(item.name)}">
            <div class="profile-card-icon"></div>
            <div class="profiles-card-info">
                <span>${escapeHtml(item.name)}</span>
                ${appMode.cardDescription ? `<small>${escapeHtml(appMode.cardDescription)}</small>` : ''}
            </div>
        </a>
    `;
}

function renderProfilesLanding(files, ownerProfile = '') {

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

    const profileItems =
        dashboardLandingItems(files);

    const cardsHtml =
        Array.from(
            profileItems.reduce((groups, item) => {
                if (!groups.has(item.category)) {
                    groups.set(item.category, []);
                }
                groups.get(item.category).push(item);
                return groups;
            }, new Map())
        )
            .map(([category, items]) => `
                <section class="dashboard-category">
                    <h2>${escapeHtml(category)}</h2>
                    <div class="profiles-card-grid">
                        ${items.map(renderProfileCard).join('')}
                    </div>
                </section>
            `)
            .join('');
    const isProfileLanding =
        Boolean(ownerProfile);
    const landingTitle =
        isProfileLanding
            ? 'Profile Based Dashboard'
            : appMode.pageTitle;
    const landingKicker =
        isProfileLanding
            ? ''
            : appMode.landingKicker;

    content.classList.add('fullwidth');
    content.innerHTML = `
        <section class="profiles-dashboard">
            ${landingKicker ? `<p class="wiki-error-kicker">${escapeHtml(landingKicker)}</p>` : ''}
            <h1>${escapeHtml(landingTitle)}</h1>
            ${appMode.landingDescription ? `<p>${escapeHtml(appMode.landingDescription)}</p>` : ''}
            <button id="refreshProfilesLandingBtn" class="command-button primary" type="button">Refresh</button>
            ${cardsHtml}
        </section>
    `;

    bindReloadRefresh('refreshProfilesLandingBtn');
}

async function loadProfilesLanding() {

    try {
        const params =
            new URLSearchParams(window.location.search);
        const ownerProfile =
            String(params.get('profile') || '').trim();
        if (ownerProfile) {
            renderProfileWorkspaceNav(ownerProfile);
        }
        const indexUrl =
            ownerProfile
                ? `/${appMode.indexFile}?profile=${encodeURIComponent(ownerProfile)}&nocache=${Date.now()}`
                : `/${appMode.indexFile}?nocache=${Date.now()}`;
        const response =
            await fetch(indexUrl);

        if (!response.ok) {
            throw new Error(`${appMode.pageTitle} index failed with HTTP ${response.status}`);
        }

        const files =
            await response.json();

        renderProfilesLanding(
            Array.isArray(files) ? files : [],
            ownerProfile
        );
    }
    catch (err) {
        console.warn(err);
        renderDashboardError('The dashboards index could not be loaded. Restart Rock-OS from the latest server source or release binary.');
    }
}

async function dashboardSessionAllows(profile) {
    try {
        const owner =
            String(profile || '').split('/').filter(Boolean)[0] || '';
        const response =
            await fetch(`/${appMode.indexFile}?nocache=` + Date.now());

        if (!response.ok) {
            return false;
        }

        const files =
            await response.json();

        return Array.isArray(files) &&
            files.some(file => String(file?.path || file).startsWith(`ENCRYPTED/Sessions/${owner}/`));
    }
    catch {
        return false;
    }
}

const REDDIT_PLACEHOLDER = '/assets/widget-icons/reddit.png';

const YOUTUBE_PLACEHOLDER = 'data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCA4MCA1MCIgd2lkdGg9IjgwIiBoZWlnaHQ9IjUwIj48cmVjdCB3aWR0aD0iODAiIGhlaWdodD0iNTAiIHJ4PSI0IiBmaWxsPSIjMWExYTI0IiBzdHJva2U9IiMzZTRhNTYiIHN0cm9rZS13aWR0aD0iMSIvPjxwb2x5Z29uIHBvaW50cz0iMzUsMTggNTAsMjUgMzUsMzIiIGZpbGw9IiNmZjAwMDAiLz48L3N2Zz4=';
const PODCAST_PLACEHOLDER = 'data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCA4MCA1MCIgd2lkdGg9IjgwIiBoZWlnaHQ9IjUwIj48cmVjdCB3aWR0aD0iODAiIGhlaWdodD0iNTAiIHJ4PSI0IiBmaWxsPSIjMWExYTI0IiBzdHJva2U9IiMzZTRhNTYiIHN0cm9rZS13aWR0aD0iMSIvPjxjaXJjbGUgY3g9IjQwIiBjeT0iMjAiIHI9IjYiIGZpbGw9Im5vbmUiIHN0cm9rZT0iIzQ2ODJCNCIgc3Ryb2tlLXdpZHRoPSIyIi8+PHJlY3QgeD0iMzciIHk9IjIwIiB3aWR0aD0iNiIgaGVpZ2h0PSI4IiByeD0iMyIgZmlsbD0iIzQ2ODJCNCIvPjxwYXRoIGQ9Ik0gMzQgMjIgQSA4IDggMCAwIDAgNDYgMjIiIGZpbGw9Im5vbmUiIHN0cm9rZT0iIzQ2ODJCNCIgc3Ryb2tlLXdpZHRoPSIyIi8+PGxpbmUgeDE9IjQwIiB5MT0iMzAiIHgyPSI0MCIgeTI9IjM2IiBzdHJva2U9IiM0NjgyQjQiIHN0cm9rZS13aWR0aD0iMiIvPjxsaW5lIHgxPSIzNSIgeTE9IjM2IiB4Mj0iNDUiIHkyPSIzNiIgc3Ryb2tlPSIjNDY4MkI0IiBzdHJva2Utd2lkdGg9IjIiLz48L3N2Zz4=';
const SPOTIFY_PLACEHOLDER = 'data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCA4MCA1MCIgd2lkdGg9IjgwIiBoZWlnaHQ9IjUwIj48cmVjdCB3aWR0aD0iODAiIGhlaWdodD0iNTAiIHJ4PSI0IiBmaWxsPSIjMWExYTI0IiBzdHJva2U9IiMzZTRhNTYiIHN0cm9rZS13aWR0aD0iMSIvPjxjaXJjbGUgY3g9IjQwIiBjeT0iMjUiIHI9IjEyIiBmaWxsPSIjMWRiOTU0Ii8+PHBhdGggZD0iTSAzMiAyNCBDIDM3IDIxIDQzIDIxIDQ4IDI0IE0gMzQgMjcgQyAzOCAyNSA0MiAyNSA0NiAyNyBNIDM2IDMwIEMgMzkgMjkgNDEgMjkgNDQgMzAiIGZpbGw9Im5vbmUiIHN0cm9rZT0iIzEyMTIxMiIgc3Ryb2tlLXdpZHRoPSIxLjUiIHN0cm9rZS1saW5lY2FwPSJyb3VuZCIvPjwvc3ZnPg==';
const NEWS_PLACEHOLDER = 'data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCA4MCA1MCIgd2lkdGg9IjgwIiBoZWlnaHQ9IjUwIj48cmVjdCB3aWR0aD0iODAiIGhlaWdodD0iNTAiIHJ4PSI0IiBmaWxsPSIjMWExYTI0IiBzdHJva2U9IiMzZTRhNTYiIHN0cm9rZS13aWR0aD0iMSIvPjxyZWN0IHg9IjE1IiB5PSIxNSIgd2lkdGg9IjgwIiBoZWlnaHQ9IjIwIiByeD0iMiIgZmlsbD0ibm9uZSIgc3Ryb2tlPSIjNDZCOEQzIiBzdHJva2Utd2lkdGg9IjIiLz48bGluZSB4MT0iMjAiIHkxPSIyMCIgeDI9Ijg1IiB5Mj0iMjAiIHN0cm9rZT0iIzQ2QjhEMyIgc3Ryb2tlLXdpZHRoPSIyIi8+PGxpbmUgeDE9IjIwIiB5MT0iMjUiIHgyPSI2MCIgeTI9IjI1IiBzdHJva2U9IiM0NkI4RDMiIHN0cm9rZS13aWR0aD0iMiIvPjxsaW5lIHgxPSIyMCIgeTE9IjMwIiB4Mj0iNTUiIHkyPSIzMCIgc3Ryb2tlPSIjNDZCOEQzIiBzdHJva2Utd2lkdGg9IjIiLz48L3N2Zz4=';
const GOOGLE_NEWS_PLACEHOLDER = '/assets/widget-icons/google-news.png';

const FILE_ICON_SVG = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="40" height="40" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>';

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
                    <button id="refreshDashboardBtn" class="glance-btn">Refresh</button>
                    <button id="viewOverviewBtn" class="glance-btn">
                        <span>📄</span> ${escapeHtml(viewOverviewButtonText(profile))}
                    </button>
                </div>
            </div>
            <div class="glance-dashboard">
                ${config.widgets.map((w, idx) => {
                    if (w.type === 'files') {
                        return `
                            <div class="glance-card widget-files card-size-${w.card_size} link-size-${w.link_size}">
                                <div class="glance-card-header">
                                    <h2>${escapeHtml(w.title)}</h2>
                                    <span class="glance-badge">${escapeHtml(w.badge)}</span>
                                </div>
                                <div class="glance-files-grid link-size-${w.link_size}">
                                    ${(w.files || []).map(item => {
                                        const copyText = item.copy || item.path;
                                        const hint = item.copy ? 'Click to copy command' : 'Click to copy path';
                                        return `
                                        <button type="button" class="glance-file-card link-size-${w.link_size}${item.desc ? ' has-desc' : ''}" data-path="${escapeHtml(copyText)}" data-hint="${escapeHtml(hint)}">
                                            <span class="glance-file-icon">${FILE_ICON_SVG}</span>
                                            <span class="glance-file-name">${escapeHtml(item.name)}</span>
                                            <span class="glance-file-hint">${escapeHtml(hint)}</span>
                                            ${item.desc ? `<span class="glance-file-tooltip" role="tooltip">${escapeHtml(item.desc)}</span>` : ''}
                                        </button>
                                    `;
                                    }).join('')}
                                </div>
                            </div>
                        `;
                    }
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

        // Wire up click-to-copy for files widgets.
        content.querySelectorAll('.glance-file-card').forEach(btn => {
            btn.addEventListener('click', () => {
                const path = btn.getAttribute('data-path') || '';
                const hint = btn.querySelector('.glance-file-hint');
                const original = hint ? hint.textContent : '';
                const showCopied = () => {
                    btn.classList.add('copied');
                    if (hint) hint.textContent = 'Copied!';
                    setTimeout(() => {
                        btn.classList.remove('copied');
                        if (hint) hint.textContent = original;
                    }, 1500);
                };
                const fallbackCopy = () => {
                    const ta = document.createElement('textarea');
                    ta.value = path;
                    ta.style.position = 'fixed';
                    ta.style.opacity = '0';
                    document.body.appendChild(ta);
                    ta.select();
                    try { document.execCommand('copy'); } catch {}
                    document.body.removeChild(ta);
                    showCopied();
                };
                if (navigator.clipboard && navigator.clipboard.writeText) {
                    navigator.clipboard.writeText(path).then(showCopied).catch(fallbackCopy);
                } else {
                    fallbackCopy();
                }
            });
        });

        // Async load feed content for non-bookmarks widgets
        config.widgets.forEach((w, idx) => {
            if (w.type === 'bookmarks' || w.type === 'files') return;

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
    const viewOverviewBtn = document.getElementById('viewOverviewBtn');
    bindReloadRefresh('refreshDashboardBtn');

    if (viewOverviewBtn) {
        viewOverviewBtn.addEventListener('click', () => {
            renderDashboardOverview(profile, config);
        });
    }
}

async function renderDashboardOverview(profile, config) {
    const displayName =
        displayNameFromProfile(profile);

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
                    <div class="${escapeHtml(config.avatarClass || '')}"></div>
                    <div class="glance-header-text">
                        <h1>${escapeHtml(config.title || displayName)}</h1>
                        <p>${escapeHtml(config.subtitle || 'Dashboard overview')}</p>
                    </div>
                </div>
                <div class="glance-header-actions">
                    <button id="backToDashboardBtn" class="glance-btn">Back to Dashboard</button>
                    <button id="refreshDashboardBtn" class="glance-btn">Refresh</button>
                </div>
            </div>
            <article class="dashboard-overview-card">
                <div class="dashboard-overview-kicker">${escapeHtml(overviewDisplayLabel(profile))}</div>
                <div id="dashboardOverviewContent" class="dashboard-overview-content">
                    <p>Loading overview...</p>
                </div>
            </article>
        `;
    }

    bindReloadRefresh('refreshDashboardBtn');

    const backButton =
        document.getElementById('backToDashboardBtn');
    if (backButton) {
        backButton.addEventListener('click', () => {
            renderDashboard(profile, config);
        });
    }

    const overviewContent =
        document.getElementById('dashboardOverviewContent');
    if (!overviewContent) {
        return;
    }

    const overviewPath =
        `${appMode.rootDir}/${profile}/${overviewDocumentName(profile)}`;
    const docUrl =
        `/api/${appMode.apiRoot}/doc?path=${encodeURIComponent(overviewPath)}&nocache=${Date.now()}`;

    try {
        const res =
            await fetch(docUrl);
        if (!res.ok) {
            throw new Error(`HTTP ${res.status}`);
        }
        const doc =
            await res.json();
        overviewContent.innerHTML =
            doc.html || '<p>No overview content found.</p>';
    } catch (e) {
        console.warn(`Could not load dashboard overview for "${profile}".`, e);
        overviewContent.innerHTML = `
            <p class="wiki-error-kicker">Overview unavailable</p>
            <p>Rock-OS could not load <code>${escapeHtml(overviewPath)}</code>.</p>
        `;
    }
}

async function loadWidgetsConfig(profile) {
    try {
        const res = await fetch(`${profileFileUrl(profile, 'widgets.txt')}?nocache=${Date.now()}`);
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
                    if (!currentWidget.urlMeta) currentWidget.urlMeta = [];
                    currentWidget.urlMeta.push({});
                } else if (key === 'name_of_file' || key === 'file_name' || key === 'filename') {
                    // Start a new file entry (files widget).
                    if (!currentWidget.fileEntries) currentWidget.fileEntries = [];
                    currentWidget.fileEntries.push({ name: value });
                } else if (key === 'path') {
                    // Attach the path to the most recent file entry (files widget).
                    if (currentWidget.fileEntries && currentWidget.fileEntries.length) {
                        currentWidget.fileEntries[currentWidget.fileEntries.length - 1].path = value;
                    } else {
                        // Bare path with no preceding name: start an entry from it.
                        if (!currentWidget.fileEntries) currentWidget.fileEntries = [];
                        currentWidget.fileEntries.push({ path: value });
                    }
                } else if (key === 'desc' || key === 'description') {
                    // Attach to the most recent file entry (or legacy url entry).
                    if (currentWidget.fileEntries && currentWidget.fileEntries.length) {
                        currentWidget.fileEntries[currentWidget.fileEntries.length - 1].desc = value;
                    } else if (currentWidget.urlMeta && currentWidget.urlMeta.length) {
                        currentWidget.urlMeta[currentWidget.urlMeta.length - 1].desc = value;
                    } else {
                        currentWidget[key] = value;
                    }
                } else if (key === 'command' || key === 'cmd' || key === 'copy') {
                    // Attach to the most recent file entry (or legacy url entry).
                    if (currentWidget.fileEntries && currentWidget.fileEntries.length) {
                        currentWidget.fileEntries[currentWidget.fileEntries.length - 1].copy = value;
                    } else if (currentWidget.urlMeta && currentWidget.urlMeta.length) {
                        currentWidget.urlMeta[currentWidget.urlMeta.length - 1].copy = value;
                    } else {
                        currentWidget[key] = value;
                    }
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
    if (await profilesAreLocked()) {
        window.location.replace('/index.html');
        return;
    }

    const profile = currentProfileName();
    if (!profile) {
        applyKidProfileTheme('');
        await loadProfilesLanding();
        return;
    }

    applyKidProfileTheme(profile);

    const displayName =
        displayNameFromProfile(profile);

    if (!await dashboardSessionAllows(profile)) {
        renderDashboardError(`${displayName} is not available in the active Rock-OS dashboard session.`);
        return;
    }

    const profileOwner =
        ownerProfileFromPath(profile);
    if (profileOwner) {
        renderProfileWorkspaceNav(profileOwner);
    }

    document.title = `${appMode.documentTitlePrefix} ${displayName}`;

    const params = new URLSearchParams(window.location.search);

    // Try loading item-specific dashboard config dynamically
    try {
        const res = await fetch(`${profileFileUrl(profile, 'dashboard.json')}?nocache=${Date.now()}`);
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
                    if (targetType === 'files') {
                        let fileItems;
                        if (f.fileEntries && f.fileEntries.length) {
                            // Preferred form: separate name_of_file / path / description / command lines.
                            fileItems = f.fileEntries.map(e => {
                                const path = (e.path || '').trim();
                                const name = (e.name || (path ? path.split(/[\\/]/).pop() : '') || path).trim();
                                return {
                                    name: name,
                                    path: path,
                                    desc: (e.desc || '').trim(),
                                    copy: (e.copy || '').trim()
                                };
                            }).filter(it => it.path);
                        } else {
                            // Legacy form: url = Name | Path | Description | CopyText (with optional
                            // desc/command lines attached via urlMeta).
                            const fileMeta = f.urlMeta || [];
                            fileItems = (f.urls || []).map((uStr, i) => {
                                const parts = uStr.split('|');
                                const m = fileMeta[i] || {};
                                let name, path, desc, copy;
                                if (parts.length >= 2) {
                                    name = parts[0].trim();
                                    path = parts[1].trim();
                                    desc = parts.length >= 3 ? parts[2].trim() : '';
                                    copy = parts.length >= 4 ? parts.slice(3).join('|').trim() : '';
                                } else {
                                    path = parts[0].trim();
                                    name = path.split(/[\\/]/).pop() || path;
                                    desc = '';
                                    copy = '';
                                }
                                if (m.desc) desc = m.desc;
                                if (m.copy) copy = m.copy;
                                return { name: name, path: path, desc: desc, copy: copy };
                            }).filter(it => it.path);
                        }
                        return {
                            type: 'files',
                            feedKey: f.feedKey,
                            title: f.title || f.feedKey,
                            badge: f.badge || 'Files',
                            layout: f.layout || 'horizontal',
                            size: f.size || 'medium',
                            card_size: f.card_size || f.size || 'medium',
                            link_size: f.link_size || f.size || 'medium',
                            files: fileItems
                        };
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

            const requestedView =
                params.get('view');
            if (requestedView === 'overview' || requestedView === 'notes') {
                await renderDashboardOverview(profile, config);
                return;
            }

            renderDashboard(profile, config);
            return;
        }
    } catch (e) {
        console.warn(`No dashboard configuration found for "${profile}".`, e);
    }

    renderDashboardError(`${displayName} does not have a dashboard configuration.`);
}

startProfiles();
