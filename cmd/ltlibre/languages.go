// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"net/http"
)

// Language represents a supported language.
type Language struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// supportedLanguages is the static list of supported languages.
var supportedLanguages = []Language{
	{Code: "en", Name: "English"},
	{Code: "es", Name: "Spanish"},
	{Code: "fr", Name: "French"},
	{Code: "de", Name: "German"},
	{Code: "it", Name: "Italian"},
	{Code: "pt", Name: "Portuguese"},
	{Code: "ru", Name: "Russian"},
	{Code: "zh", Name: "Chinese"},
	{Code: "ja", Name: "Japanese"},
	{Code: "ko", Name: "Korean"},
	{Code: "ar", Name: "Arabic"},
	{Code: "hi", Name: "Hindi"},
	{Code: "nl", Name: "Dutch"},
	{Code: "pl", Name: "Polish"},
	{Code: "tr", Name: "Turkish"},
	{Code: "vi", Name: "Vietnamese"},
	{Code: "th", Name: "Thai"},
	{Code: "sv", Name: "Swedish"},
	{Code: "da", Name: "Danish"},
	{Code: "fi", Name: "Finnish"},
	{Code: "no", Name: "Norwegian"},
	{Code: "cs", Name: "Czech"},
	{Code: "el", Name: "Greek"},
	{Code: "he", Name: "Hebrew"},
	{Code: "id", Name: "Indonesian"},
	{Code: "ms", Name: "Malay"},
	{Code: "ro", Name: "Romanian"},
	{Code: "uk", Name: "Ukrainian"},
	{Code: "hu", Name: "Hungarian"},
	{Code: "bg", Name: "Bulgarian"},
}

// handleLanguages handles GET /languages requests.
func (s *server) handleLanguages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	writeJSONResponse(w, http.StatusOK, supportedLanguages)
}
