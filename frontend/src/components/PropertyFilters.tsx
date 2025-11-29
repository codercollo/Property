import { useState } from "react";
import type { PropertyQueryParams } from "../services/properties";

interface PropertyFiltersProps {
  onChange: (filters: Omit<PropertyQueryParams, "page" | "page_size" | "sort">) => void;
}

export function PropertyFilters({ onChange }: PropertyFiltersProps) {
  const [title, setTitle] = useState("");
  const [location, setLocation] = useState("");
  const [propertyType, setPropertyType] = useState("");

  function applyFilters(e: React.FormEvent) {
    e.preventDefault();
    onChange({
      title: title || undefined,
      location: location || undefined,
      property_type: propertyType || undefined
    });
  }

  function clearFilters() {
    setTitle("");
    setLocation("");
    setPropertyType("");
    onChange({});
  }

  return (
    <div className="rounded-lg border border-neutral-200 bg-white p-5 shadow-sm">
      <form onSubmit={applyFilters} className="space-y-4">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div>
            <label htmlFor="title" className="block text-sm font-medium text-neutral-700 mb-1.5">
              Search by title
            </label>
            <input
              id="title"
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Enter property title"
              className="input"
            />
          </div>
          <div>
            <label htmlFor="location" className="block text-sm font-medium text-neutral-700 mb-1.5">
              Location
            </label>
            <input
              id="location"
              type="text"
              value={location}
              onChange={(e) => setLocation(e.target.value)}
              placeholder="City, neighborhood"
              className="input"
            />
          </div>
          <div>
            <label htmlFor="property-type" className="block text-sm font-medium text-neutral-700 mb-1.5">
              Property type
            </label>
            <input
              id="property-type"
              type="text"
              value={propertyType}
              onChange={(e) => setPropertyType(e.target.value)}
              placeholder="Apartment, house, etc."
              className="input"
            />
          </div>
          <div className="flex items-end gap-2">
            <button
              type="submit"
              className="btn-primary flex-1"
            >
              Apply filters
            </button>
            <button
              type="button"
              onClick={clearFilters}
              className="btn-secondary flex-1"
            >
              Clear
            </button>
          </div>
        </div>
      </form>
    </div>
  );
}


