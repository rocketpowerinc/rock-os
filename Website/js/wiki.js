import { createMarkdownTabApp } from './wiki/markdown-tab.js';

createMarkdownTabApp({
    key: 'wiki',
    label: 'Wiki',
    emptyLabel: 'markdown files',
    searchStatusId: 'wikiSearchStatus',
    searchInputId: 'wikiSearchInput',
    refreshButtonId: 'refreshWikiBtn',
    indexUrl: 'wiki-index.json',
    docApiUrl: '/api/wiki/doc',
    searchApiUrl: '/api/wiki/search',
    pathPrefix: 'ENCRYPTED/menu/wiki',
    directOpenPageName: 'wiki.html'
});
