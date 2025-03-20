import {CSSResultGroup, LitElement, TemplateResult, css, html} from 'lit';
import {property, state} from 'lit/decorators.js';
import {AppLocation, getCurrentLocation} from '../utils/app-router.js';
import {formatOverviewPageUrl} from '../utils/urls.js';
import {Task} from '@lit/task';

export interface Bookmark {
  // Bookmark display name
  name: string;
  // Query for filtering
  query: string;
  // Overview page description
  description?: string;
  // Should display query results in query's order.
  is_ordered?: boolean;
  // Override the num parameter value, if provided.
  override_num_param?: number;
}

interface GetLocationFunction {
  (): AppLocation;
}

export abstract class WebstatusSidebarBookmarkSection extends LitElement {
  /**
   * The default list of bookmarks in this section.
   * This is an abstract method instead of an abstract property so that it can be used in the constructor.
   */
  abstract getDefaultBookmarks(): Bookmark[] | undefined;

  /**
   * An optional task to load bookmarks.
   * Implementers of this abstract class can choose to download bookmarks asynchronously by setting this property.
   */
  _loadBookmarksTask: Task | undefined;
  /**
   * The unique identifier for this section.
   */
  @property({type: String})
  abstract id: string;

  /**
   * The URL pathname that applies to all bookmarks in this section.
   */
  @property({type: String})
  abstract bookmarkPathname: string;

  /**
   * Helper to get the current location.
   */
  getLocation: GetLocationFunction = getCurrentLocation;

  /**
   * The active bookmark query
   */
  @state()
  private activeBookmarkQuery: string | null = null;

  @state()
  bookmarks?: Bookmark[];

  constructor() {
    super();
    window.addEventListener('popstate', this.handlePopState.bind(this));
    const defaultBookmarks = this.getDefaultBookmarks();
    if (defaultBookmarks !== undefined) {
      this.bookmarks = defaultBookmarks;
    }
  }

  private handlePopState() {
    this.updateActiveStatus();
  }

  // Must render to light DOM, so sl-tree works as intended.
  createRenderRoot() {
    return this;
  }

  updateActiveStatus(): void {
    const location = this.getLocation();
    const queryParams = new URLSearchParams(location.search);
    const currentQuery = queryParams.get('q');

    // Check if activeBookmarkQuery needs to be updated
    const newActiveBookmarkQuery =
      this.bookmarks?.find(bookmark => bookmark.query === currentQuery)
        ?.query || null;

    this.activeBookmarkQuery = newActiveBookmarkQuery;
    this.requestUpdate();
  }

  static get styles(): CSSResultGroup {
    return [
      css`
        sl-skeleton {
          width: 10rem;
        }
      `,
    ];
  }

  renderBookmark(bookmark: Bookmark, index: number): TemplateResult {
    const bookmarkId = `${this.id}-bookmark${index}`;
    const currentLocation = this.getLocation();
    const currentURL = new URL(currentLocation.href);

    let bookmarkUrl;
    if (bookmark.override_num_param) {
      bookmarkUrl = formatOverviewPageUrl(currentURL, {
        q: bookmark.query,
        start: 0,
        num: bookmark.override_num_param,
      });
    } else {
      bookmarkUrl = formatOverviewPageUrl(currentURL, {
        q: bookmark.query,
        start: 0,
      });
    }
    // The bookmark should only be active when the path is the FEATURES path
    // and the query is set to the active query.
    const isQueryActive =
      currentURL.pathname === this.bookmarkPathname &&
      new URLSearchParams(currentLocation.search).get('q') ===
        this.activeBookmarkQuery &&
      bookmark.query === this.activeBookmarkQuery;
    const bookmarkIcon = isQueryActive ? 'bookmark-star' : 'bookmark';

    return html`
      <sl-tree-item id=${bookmarkId} ?selected=${isQueryActive}>
        <a class="bookmark-link" href="${bookmarkUrl}">
          <sl-icon name="${bookmarkIcon}"></sl-icon> ${bookmark.name}
        </a>
      </sl-tree-item>
    `;
  }

  render(): TemplateResult {
    return html` ${this.bookmarks?.map((bookmark, index) =>
      this.renderBookmark(bookmark, index),
    )}`;
  }
}
