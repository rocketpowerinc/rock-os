const themeStorageKey = 'rock-os-theme';
const defaultTheme = 'steel';
const allowedThemes = [
    'steel',
    'papers',
    'rock',
    'scissors'
];

const legacyThemes = {
    terminal: 'rock',
    'warm-paper': 'papers',
    'blue-glass': 'scissors'
};

const themeImages = {
    rock: 'assets/Rock-OS-Hero-Cyberpunk.png',
    papers: 'assets/Rock-OS-Hero-Rugged.png',
    steel: 'assets/Rock-OS-Hero-Steel.png',
    scissors: 'assets/Rock-OS-Hero-Blue-Grass.png'
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

    document.querySelectorAll('.theme-logo')
        .forEach(image => {
            image.src = themeImages[nextTheme];
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
