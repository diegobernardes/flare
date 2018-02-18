// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package url

import "net/url"

// String transform a url.URL into a string and escape the result.
func String(endpoint url.URL) (string, error) {
	return url.QueryUnescape(endpoint.String())
}
