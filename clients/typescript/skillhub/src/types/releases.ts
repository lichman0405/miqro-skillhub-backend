/** A skill release. */
export interface Release {
  id: number;
  skillId: number;
  versionId: number;
  channel: string;
  title: string;
  notes?: string;
  draft: boolean;
  prerelease: boolean;
  yanked: boolean;
  publishedAt?: string;
  publisherId: string;
  reviewerId?: string;
  packageHash?: string;
  ciCheckRunId?: string;
}

/** A release asset. */
export interface ReleaseAsset {
  id: number;
  name: string;
  label?: string;
  contentType: string;
  size: number;
  downloadCount: number;
}

/** Paginated release list. */
export interface ReleaseListResult {
  releases: Release[];
  totalCount: number;
  page: number;
  size: number;
}

/** Release with assets. */
export interface ReleaseDetailResponse {
  release: Release;
  assets: ReleaseAsset[];
}

/** Create release request body. */
export interface CreateReleaseRequest {
  versionId: number;
  channel?: string;
  title: string;
  notes?: string;
  draft?: boolean;
  prerelease?: boolean;
}

/** Update release request body. */
export interface UpdateReleaseRequest {
  title?: string;
  notes?: string;
  draft?: boolean;
  prerelease?: boolean;
  yanked?: boolean;
}

/** Release list read model. */
export interface ReleaseListReadModel {
  releases: ReleaseListView[];
  totalCount: number;
  page: number;
  size: number;
  availableActions: ReleaseListActions;
}

/** Release list view (summary). */
export interface ReleaseListView {
  id: number;
  versionId: number;
  channel: string;
  title: string;
  draft: boolean;
  prerelease: boolean;
  yanked: boolean;
  publishedAt?: string;
  publisherId: string;
}

/** Actions for release list page. */
export interface ReleaseListActions {
  canCreateRelease: boolean;
}

/** Release detail read model. */
export interface ReleaseDetailReadModel {
  release: ReleaseDetailView;
  assets?: ReleaseAssetView[];
  availableActions: ReleaseDetailActions;
}

/** Release detail view. */
export interface ReleaseDetailView {
  id: number;
  skillId: number;
  versionId: number;
  channel: string;
  title: string;
  notes?: string;
  draft: boolean;
  prerelease: boolean;
  yanked: boolean;
  publishedAt?: string;
  publisherId: string;
  reviewerId?: string;
  packageHash?: string;
  ciCheckRunId?: string;
}

/** Release asset view. */
export interface ReleaseAssetView {
  id: number;
  name: string;
  label?: string;
  contentType: string;
  size: number;
  downloadCount: number;
}

/** Actions for release detail page. */
export interface ReleaseDetailActions {
  canEdit: boolean;
  canDelete: boolean;
  canYank: boolean;
  canUnYank: boolean;
}
