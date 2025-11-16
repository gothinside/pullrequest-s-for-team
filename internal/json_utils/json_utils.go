package jsonutils

import (
	"encoding/json"
	"net/http"
)

func JsonResponse(w http.ResponseWriter, response map[string]interface{}, ststus int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(ststus)

	json.NewEncoder(w).Encode(response)
}
