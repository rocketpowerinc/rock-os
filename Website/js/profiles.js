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

async function startProfiles() {

    if (await profilesAreLocked()) {
        renderLockedProfiles();
        return;
    }

    const profile =
        currentProfileName();

    if (!profile) {
        await loadProfilesLanding();
        return;
    }

    document.title =
        `Rock-OS ${profile}`;

    const heading =
        document.querySelector('.sidebar-header h3');
    const search =
        document.getElementById('profilesSearchInput');
    const content =
        document.getElementById('content');

    if (heading) {
        heading.textContent = profile;
    }
    if (search) {
        search.placeholder = `Search ${profile}`;
        search.setAttribute('aria-label', `Search ${profile}`);
    }
    if (content) {
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

startProfiles();
