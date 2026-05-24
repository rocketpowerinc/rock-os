import { createMarkdownTabApp } from './wiki/markdown-tab.js';

createMarkdownTabApp({
    key: 'bookmarks',
    label: 'Bookmarks',
    emptyLabel: 'bookmarks',
    searchStatusId: 'bookmarksSearchStatus',
    searchInputId: 'bookmarksSearchInput',
    refreshButtonId: 'refreshBookmarksBtn',
    indexUrl: 'bookmarks-index.json',
    docApiUrl: '/api/bookmarks/doc',
    searchApiUrl: '/api/bookmarks/search',
    pathPrefix: 'menu/bookmarks',
    directOpenPageName: 'bookmarks.html'
});