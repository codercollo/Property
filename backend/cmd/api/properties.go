package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/codercollo/property/backend/internal/data"
	"github.com/codercollo/property/backend/internal/validator"
)

// createPropertyHandler handles creating a new property
func (app *application) createPropertyHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title        string   `json:"title"`
		YearBuilt    int32    `json:"year_built"`
		Area         int32    `json:"area"`
		Bedrooms     int32    `json:"bedrooms"`
		Bathrooms    int32    `json:"bathrooms"`
		Floor        int32    `json:"floor"`
		Price        float64  `json:"price"`
		Location     string   `json:"location"`
		PropertyType string   `json:"property_type"`
		Features     []string `json:"features"`
		Images       []string `json:"images"`
		// Note: NO agent_id field - we get it from the authenticated user
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Get the authenticated user
	user := app.contextGetUser(r)

	// Create property with values from input and authenticated user
	property := &data.Property{
		Title:        input.Title,
		YearBuilt:    input.YearBuilt,
		Area:         data.Area(input.Area),
		Bedrooms:     data.Bedrooms(input.Bedrooms),
		Bathrooms:    data.Bathrooms(input.Bathrooms),
		Floor:        data.Floor(input.Floor),
		Price:        data.Price(input.Price),
		Location:     input.Location,
		PropertyType: input.PropertyType,
		Features:     input.Features,
		Images:       input.Images,
		AgentID:      sql.NullInt64{Int64: user.ID, Valid: true},
	}

	v := validator.New()
	if data.ValidateProperty(v, property); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Properties.Insert(property)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/properties/%d", property.ID))

	err = app.writeJSON(w, http.StatusCreated, envelope{"property": property}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// showPropertyHandler returns a property by ID as a JSON response
func (app *application) showPropertyHandler(w http.ResponseWriter, r *http.Request) {
	//Read and validate the "id" parameter from the URL
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	//Fetch property by ID
	property, err := app.models.Properties.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrPropertyNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	//Send JSON response
	err = app.writeJSON(w, http.StatusOK, envelope{"property": property}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// updatePropertyHandler updates an existing property and returning the new property data
func (app *application) updatePropertyHandler(w http.ResponseWriter, r *http.Request) {
	//Get the property ID from the URL
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	//Retrieve the existing property from the database
	property, err := app.models.Properties.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrPropertyNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	//Struct to capture incomming JSON updates
	var input struct {
		Title        *string         `json:"title"`
		YearBuilt    *int32          `json:"year_built"`
		Area         *data.Area      `json:"area"`
		Bedrooms     *data.Bedrooms  `json:"bedrooms"`
		Bathrooms    *data.Bathrooms `json:"bathrooms"`
		Floor        *data.Floor     `json:"floor"`
		Price        *data.Price     `json:"price"`
		Location     *string         `json:"location"`
		PropertyType *string         `json:"property_type"`
		Features     []string        `json:"features"`
		Images       []string        `json:"images"`
	}

	//Decode JSON request into the input struct
	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Update only the fields provided
	if input.Title != nil {
		property.Title = *input.Title
	}
	if input.YearBuilt != nil {
		property.YearBuilt = *input.YearBuilt
	}
	if input.Area != nil {
		property.Area = *input.Area
	}
	if input.Bedrooms != nil {
		property.Bedrooms = *input.Bedrooms
	}
	if input.Bathrooms != nil {
		property.Bathrooms = *input.Bathrooms
	}
	if input.Floor != nil {
		property.Floor = *input.Floor
	}
	if input.Price != nil {
		property.Price = *input.Price
	}
	if input.Location != nil {
		property.Location = *input.Location
	}
	if input.PropertyType != nil {
		property.PropertyType = *input.PropertyType
	}
	// Slices: check for nil to determine if provided
	if input.Features != nil {
		property.Features = input.Features
	}
	if input.Images != nil {
		property.Images = input.Images
	}

	//Validate the updated property
	v := validator.New()
	if data.ValidateProperty(v, property); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	//Check version header for edit conflicts before updating
	if r.Header.Get("X-Expected-Version") != "" {
		expectedVersion, err := strconv.ParseInt(r.Header.Get("X-Expected-Version"), 10, 32)
		if err != nil || property.Version != int32(expectedVersion) {
			app.editConflictResponse(w, r)
			return
		}
	}

	//Save changes using Update method
	err = app.models.Properties.Update(property)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	//Return the updated property in the response
	err = app.writeJSON(w, http.StatusOK, envelope{"property": property}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}

// deletePropertyHandler deletes a property by ID and returns a success message
func (app *application) deletePropertyHandler(w http.ResponseWriter, r *http.Request) {
	//Get property by ID from URL
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	//Attempt to delete the property from the database
	err = app.models.Properties.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrPropertyNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	//Respond with success message
	err = app.writeJSON(w, http.StatusOK, envelope{"message": "property successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}

// listPropertyHandler reads query parameters for filtering, pagination, and sorting of properties
func (app *application) listPropertiesHandler(w http.ResponseWriter, r *http.Request) {
	//An anonymous struct to hold filter and pagination parameters from the query string
	var input struct {
		Title        string
		Location     string
		PropertyType string
		Features     []string
		data.Filters
	}

	//Initialize a new validator instance
	//Get query string values
	v := validator.New()
	qs := r.URL.Query()

	//Read filter values from the query string
	input.Title = app.readString(qs, "title", "")
	input.Location = app.readString(qs, "location", "")
	input.PropertyType = app.readString(qs, "property_type", "")
	input.Features = app.readCSV(qs, "features", []string{})

	//Read pagination and sorting values from query string
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	input.Filters.Sort = app.readString(qs, "sort", "id")

	//Define a whitelist of allowed sort values to prevent SQL injection
	input.Filters.SortSafelist = []string{
		"id", "title", "year_built", "price", "-id", "-title", "-year_built", "-price",
	}

	//Validate the filters(page, page_size, sort)
	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	//Fetch filtered, sorted and paginated properties from the database
	properties, metadata, err := app.models.Properties.GetAll(
		input.Title,
		input.Location,
		input.PropertyType,
		input.Features,
		input.Filters,
	)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	//Return the properties as a JSON response
	err = app.writeJSON(w, http.StatusOK, envelope{"properties": properties, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// featurePropertyHandler features properties, validate ID, handle errors and returns JSON
func (app *application) featurePropertyHandler(w http.ResponseWriter, r *http.Request) {
	//Get property ID from URL
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	//Mark property as featured
	err = app.models.Properties.Feature(id)
	if err != nil {
		switch err {
		case data.ErrPropertyNotFound:
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	//Fetch the updated property
	property, err := app.models.Properties.Get(id)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.writeJSON(w, http.StatusOK, envelope{"property": property}, nil)
}

// UnfeaturePropertyHandler unfeatures properties
func (app *application) unfeaturePropertyHandler(w http.ResponseWriter, r *http.Request) {
	//Get property ID from URL
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	//Clear featured status
	err = app.models.Properties.Unfeature(id)
	if err != nil {
		switch err {
		case data.ErrPropertyNotFound:
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	property, err := app.models.Properties.Get(id)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.writeJSON(w, http.StatusOK, envelope{"property": property}, nil)
}
