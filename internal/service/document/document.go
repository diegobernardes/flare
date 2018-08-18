package document

import "github.com/diegobernardes/flare/internal"

func Newer(a, b internal.Document) bool {
	return a.Revision > b.Revision
}
