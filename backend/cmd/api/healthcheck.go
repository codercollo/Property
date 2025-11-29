package main

import (
	"net/http"
)

// Handles the health check endpoint and returns a JSON response
func (app *application) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	//Prepare health status and system metadata
	env := envelope{
		"status": "available",
		"system_info": map[string]string{
			"environment": app.config.env,
			"version":     version,
		},
	}

	//Send the JSON response
	err := app.writeJSON(w, http.StatusOK, env, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}
