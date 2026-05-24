import { createMarkdownTabApp } from './wiki/markdown-tab.js';

createMarkdownTabApp({
    key: 'rocket',
    label: 'Rocket',
    emptyLabel: 'rocket files',
    searchStatusId: 'rocketSearchStatus',
    searchInputId: 'rocketSearchInput',
    refreshButtonId: 'refreshRocketBtn',
    indexUrl: 'rocket-index.json',
    docApiUrl: '/api/rocket/doc',
    searchApiUrl: '/api/rocket/search',
    pathPrefix: 'menu/rocket',
    directOpenPageName: 'rocket.html'
});