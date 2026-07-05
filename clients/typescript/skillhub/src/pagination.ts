/** Options for bounded async iteration over paginated endpoints. */
export interface PageIteratorOptions {
  /** Starting page (default 0). */
  page?: number;
  /** Page size (default 20, backend-capped at 100). */
  size?: number;
  /** Maximum pages to fetch before stopping (default 10). */
  maxPages?: number;
}
