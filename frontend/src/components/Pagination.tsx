import type { PaginationMetadata } from "../types/property";

interface PaginationProps {
  metadata: PaginationMetadata;
  currentPage: number;
  onPageChange: (page: number) => void;
}

export function Pagination({ metadata, currentPage, onPageChange }: PaginationProps) {
  const lastPage = metadata.last_page ?? 1;
  if (!metadata.total_listings || lastPage <= 1) return null;

  const canPrev = currentPage > 1;
  const canNext = currentPage < lastPage;

  return (
    <div className="flex flex-col items-center justify-between gap-4 border-t border-neutral-200 pt-6 sm:flex-row">
      <p className="text-sm text-neutral-600">
        Showing page{" "}
        <span className="font-semibold text-neutral-900">
          {metadata.current_page ?? currentPage}
        </span>{" "}
        of{" "}
        <span className="font-semibold text-neutral-900">{lastPage}</span> •{" "}
        <span className="font-semibold text-neutral-900">
          {metadata.total_listings}
        </span>{" "}
        total {metadata.total_listings === 1 ? 'listing' : 'listings'}
      </p>
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={() => canPrev && onPageChange(currentPage - 1)}
          disabled={!canPrev}
          className="btn-secondary disabled:cursor-not-allowed disabled:opacity-50"
        >
          <svg className="mr-1.5 h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
          Previous
        </button>
        <div className="flex items-center gap-1">
          <span className="px-3 py-2 text-sm font-semibold text-neutral-900">
            {currentPage}
          </span>
          <span className="text-sm text-neutral-400">of</span>
          <span className="px-3 py-2 text-sm font-medium text-neutral-600">
            {lastPage}
          </span>
        </div>
        <button
          type="button"
          onClick={() => canNext && onPageChange(currentPage + 1)}
          disabled={!canNext}
          className="btn-secondary disabled:cursor-not-allowed disabled:opacity-50"
        >
          Next
          <svg className="ml-1.5 h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
          </svg>
        </button>
      </div>
    </div>
  );
}


