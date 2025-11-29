import type { Property } from "../types/property";

interface PropertyCardProps {
  property: Property;
}

export function PropertyCard({ property }: PropertyCardProps) {
  const mainImage = property.images?.[0];
  
  const formatPrice = (price: string | number): string => {
    if (typeof price === 'string') {
      return price;
    }
    return new Intl.NumberFormat("en-KE", {
      style: "currency",
      currency: "KES",
      maximumFractionDigits: 0
    }).format(price);
  };

  return (
    <article className="group flex cursor-pointer flex-col overflow-hidden rounded-lg border border-neutral-200 bg-white shadow-sm transition-all duration-200 hover:shadow-md hover:border-neutral-300">
      <div className="relative h-56 w-full overflow-hidden bg-neutral-100">
        {mainImage ? (
          <img
            src={mainImage}
            alt={property.title}
            className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-105"
          />
        ) : (
          <div className="flex h-full w-full items-center justify-center bg-neutral-100">
            <svg
              className="h-12 w-12 text-neutral-400"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909m-18 3.75h16.5a1.5 1.5 0 001.5-1.5V6a1.5 1.5 0 00-1.5-1.5H3.75A1.5 1.5 0 002.25 6v12a1.5 1.5 0 001.5 1.5zm10.5-11.25h.008v.008h-.008V8.25zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0z"
              />
            </svg>
          </div>
        )}
        <div className="absolute top-3 left-3">
          <span className="rounded-md bg-white/95 backdrop-blur-sm px-2.5 py-1 text-xs font-semibold text-neutral-700 shadow-sm">
            {property.property_type}
          </span>
        </div>
        <div className="absolute bottom-3 right-3">
          <span className="rounded-md bg-primary-600 px-3 py-1.5 text-sm font-bold text-white shadow-md">
            {formatPrice(property.price)}
          </span>
        </div>
      </div>
      
      <div className="flex flex-1 flex-col gap-3 p-5">
        <div>
          <h3 className="line-clamp-2 text-lg font-semibold text-neutral-900 leading-snug">
            {property.title}
          </h3>
          <p className="mt-1.5 line-clamp-1 text-sm text-neutral-600">
            {property.location}
          </p>
        </div>
        
        <div className="flex flex-wrap gap-2">
          <div className="inline-flex items-center gap-1.5 rounded-md bg-neutral-50 px-2.5 py-1.5 text-xs font-medium text-neutral-700">
            <svg className="h-3.5 w-3.5 text-neutral-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
            </svg>
            {property.area} m²
          </div>
          <div className="inline-flex items-center gap-1.5 rounded-md bg-neutral-50 px-2.5 py-1.5 text-xs font-medium text-neutral-700">
            <svg className="h-3.5 w-3.5 text-neutral-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
            </svg>
            {property.bedrooms} beds
          </div>
          <div className="inline-flex items-center gap-1.5 rounded-md bg-neutral-50 px-2.5 py-1.5 text-xs font-medium text-neutral-700">
            <svg className="h-3.5 w-3.5 text-neutral-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 14v3m4-3v3m4-3v3M3 21h18M3 10h18M3 7l9-4 9 4M4 10h16v11H4V10z" />
            </svg>
            {property.bathrooms} baths
          </div>
          <div className="inline-flex items-center gap-1.5 rounded-md bg-neutral-50 px-2.5 py-1.5 text-xs font-medium text-neutral-700">
            <svg className="h-3.5 w-3.5 text-neutral-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            {property.year_built}
          </div>
        </div>
        
        {property.features && property.features.length > 0 && (
          <div className="flex flex-wrap gap-1.5 pt-2 border-t border-neutral-100">
            {property.features.slice(0, 3).map((feature) => (
              <span
                key={feature}
                className="rounded-md bg-primary-50 px-2 py-1 text-xs font-medium text-primary-700 capitalize"
              >
                {feature}
              </span>
            ))}
            {property.features.length > 3 && (
              <span className="rounded-md bg-neutral-50 px-2 py-1 text-xs font-medium text-neutral-600">
                +{property.features.length - 3} more
              </span>
            )}
          </div>
        )}
      </div>
    </article>
  );
}