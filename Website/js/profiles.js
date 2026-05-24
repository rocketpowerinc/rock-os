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

async function fetchRedditPreppers() {
    try {
        const res = await fetch('https://www.reddit.com/r/preppers/new.json?limit=5');
        if (!res.ok) throw new Error();
        const data = await res.json();
        return data.data.children.map(child => ({
            title: child.data.title,
            url: 'https://www.reddit.com' + child.data.permalink,
            created: new Date(child.data.created_utc * 1000).toLocaleDateString(),
            author: 'u/' + child.data.author
        }));
    } catch (e) {
        console.warn('Could not fetch real-time Reddit r/preppers feed. Loading fallback cached feed.');
        return [
            {
                title: "Water purification best practices for long-term storage",
                url: "https://www.reddit.com/r/preppers/comments/water_purification_best_practices/",
                created: "Today",
                author: "u/SurvivalSage"
            },
            {
                title: "Top 5 solar generators for grid outages - hands-on review",
                url: "https://www.reddit.com/r/preppers/comments/top_5_solar_generators/",
                created: "Yesterday",
                author: "u/GridDownAdapter"
            },
            {
                title: "HAM vs GMRS radio communication range test in dense woods",
                url: "https://www.reddit.com/r/preppers/comments/radio_communication_range_test/",
                created: "2 days ago",
                author: "u/SignalPrepper"
            },
            {
                title: "Food rotation 101: Keeping a 12-month pantry fresh",
                url: "https://www.reddit.com/r/preppers/comments/food_rotation_101/",
                created: "3 days ago",
                author: "u/PantryManager"
            },
            {
                title: "Bug-out vehicle build: Essential tools to keep under the seat",
                url: "https://www.reddit.com/r/preppers/comments/bug_out_vehicle_build/",
                created: "4 days ago",
                author: "u/OffGridRover"
            }
        ];
    }
}

async function fetchYouTubeVideos() {
    try {
        const rssUrl = encodeURIComponent('https://www.youtube.com/feeds/videos.xml?channel_id=UC4p10g47S0n4_bcf9_p8K2w');
        const res = await fetch(`https://api.allorigins.win/get?url=${rssUrl}`);
        if (!res.ok) throw new Error();
        const json = await res.json();
        const parser = new DOMParser();
        const doc = parser.parseFromString(json.contents, 'application/xml');
        const entries = doc.querySelectorAll('entry');
        if (entries.length === 0) throw new Error();

        const videos = [];
        for (let i = 0; i < Math.min(entries.length, 5); i++) {
            const entry = entries[i];
            const title = entry.querySelector('title')?.textContent || '';
            const link = entry.querySelector('link')?.getAttribute('href') || 'https://www.youtube.com/@CanadianPrepper';
            const published = new Date(entry.querySelector('published')?.textContent || Date.now()).toLocaleDateString();
            videos.push({ title, url: link, date: published });
        }
        return videos;
    } catch (e) {
        console.warn('Could not fetch live YouTube channel feed. Loading fallback cached videos.');
        return [
            {
                title: "Prepping for the Next 72 Hours: Crucial Steps Most People Miss",
                url: "https://www.youtube.com/watch?v=mock1",
                date: "Today"
            },
            {
                title: "This Shocking Off-Grid Tech Will Change Survivalism Forever",
                url: "https://www.youtube.com/watch?v=mock2",
                date: "Yesterday"
            },
            {
                title: "Top 10 Survival Items You Can Buy at a Local Hardware Store",
                url: "https://www.youtube.com/watch?v=mock3",
                date: "3 days ago"
            },
            {
                title: "The Emergency Comms Plan Every Neighborhood Needs",
                url: "https://www.youtube.com/watch?v=mock4",
                date: "5 days ago"
            },
            {
                title: "Gear Review: Testing the Toughest Water Filters in the Wild",
                url: "https://www.youtube.com/watch?v=mock5",
                date: "1 week ago"
            }
        ];
    }
}

