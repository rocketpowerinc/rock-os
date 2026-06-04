const themeStorageKey = 'rock-os-theme';
const defaultTheme = 'steel';
const allowedThemes = [
    'steel',
    'rugged',
    'cyberpunk',
    'blue-grass'
];

const themeImages = {
    'cyberpunk': '/assets/Rock-OS-Hero-Cyberpunk.png',
    'rugged': '/assets/Rock-OS-Hero-Rugged.png',
    'steel': '/assets/Rock-OS-Hero-Steel.png',
    'blue-grass': '/assets/Rock-OS-Hero-Blue-Grass.png'
};

function normalizeTheme(theme) {
    if (allowedThemes.includes(theme)) {
        return theme;
    }
    return defaultTheme;
}

function applyTheme(theme) {
    const nextTheme = normalizeTheme(theme);
    const themeImage = themeImages[nextTheme] || themeImages[defaultTheme];

    document.documentElement.dataset.theme = nextTheme;

    document.querySelectorAll('.theme-logo')
        .forEach(image => {
            image.onerror = () => {
                image.onerror = null;
                image.src = themeImages[defaultTheme];
            };
            image.src = themeImage;
        });

    try {
        localStorage.setItem(themeStorageKey, nextTheme);
    }
    catch (err) {
        console.warn('Could not save theme:', err);
    }
}

function savedTheme() {
    try {
        const theme =
            localStorage.getItem(themeStorageKey) ||
            document.documentElement.dataset.theme ||
            defaultTheme;

        return normalizeTheme(theme);
    }
    catch (err) {
        console.warn('Could not load theme:', err);
        return defaultTheme;
    }
}

const themeSelect = document.getElementById('themeSelect');
const initialTheme = savedTheme();

applyTheme(initialTheme);

if (themeSelect) {
    themeSelect.value = initialTheme;
    themeSelect.addEventListener('change', () => {
        applyTheme(themeSelect.value);
    });
}

async function updateNavigationLockState() {
    const homeLinks =
        document.querySelectorAll('.nav-links a[href$="index.html"]');
    const dashboardLinks =
        document.querySelectorAll('.nav-links a[href$="dashboards.html"]');
    const sessionSelect =
        document.getElementById('sessionSelect');

    if (
        homeLinks.length === 0 &&
        dashboardLinks.length === 0 &&
        !sessionSelect
    ) {
        return;
    }

    function setNavigationLocked(locked) {
        document.documentElement.dataset.encryptedNav =
            locked ? 'locked' : 'unlocked';

        homeLinks.forEach(link => {
            link.hidden =
                false;
        });

        dashboardLinks.forEach(link => {
            link.hidden =
                locked;
            link.classList.toggle('is-locked', locked);
            link.classList.toggle('is-unlocked', !locked);
            link.setAttribute('aria-disabled', String(locked));
            link.tabIndex =
                locked ? -1 : 0;
            link.title =
                locked ? 'Unlock Rock-OS content to open dashboards.' : '';
        });

        if (sessionSelect) {
            sessionSelect.hidden =
                locked;
        }
    }

    // Fail closed until the server explicitly confirms encrypted content is unlocked.
    setNavigationLocked(true);

    try {
        const response =
            await fetch('/api/server/status?nocache=' + Date.now());
        const status =
            response.ok ? await response.json() : null;
        const locked =
            status?.gitCrypt !== 'unlocked';

        setNavigationLocked(locked);
    }
    catch (err) {
        console.warn('Could not load encrypted content status:', err);
        setNavigationLocked(true);
    }
}

updateNavigationLockState();
