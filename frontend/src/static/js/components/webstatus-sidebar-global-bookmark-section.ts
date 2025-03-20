import {
  Bookmark,
  WebstatusSidebarBookmarkSection,
} from './webstatus-sidebar-bookmark-section.js';
import {DEFAULT_BOOKMARKS} from '../utils/constants.js';
import {customElement} from 'lit/decorators.js';

@customElement('webstatus-sidebar-global-bookmark-section')
export class WebstatusSidebarGlobalBookmarkSection extends WebstatusSidebarBookmarkSection {
  getDefaultBookmarks(): Bookmark[] | undefined {
    return DEFAULT_BOOKMARKS;
  }
  id: string = 'global-bookmarks';
  bookmarkPathname: string = '/';
}
