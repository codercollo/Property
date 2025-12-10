package main

import (
	"net/http"

	"github.com/codercollo/property/backend/internal/data"
	"github.com/codercollo/property/backend/internal/validator"
)

// advancedPropertySearchHandler handles advanced property search with multiple filters
func (app *application) advancedPropertySearchHandler(w http.ResponseWriter, r *http.Request) {
	// Define input struct for all search parameters
	var input struct {
		Location     string
		PropertyType string
		Status       string // featured, standard, all
		MinPrice     float64
		MaxPrice     float64
		MinBedrooms  int32
		MaxBedrooms  int32
		MinBathrooms int32
		MaxBathrooms int32
		MinArea      int32
		MaxArea      int32
		Features     []string
		SortBy       string // price, -price, bedrooms, -bedrooms, area, -area, created_at, -created_at
		data.Filters
	}

	// Initialize validator and get query string
	v := validator.New()
	qs := r.URL.Query()

	// Read filter values from query string
	input.Location = app.readString(qs, "location", "")
	input.PropertyType = app.readString(qs, "property_type", "")
	input.Status = app.readString(qs, "status", "all")

	// Price range
	input.MinPrice = float64(app.readInt(qs, "min_price", 0, v))
	input.MaxPrice = float64(app.readInt(qs, "max_price", 0, v))

	// Bedrooms range
	input.MinBedrooms = int32(app.readInt(qs, "min_bedrooms", 0, v))
	input.MaxBedrooms = int32(app.readInt(qs, "max_bedrooms", 0, v))

	// Bathrooms range
	input.MinBathrooms = int32(app.readInt(qs, "min_bathrooms", 0, v))
	input.MaxBathrooms = int32(app.readInt(qs, "max_bathrooms", 0, v))

	// Area range
	input.MinArea = int32(app.readInt(qs, "min_area", 0, v))
	input.MaxArea = int32(app.readInt(qs, "max_area", 0, v))

	// Features/amenities
	input.Features = app.readCSV(qs, "features", []string{})

	// Pagination and sorting
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	input.Filters.Sort = app.readString(qs, "sort", "-created_at")

	// Define allowed sort values
	input.Filters.SortSafelist = []string{
		"id", "price", "bedrooms", "bathrooms", "area", "created_at",
		"-id", "-price", "-bedrooms", "-bathrooms", "-area", "-created_at",
	}

	// Validate filters
	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Validate price range
	if input.MaxPrice > 0 && input.MinPrice > input.MaxPrice {
		v.AddError("max_price", "must be greater than min_price")
	}

	// Validate bedrooms range
	if input.MaxBedrooms > 0 && input.MinBedrooms > input.MaxBedrooms {
		v.AddError("max_bedrooms", "must be greater than min_bedrooms")
	}

	// Validate bathrooms range
	if input.MaxBathrooms > 0 && input.MinBathrooms > input.MaxBathrooms {
		v.AddError("max_bathrooms", "must be greater than min_bathrooms")
	}

	// Validate area range
	if input.MaxArea > 0 && input.MinArea > input.MaxArea {
		v.AddError("max_area", "must be greater than min_area")
	}

	// Validate status
	validStatuses := []string{"all", "featured", "standard"}
	if !validator.In(input.Status, validStatuses...) {
		v.AddError("status", "must be one of: all, featured, standard")
	}

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Create search criteria struct
	searchCriteria := data.PropertySearchCriteria{
		Location:     input.Location,
		PropertyType: input.PropertyType,
		Status:       input.Status,
		MinPrice:     input.MinPrice,
		MaxPrice:     input.MaxPrice,
		MinBedrooms:  input.MinBedrooms,
		MaxBedrooms:  input.MaxBedrooms,
		MinBathrooms: input.MinBathrooms,
		MaxBathrooms: input.MaxBathrooms,
		MinArea:      input.MinArea,
		MaxArea:      input.MaxArea,
		Features:     input.Features,
	}

	// Perform advanced search
	properties, metadata, err := app.models.Properties.AdvancedSearch(searchCriteria, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Return results
	err = app.writeJSON(w, http.StatusOK, envelope{
		"properties": properties,
		"metadata":   metadata,
		"filters":    searchCriteria,
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// getPropertyFiltersHandler returns available filter options
func (app *application) getPropertyFiltersHandler(w http.ResponseWriter, r *http.Request) {
	filters, err := app.models.Properties.GetAvailableFilters()
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"filters": filters}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
