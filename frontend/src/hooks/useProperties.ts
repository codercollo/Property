import { useEffect, useMemo, useState } from "react";
import {
  fetchProperties,
  type PropertyListResponse,
  type PropertyQueryParams
} from "../services/properties";
import type { Property, PaginationMetadata } from "../types/property";

export function useProperties(initialParams?: PropertyQueryParams) {
  const [properties, setProperties] = useState<Property[]>([]);
  const [metadata, setMetadata] = useState<PaginationMetadata>({});
  const [filters, setFilters] = useState<Omit<PropertyQueryParams, "page" | "page_size">>(
    {
      title: initialParams?.title,
      location: initialParams?.location,
      property_type: initialParams?.property_type
    }
  );
  const [page, setPage] = useState(initialParams?.page ?? 1);
  const [pageSize] = useState(initialParams?.page_size ?? 12);
  const [sort, setSort] = useState(initialParams?.sort ?? "id");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const queryParams: PropertyQueryParams = useMemo(
    () => ({
      ...filters,
      page,
      page_size: pageSize,
      sort
    }),
    [filters, page, pageSize, sort]
  );

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);

    fetchProperties(queryParams)
      .then((data: PropertyListResponse) => {
        if (cancelled) return;
        setProperties(data.properties);
        setMetadata(data.metadata);
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        setError(err instanceof Error ? err.message : "Failed to load properties");
      })
      .finally(() => {
        if (cancelled) return;
        setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [queryParams]);

  function updateFilters(next: Omit<PropertyQueryParams, "page" | "page_size">) {
    setPage(1);
    setFilters(next);
  }

  function goToPage(nextPage: number) {
    setPage(nextPage);
  }

  return {
    properties,
    metadata,
    loading,
    error,
    page,
    pageSize,
    sort,
    filters,
    setSort,
    setFilters: updateFilters,
    goToPage
  };
}


