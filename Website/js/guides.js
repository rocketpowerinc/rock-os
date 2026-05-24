import { createMarkdownTabApp } from './wiki/markdown-tab.js';

createMarkdownTabApp({
    key: 'guides',
    label: 'Guides',
    emptyLabel: 'guide files',
    searchStatusId: 'guidesSearchStatus',
    searchInputId: 'guidesSearchInput',
    refreshButtonId: 'refreshGuidesBtn',
    indexUrl: 'guides-index.json',
    docApiUrl: '/api/guides/doc',
    searchApiUrl: '/api/guides/search',
    pathPrefix: 'menu/guides',
    directOpenPageName: 'guides.html'
});