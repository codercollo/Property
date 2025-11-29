import type { Property } from "../types/property";
import { PropertyCard } from "./PropertyCard";

interface PropertyListProps {
  properties: Property[];
  isLoading?: boolean;
}

export function PropertyList({ properties, isLoading }: PropertyListProps) {
  if (isLoading) {
    return (
      <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: 6 }).map((_, index) => (
          <div
            key={index}
            className="h-[28rem] animate-pulse rounded-lg bg-neutral-100"
          />
        ))}
      </div>
    );
  }

  if (!properties.length) {
    return (
      <div className="flex flex-col items-center justify-center rounded-lg border border-neutral-200 bg-white px-6 py-16 text-center shadow-sm">
        <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-neutral-100">
          <svg
            className="h-7 w-7 text-neutral-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1.5}
              d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"
            />
          </svg>
        </div>
        <h2 className="text-lg font-semibold text-neutral-900">No properties found</h2>
        <p className="mt-2 max-w-md text-sm text-neutral-600">
          Try adjusting your filters or searching in a different area to discover more listings.
        </p>
      </div>
    );
  }

  return (
    <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
      {properties.map((property) => (
        <PropertyCard key={property.id} property={property} />
      ))}
    </div>
  );
}


