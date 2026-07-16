export interface PaginationMeta {
  page: number;
  limit: number;
  total: number;
  totalPages: number;
}

export interface RawPaginationMeta {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
}

export function toPaginationMeta(raw: RawPaginationMeta): PaginationMeta {
  return { page: raw.page, limit: raw.limit, total: raw.total, totalPages: raw.total_pages };
}

export const EMPTY_PAGINATION_META: PaginationMeta = { page: 1, limit: 10, total: 0, totalPages: 1 };
