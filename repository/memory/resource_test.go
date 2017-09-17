package memory

import (
	"reflect"
	"testing"

	"github.com/diegobernardes/flare"
)

func TestResourceGenResourceSegments(t *testing.T) {
	tests := []struct {
		name        string
		resources   []flare.Resource
		qtySegments int
		want        [][]string
	}{
		{
			"valid",
			nil,
			0,
			[][]string{},
		},
		{
			"valid",
			[]flare.Resource{
				{Id: "1", Path: "/product/123/stock/{track}"},
				{Id: "2", Path: "/product/{*}/stock/{track}"},
				{Id: "3", Path: "/product/456/stock/{track}"},
			},
			5,
			[][]string{
				{"1", "", "product", "123", "stock", "{track}"},
				{"3", "", "product", "456", "stock", "{track}"},
				{"2", "", "product", "{*}", "stock", "{track}"},
			},
		},
	}

	var r Resource
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.genResourceSegments(tt.resources, tt.qtySegments); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Resource.genResourceSegments() = %v, want %v", got, tt.want)
			}
		})
	}
}
