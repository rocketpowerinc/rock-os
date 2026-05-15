const themeStorageKey = 'rock-os-theme';
const defaultTheme = 'rock';
const allowedThemes = [
    'rock',
    'papers',
    'scissors'
];

const legacyThemes = {
    terminal: 'rock',
    'warm-paper': 'papers',
    'blue-glass': 'scissors'
};

function normalizeTheme(theme) {

    if (allowedThemes.includes(theme)) {
        return theme;
    }

    return legacyThemes[theme] || defaultTheme;
}

function applyTheme(theme) {

    const nextTheme =
        normalizeTheme(theme);

    document.documentElement.dataset.theme = nextTheme;

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

const themeSelect =
    document.getElementById('themeSelect');

const initialTheme =
    savedTheme();

applyTheme(initialTheme);

if (themeSelect) {

    themeSelect.value = initialTheme;

    themeSelect.addEventListener('change', () => {
        applyTheme(themeSelect.value);
    });
}
