// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wildcard

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// Reserved is the list of reserved wildcards.
var Reserved = []string{"revision"}

// Present check if a given content has a wildcard.
func Present(content string) bool {
	return strings.Contains(content, "{") || strings.Contains(content, "}")
}

// Valid check if a given content has valid wildcards.
func Valid(content string) error {
	return valid(content, false)
}

// ValidURL check whenever a endpoint is valid.
func ValidURL(rawEndpoint string) error {
	if err := valid(rawEndpoint, true); err != nil {
		return err
	}

	endpoint, err := url.Parse(rawEndpoint)
	if err != nil {
		panic(err)
	}

	for _, fragment := range strings.Split(endpoint.Path, "/") {
		if fragment == "" {
			continue
		}

		if !Present(fragment) {
			continue
		}

		if fragment[0] == '{' && fragment[len(fragment)-1] == '}' {
			continue
		}

		return fmt.Errorf("wildcard must appear alone at a path fragment '%s'", fragment)
	}

	return nil
}

// Replace the wildcard tag for it's value.
func Replace(wildcard string, content map[string]string) string {
	type position struct {
		start int
		end   int
	}
	positions := make(map[string]position)

	var offset int
	for {
		pos := position{
			start: strings.Index(wildcard[offset:], "{"),
			end:   strings.Index(wildcard[offset:], "}"),
		}

		if pos.start == -1 {
			break
		}
		pos.start += offset
		pos.end += offset + 1

		key := strings.TrimSpace(wildcard[pos.start+1 : pos.end-1])
		positions[key] = pos

		offset = pos.end
	}

	for key, value := range content {
		meta, ok := positions[key]
		if !ok {
			continue
		}

		lenBefore := len(wildcard)
		wildcard = strings.Replace(wildcard, wildcard[meta.start:meta.end], value, -1)
		lenth := len(wildcard) - lenBefore
		if lenth == 0 {
			continue
		}

		delete(positions, key)
		for key, p := range positions {
			if p.end < meta.start {
				continue
			}
			p.start += lenth
			p.end += lenth
			positions[key] = p
		}
	}

	return wildcard
}

// Extract return the list of wildcards.
func Extract(wildcard string) []string {
	type position struct {
		start int
		end   int
	}

	var (
		result  []string
		pos     position
		present = make(map[string]struct{})
	)

	for i, char := range wildcard {
		if char == '{' {
			pos.start = i
		}

		if char == '}' {
			pos.end = i + 1
			key := strings.TrimSpace(wildcard[pos.start+1 : i])

			if _, ok := present[key]; !ok {
				present[key] = struct{}{}
				result = append(result, key)
			}
		}
	}

	return result
}

// ExtractValue is used to extract the wildcard values.
func ExtractValue(a, b string) map[string]string {
	result := make(map[string]string)

	for i := 0; ; i++ {
		if !strings.Contains(a, "{") {
			break
		}

		if a[i] == b[i] {
			continue
		}

		end := strings.Index(a, "}") + 1
		key := a[i+1 : end-1]
		a = a[end:]
		b = b[i:]
		i = -1

		indexa := strings.Index(a, "{")
		if indexa == -1 {
			if len(a) == 0 {
				result[key] = b
			} else {
				result[key] = b[:strings.Index(b, a)]
			}
			continue
		}

		indexb := strings.Index(b, a[:indexa])
		result[key] = b[:indexb]
		b = b[indexb:]
	}

	return result
}

// Normalize is used to normalize the wildcards.
func Normalize(content string) string {
	if content == "" {
		return content
	}

	var offset int
	for {
		start := strings.Index(content[offset:], "{") + offset
		end := strings.Index(content[offset:], "}") + offset

		if start-offset == -1 {
			break
		}
		offset = end

		key := strings.TrimSpace(content[start+1 : end])
		lenBefore := len(content)
		content = content[:start] + "{" + key + "}" + content[end+1:]
		offset += len(content) - lenBefore + 1
	}

	return content
}

func valid(content string, duplicate bool) error {
	var (
		offset  int
		present = make(map[string]struct{})
	)

	for {
		indexStart := strings.Index(content[offset:], "{")
		indexEnd := strings.Index(content[offset:], "}")

		value, err := validContentStruct(content, offset, indexStart, indexEnd)
		if err != nil {
			return err
		}
		if value == "" {
			return nil
		}

		for _, reserved := range Reserved {
			if value == reserved {
				return fmt.Errorf("'%s' is a reserved wildcard", value)
			}
		}

		if duplicate {
			if _, ok := present[value]; ok {
				return fmt.Errorf("'%s' is present more then one time", value)
			}
			present[value] = struct{}{}
		}

		offset += indexEnd + 1
	}
}

func validContentStruct(content string, offset, indexStart, indexEnd int) (string, error) {
	if indexStart == -1 && indexEnd == -1 {
		return "", nil
	}

	if indexStart == -1 && indexEnd != -1 {
		return "", errors.New("missing '{'")
	}

	if indexStart != -1 && indexEnd == -1 {
		return "", errors.New("missing '}'")
	}

	value := strings.TrimSpace(content[offset+indexStart+1 : offset+indexEnd])
	if err := validFragment(content, value, offset, indexStart); err != nil {
		return "", err
	}

	return value, nil
}

func validFragment(content, value string, offset, index int) error {
	if strings.Contains(value, "{") {
		return errors.New("found a '{' inside a wildcard")
	}

	if strings.Contains(value, "}") {
		return errors.New("found a '}' inside a wildcard")
	}

	if value == "" {
		return errors.New("missing the wildcard id")
	}

	if offset > 0 && (content[offset+index] == '{' && content[offset+index-1] == '}') {
		return errors.New("there must be at least one char between wildcard")
	}
	return nil
}
