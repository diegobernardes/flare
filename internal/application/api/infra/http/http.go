package http

import (
	"fmt"
	"net/url"
)

// CheckPermitedQS validate if the current query has only the permited options.
func CheckPermitedQS(query url.Values, permited []string) error {
	for key := range query {
		var found bool

		for _, value := range permited {
			if key == value {
				found = true
			}
		}

		if !found {
			return fmt.Errorf("invalid query '%s'", key)
		}
	}

	return nil
}
