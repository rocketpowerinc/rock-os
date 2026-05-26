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
