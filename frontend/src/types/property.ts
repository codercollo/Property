export interface Property {
  id: number;
  title: string;
  year_built: number;
  area: number;
  bedrooms: number;
  bathrooms: number;
  floor: number;
  price: string | number;
  location: string;
  property_type: string;
  features: string[];
  images: string[];
  version: number;
}

export interface PaginationMetadata {
  current_page?: number;
  page_size?: number;
  first_page?: number;
  last_page?: number;
  total_listings?: number;
}


