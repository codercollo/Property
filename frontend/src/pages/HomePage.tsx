import { useProperties } from "../hooks/useProperties";
import { PropertyFilters } from "../components/PropertyFilters";
import { PropertyList } from "../components/PropertyList";
import { Pagination } from "../components/Pagination";
import { ErrorBanner } from "../components/ErrorBanner";

export function HomePage() {
  const { properties, metadata, loading, error, page, setFilters, goToPage } =
    useProperties();

  return (
    <div className="space-y-8">
      <div className="space-y-3">
        <h1 className="text-3xl font-semibold text-neutral-900 sm:text-4xl">
          Property Listings
        </h1>
        <p className="text-base text-neutral-600 max-w-2xl">
          Browse through available properties and find your perfect home. Use the filters below to narrow down your search.
        </p>
      </div>

      <PropertyFilters onChange={setFilters} />

      {error && <ErrorBanner message={error} />}

      <PropertyList properties={properties} isLoading={loading} />

      <Pagination metadata={metadata} currentPage={page} onPageChange={goToPage} />
    </div>
  );
}