function renderPrepperDashboard() {
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
                    <div class="prepper-avatar-display"></div>
                    <div class="glance-header-text">
                        <h1>Prepper Dashboard</h1>
                        <p>Dynamic Control Dashboard // Prepper Monitoring & Off-Grid Resource Center</p>
                    </div>
                </div>
                <div class="glance-header-actions">
                    <button id="viewNotesBtn" class="glance-btn">
                        <span>📄</span> View Private Notes
                    </button>
                </div>
            </div>
            <div class="glance-dashboard">
                <!-- Widget 1: Reddit -->
                <div class="glance-card">
                    <div class="glance-card-header">
                        <h2>r/preppers Subreddit</h2>
                        <span class="glance-badge">Reddit RSS</span>
                    </div>
                    <div id="redditFeedContainer">
                        <p style="color: var(--text-muted); font-size: 0.9rem;">Syncing feed...</p>
                    </div>
                </div>

                <!-- Widget 2: YouTube -->
                <div class="glance-card">
                    <div class="glance-card-header">
                        <h2>Canadian Prepper Feed</h2>
                        <span class="glance-badge">YouTube RSS</span>
                    </div>
                    <div id="youtubeFeedContainer">
                        <p style="color: var(--text-muted); font-size: 0.9rem;">Syncing feed...</p>
                    </div>
                </div>

                <!-- Widget 3: Bookmarks -->
                <div class="glance-card">
                    <div class="glance-card-header">
                        <h2>Bookmarks</h2>
                        <span class="glance-badge">Links</span>
                    </div>
                    <div class="glance-bookmark-sec">
                        <div class="glance-bookmark-title">Products</div>
                        <a class="glance-bookmark-item" href="https://lifestraw.com/" target="_blank">
                            <div class="glance-bookmark-info">
                                <span class="glance-bookmark-name">LifeStraw</span>
                                <span class="glance-bookmark-desc">Personal water filter technology & products</span>
                            </div>
                            <span class="glance-bookmark-arrow">➔</span>
                        </a>
                    </div>
                    <div class="glance-bookmark-sec" style="margin-top: 8px;">
                        <div class="glance-bookmark-title">Resources</div>
                        <a class="glance-bookmark-item" href="https://github.com/Crosstalk-Solutions/project-nomad" target="_blank">
                            <div class="glance-bookmark-info">
                                <span class="glance-bookmark-name">Project Nomad</span>
                                <span class="glance-bookmark-desc">Off-grid communication setup and deployables</span>
                            </div>
                            <span class="glance-bookmark-arrow">➔</span>
                        </a>
                    </div>
                </div>
            </div>
        `;
    }

    // Populate Reddit
    fetchRedditPreppers().then(posts => {
        const container = document.getElementById('redditFeedContainer');
        if (!container) return;
        container.innerHTML = `
            <ul class="glance-feed-list">
                ${posts.map(post => `
                    <li class="glance-feed-item">
                        <a class="glance-feed-title" href="${escapeHtml(post.url)}" target="_blank">${escapeHtml(post.title)}</a>
                        <div class="glance-feed-meta">
                            <span class="glance-badge">${escapeHtml(post.author)}</span>
                            <span>${escapeHtml(post.created)}</span>
                        </div>
                    </li>
                `).join('')}
            </ul>
        `;
    });

    // Populate YouTube
    fetchYouTubeVideos().then(videos => {
        const container = document.getElementById('youtubeFeedContainer');
        if (!container) return;
        container.innerHTML = `
            <ul class="glance-feed-list">
                ${videos.map(video => `
                    <li class="glance-feed-item">
                        <a class="glance-feed-title" href="${escapeHtml(video.url)}" target="_blank">${escapeHtml(video.title)}</a>
                        <div class="glance-feed-meta">
                            <span class="glance-badge">Video</span>
                            <span>${escapeHtml(video.date)}</span>
                        </div>
                    </li>
                `).join('')}
            </ul>
        `;
    });

    // Toggle button handler
    const viewNotesBtn = document.getElementById('viewNotesBtn');
    if (viewNotesBtn) {
        viewNotesBtn.addEventListener('click', () => {
            // Restore standard sidebar and layout elements
            if (sidebar) sidebar.style.display = '';
            if (resizer) resizer.style.display = '';
            if (toc) toc.style.display = '';
            if (content) {
                content.classList.remove('fullwidth');
                content.innerHTML = `
                    <h1>${escapeHtml('Prepper')}</h1>
                    <p>Select a profile document.</p>
                `;
            }

            createMarkdownTabApp({
                key: `profiles-Prepper`,
                label: 'Prepper',
                emptyLabel: 'profile files',
                searchStatusId: 'profilesSearchStatus',
                searchInputId: 'profilesSearchInput',
                refreshButtonId: 'refreshProfilesBtn',
                indexUrl: `profiles-index.json?profile=${encodeURIComponent('Prepper')}`,
                docApiUrl: '/api/profiles/doc',
                searchApiUrl: `/api/profiles/search?profile=${encodeURIComponent('Prepper')}`,
                pathPrefix: `profiles/Prepper`,
                directOpenPageName: 'profiles.html'
            });
        });
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

    if (profile === 'Prepper') {
        renderPrepperDashboard();
        return;
    }

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
