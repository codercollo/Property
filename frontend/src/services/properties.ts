import type { Property, PaginationMetadata } from "../types/property";

const DEFAULT_BASE_URL = "http://localhost:4000";

function getBaseUrl() {
  return import.meta.env.VITE_API_BASE_URL || DEFAULT_BASE_URL;
}

export interface PropertyQueryParams {
  title?: string;
  location?: string;
  property_type?: string;
  features?: string[];
  page?: number;
  page_size?: number;
  sort?: string;
}

export interface PropertyListResponse {
  properties: Property[];
  metadata: PaginationMetadata;
}

function buildQueryString(params: PropertyQueryParams): string {
  const search = new URLSearchParams();

  if (params.title) search.set("title", params.title);
  if (params.location) search.set("location", params.location);
  if (params.property_type) search.set("property_type", params.property_type);
  if (params.features && params.features.length > 0) {
    search.set("features", params.features.join(","));
  }
  if (params.page) search.set("page", String(params.page));
  if (params.page_size) search.set("page_size", String(params.page_size));
  if (params.sort) search.set("sort", params.sort);

  const qs = search.toString();
  return qs ? `?${qs}` : "";
}

async function handleResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const message = await res.text();
    throw new Error(message || `Request failed with status ${res.status}`);
  }
  return res.json() as Promise<T>;
}

export async function fetchProperties(
  params: PropertyQueryParams = {}
): Promise<PropertyListResponse> {
  const qs = buildQueryString(params);
  const res = await fetch(`${getBaseUrl()}/v1/properties${qs}`);
  const data = await handleResponse<{
    properties: Property[];
    metadata: PaginationMetadata;
  }>(res);
  return {
    properties: data.properties ?? [],
    metadata: data.metadata ?? {}
  };
}

export async function fetchProperty(id: number): Promise<Property> {
  const res = await fetch(`${getBaseUrl()}/v1/properties/${id}`);
  const data = await handleResponse<{ property: Property }>(res);
  return data.property;
}


