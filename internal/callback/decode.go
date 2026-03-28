package callback

import (
	"encoding/json"
	"errors"
	"net/http"
)

const maxBodyBytes int64 = 64 * 1024 // 64KB

func decodeJSON[T any](r *http.Request, maxBytes int64) (T, error) {
	var v T
	r.Body = http.MaxBytesReader(nil, r.Body, maxBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err != nil {
		return v, err
	}
	if dec.More() {
		return v, errors.New("request body must contain a single JSON object")
	}
	return v, nil
}
