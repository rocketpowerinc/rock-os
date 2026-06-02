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

async function updateMenuLockState() {
    const menu =
        document.querySelector('.nav-menu');
    const trigger =
        menu?.querySelector('.nav-menu-trigger');
    const list =
        menu?.querySelector('.nav-menu-list');

    if (!menu || !trigger || !list) {
        return;
    }

    function setMenuLocked(locked) {
        menu.classList.toggle('is-locked', locked);
        menu.classList.toggle('is-unlocked', !locked);
        trigger.disabled = locked;
        trigger.setAttribute('aria-disabled', String(locked));
        trigger.title =
            locked ? 'Unlock Rock-OS content to open the menu.' : '';
        list.inert =
            locked;
    }

    // Fail closed until the server explicitly confirms encrypted content is unlocked.
    setMenuLocked(true);

    try {
        const response =
            await fetch('/api/server/status?nocache=' + Date.now());
        const status =
            response.ok ? await response.json() : null;
        const locked =
            status?.gitCrypt !== 'unlocked';

        setMenuLocked(locked);
    }
    catch (err) {
        console.warn('Could not load encrypted content status:', err);
        setMenuLocked(true);
    }
}

updateMenuLockState();
