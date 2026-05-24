import { createMarkdownTabApp } from './wiki/markdown-tab.js';

createMarkdownTabApp({
    key: 'dotfiles',
    label: 'Dotfiles',
    emptyLabel: 'dotfiles',
    searchStatusId: 'dotfilesSearchStatus',
    searchInputId: 'dotfilesSearchInput',
    refreshButtonId: 'refreshDotfilesBtn',
    indexUrl: 'dotfiles-index.json',
    docApiUrl: '/api/dotfiles/doc',
    searchApiUrl: '/api/dotfiles/search',
    pathPrefix: 'menu/dotfiles',
    directOpenPageName: 'dotfiles.html'
});