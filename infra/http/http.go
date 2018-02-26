package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// WriteResponse is used to write the response on http.ResponseWriter.
func WriteResponse(logger log.Logger) func(http.ResponseWriter, interface{}, int, http.Header) {
	logger = log.With(logger, "package", "infra/http")
	logger = level.Error(logger)

	return func(w http.ResponseWriter, r interface{}, status int, headers http.Header) {
		if headers != nil {
			for key, values := range headers {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
		}

		if r == nil {
			w.WriteHeader(status)
			return
		}

		content, err := json.Marshal(r)
		if err != nil {
			logger.Log("error", err.Error(), "message", "error during json.Marshal")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)

		writed, err := w.Write(content)
		if err != nil {
			logger.Log("error", err.Error(), "message", "error during write at http.ResponseWriter")
		}
		if writed != len(content) {
			logger.Log(
				"message",
				fmt.Sprintf("invalid quantity of writed bytes, expected %d and got %d", len(content), writed),
			)
		}
	}
}
