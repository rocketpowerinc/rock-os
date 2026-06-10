(function () {
    const sections = [
        { key: 'dashboards', label: 'Dashboards' },
        { key: 'bookmarks', label: 'Bookmarks' },
        { key: 'cheatsheets', label: 'Cheatsheets' },
        { key: 'dotfiles', label: 'Dotfiles' },
        { key: 'bootstraps', label: 'Bootstraps' },
        { key: 'scripts', label: 'Scripts' },
        { key: 'wiki', label: 'Wiki' }
    ];

    function encodeProfile(profile) {
        return String(profile || '')
            .split('/')
            .filter(Boolean)
            .map(part => encodeURIComponent(part))
            .join('/');
    }

    function currentProfile() {
        const params =
            new URLSearchParams(window.location.search);
        const queryProfile =
            String(params.get('profile') || '').trim();
        if (queryProfile) {
            return queryProfile;
        }

        const parts =
            window.location.pathname
                .split('/')
                .filter(Boolean)
                .map(part => decodeURIComponent(part));
        const sessionsIndex =
            parts.findIndex((part, index) =>
                part === 'Sessions' &&
                parts[index - 1] === 'ENCRYPTED'
            );
        if (sessionsIndex < 0) {
            return '';
        }

        const stopSections =
            new Set(sections.map(section => section.key));
        const profileParts = [];
        for (const part of parts.slice(sessionsIndex + 1)) {
            if (stopSections.has(part) || part === 'index.html') {
                break;
            }
            profileParts.push(part);
        }
        return profileParts.join('/');
    }

    function kidTheme(profile) {
        const normalized =
            String(profile || '').toLowerCase();
        if (normalized === 'public/family/profiles/boys') {
            return 'boys';
        }
        if (normalized === 'public/family/profiles/girls') {
            return 'girls';
        }
        return '';
    }

    function signatureFor(links) {
        return links
            .map(link => `${link.key}:${link.href}:${link.active ? '1' : '0'}`)
            .join('|');
    }

    function renderKidProfileSwitch(theme) {
        const navLinks =
            document.querySelector('.nav-links');
        if (!navLinks || !theme) {
            return;
        }

        const target =
            theme === 'boys'
                ? {
                    label: 'Girls',
                    href: '/ENCRYPTED/Sessions/Public/Family/Profiles/Girls/'
                }
                : {
                    label: 'Boys',
                    href: '/ENCRYPTED/Sessions/Public/Family/Profiles/Boys/'
                };

        let link =
            navLinks.querySelector('.kid-profile-switch');
        if (!link) {
            link =
                document.createElement('a');
            link.className =
                'kid-profile-switch';
            const homeLink =
                navLinks.querySelector('a[href$="index.html"]');
            navLinks.insertBefore(link, homeLink || navLinks.firstChild);
        }

        link.href =
            target.href;
        link.textContent =
            target.label;
        link.setAttribute('aria-label', `Open ${target.label} profile`);
    }

    function renderKidsLockButton(theme) {
        const navLinks =
            document.querySelector('.nav-links');
        if (!navLinks || !theme) {
            return;
        }

        let button =
            navLinks.querySelector('.kids-lock-button');
        if (!button) {
            button =
                document.createElement('button');
            button.className =
                'kids-lock-button';
            button.type =
                'button';
            button.textContent =
                '🔒';
            button.title =
                'Lock Rock-OS to the Family session';
            button.setAttribute('aria-label', 'Lock Rock-OS to the Family session');
            const homeLink =
                navLinks.querySelector('a[href$="index.html"]');
            navLinks.insertBefore(button, homeLink || navLinks.firstChild);
        }

        fetch('/api/kids-lock?nocache=' + Date.now())
            .then(response => response.ok ? response.json() : null)
            .then(status => {
                if (!status?.locked) {
                    return;
                }
                const homeLink =
                    navLinks.querySelector('a[href$="index.html"]');
                if (homeLink) {
                    homeLink.remove();
                }
                button.disabled =
                    true;
                button.title =
                    'Kids lock is active';
                button.setAttribute('aria-label', 'Kids lock is active');
            })
            .catch(() => {});

        if (button.dataset.lockHandlerBound === 'true') {
            return;
        }
        button.dataset.lockHandlerBound =
            'true';

        button.addEventListener('click', async () => {
            button.disabled =
                true;
            try {
                const response =
                    await fetch('/api/kids-lock', {
                        method: 'POST',
                        headers: {
                            'X-Rock-OS-Requested': 'true'
                        }
                    });
                if (!response.ok) {
                    throw new Error(await response.text());
                }
                window.location.reload();
            }
            catch (err) {
                button.disabled =
                    false;
                window.alert(`Rock-OS could not enable kids lock.\n\n${err.message}`);
            }
        });
    }

    const profile =
        currentProfile();
    const currentPath =
        window.location.pathname.toLowerCase();
    const isProfileDashboardSurface =
        Boolean(profile) &&
        (
            currentPath.endsWith('/dashboards.html') ||
            currentPath.includes('/encrypted/sessions/')
        );
    const theme =
        kidTheme(profile);

    if (!profile) {
        return;
    }

    document.documentElement.classList.add('profile-workspace-page');
    document.body?.classList.add('profile-workspace-page');

    if (isProfileDashboardSurface) {
        document.documentElement.classList.add('profile-dashboard-page');
        document.body?.classList.add('profile-dashboard-page');
    }

    if (theme) {
        document.documentElement.classList.add('kid-profile-page', `kid-profile-${theme}`);
        document.body?.classList.add('kid-profile-page', `kid-profile-${theme}`);
        renderKidProfileSwitch(theme);
        renderKidsLockButton(theme);
    }

    const encodedProfile =
        encodeProfile(profile);
    const profilePath =
        `/ENCRYPTED/Sessions/${encodedProfile}/`;
    const profileDashboardPath =
        `/encrypted/sessions/${encodedProfile.toLowerCase()}/dashboards/`;
    const links = [
        {
            key: 'overview',
            label: 'Hub',
            href: profilePath,
            active: currentPath.includes('/encrypted/sessions/') && !currentPath.includes('/dashboards/')
        },
        ...sections.map(section => ({
            ...section,
            href: section.key === 'dashboards'
                ? `/dashboards.html?profile=${encodeURIComponent(profile)}`
                : `/${section.key}.html?profile=${encodeURIComponent(profile)}`,
            active: section.key === 'dashboards'
                ? currentPath.endsWith('/dashboards.html') || currentPath.includes(profileDashboardPath)
                : currentPath.endsWith(`/${section.key}.html`)
        }))
    ];
    const signature =
        signatureFor(links);

    let nav =
        document.getElementById('profileWorkspaceNav');
    if (!nav) {
        nav =
            document.createElement('nav');
        nav.id =
            'profileWorkspaceNav';
        nav.className =
            'profile-workspace-nav';
        nav.setAttribute('aria-label', `${profile} profile workspace`);
        document.querySelector('.navbar')?.insertAdjacentElement('afterend', nav);
    }
    if (!nav || nav.dataset.navSignature === signature) {
        return;
    }

    nav.dataset.navSignature =
        signature;
    nav.replaceChildren(
        ...links.map(link => {
            const anchor =
                document.createElement('a');
            anchor.href =
                link.href;
            anchor.textContent =
                link.label;
            anchor.classList.toggle('is-active', link.active);
            if (link.active) {
                anchor.setAttribute('aria-current', 'page');
            }
            return anchor;
        })
    );
}());
