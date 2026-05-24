import { createMarkdownTabApp } from './wiki/markdown-tab.js';

createMarkdownTabApp({
    key: 'cheatsheets',
    label: 'Cheatsheets',
    emptyLabel: 'cheatsheets',
    searchStatusId: 'cheatsheetsSearchStatus',
    searchInputId: 'cheatsheetsSearchInput',
    refreshButtonId: 'refreshCheatsheetsBtn',
    indexUrl: 'cheatsheets-index.json',
    docApiUrl: '/api/cheatsheets/doc',
    searchApiUrl: '/api/cheatsheets/search',
    pathPrefix: 'menu/cheatsheets',
    directOpenPageName: 'cheatsheets.html'
});