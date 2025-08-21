package middlewares

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"schoolapi/pkg/utils"
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

func XSS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// sanitize the URL path
		sanitizedPath, err := clean(r.URL.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		fmt.Println("Original path:", r.URL.Path)
		fmt.Println("sanitizedPath:", sanitizedPath)

		params := r.URL.Query()
		sanitizedQuery := make(map[string][]string)
		for k, values := range params {
			sanitizedKey, err := clean(k)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			var sanitizedValues []string
			for _, value := range values {
				cleanValue, err := clean(value)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				strValue, ok := cleanValue.(string)
				if !ok {
					http.Error(w, "sanitized value is not a string", http.StatusBadRequest)
					return
				}
				sanitizedValues = append(sanitizedValues, strValue)
			}
			sanitizedQuery[sanitizedKey.(string)] = sanitizedValues
		}

		r.URL.Path = sanitizedPath.(string)
		r.URL.RawQuery = url.Values(sanitizedQuery).Encode()

		// Sanitized request body

		if r.Header.Get("Content-Type") == "application/json" {
			if r.Body != nil {
				bodyBytes, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, "error reading request body", http.StatusBadRequest)
					return
				}

				bodyString := strings.TrimSpace(string(bodyBytes))
				// reset the request body
				r.Body = io.NopCloser(bytes.NewReader([]byte(bodyString)))

				if len(bodyString) > 0 {
					var inputData any
					if err := json.NewDecoder(bytes.NewReader([]byte(bodyString))).Decode(&inputData); err != nil {
						http.Error(w, "invalid JSON data", http.StatusBadRequest)
						return
					}
					fmt.Println("Original JSON data:", inputData)

					sanitizedData, err := clean(inputData)
					if err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}

					sanitizedBody, err := json.Marshal(sanitizedData)
					if err != nil {
						http.Error(w, utils.ErrorHandler(err, "error sanitizing body").Error(), http.StatusBadRequest)
						return
					}

					r.Body = io.NopCloser(bytes.NewReader(sanitizedBody))
					fmt.Println("Sanitized body", string(sanitizedBody))

				} else {
					fmt.Println("Request body is empty")
				}
			} else {
				fmt.Println("No body in the request")
			}
		} else if r.Header.Get("Content-Type") != "" {
			log.Printf("Received request with unsupported content type: %s. Expected application/json\n", r.Header.Get("Content-Type"))
			http.Error(w, "Received request with unsupported content type. please use application/json", http.StatusUnsupportedMediaType)
		}

		next.ServeHTTP(w, r)
	})

}

// Sanitize input data to prevent XSS

func clean(data any) (any, error) {
	switch val := data.(type) {
	case map[string]any:
		for k, v := range val {
			val[k] = sanitizeValue(v)
		}
		return val, nil
	case []any:
		for i, v := range val {
			val[i] = sanitizeValue(v)
		}
		return val, nil
	case string:
		return sanitizeString(val), nil
	default:
		return nil, utils.ErrorHandler(fmt.Errorf("unsupported type: %T", data), fmt.Sprintf("unsupported type: %T", data))
	}
}

func sanitizeValue(data any) any {
	switch val := data.(type) {
	case string:
		return sanitizeString(val)
	case map[string]interface{}:
		for k, v := range val {
			val[k] = sanitizeValue(v)
		}
		return val
	case []any:
		for i, v := range val {
			val[i] = sanitizeValue(v)
		}
		return val
	default:
		return val // return the value back because the type is unsupported
	}
}

func sanitizeString(v string) string {
	return bluemonday.UGCPolicy().Sanitize(v)
}
