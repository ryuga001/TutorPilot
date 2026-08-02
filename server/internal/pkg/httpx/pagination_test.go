package httpx

import "testing"

func TestPageOffsetAndLimit(t *testing.T) {
	tests := []struct {
		page, size, wantOffset int
	}{
		{1, 20, 0},
		{2, 20, 20},
		{5, 10, 40},
	}
	for _, tt := range tests {
		p := Page{Page: tt.page, PageSize: tt.size}
		if got := p.Offset(); got != tt.wantOffset {
			t.Errorf("Page{%d,%d}.Offset() = %d, want %d", tt.page, tt.size, got, tt.wantOffset)
		}
		if got := p.Limit(); got != tt.size {
			t.Errorf("Limit() = %d, want %d", got, tt.size)
		}
	}
}

func TestNewPaginatedNeverReturnsNilItems(t *testing.T) {
	got := NewPaginated[string](nil, 0, Page{Page: 1, PageSize: 20})

	if got.Items == nil {
		t.Error("Items is nil; it would serialise as JSON null instead of []")
	}
	if len(got.Items) != 0 {
		t.Errorf("Items = %v, want empty", got.Items)
	}
}

func TestNewPaginatedCarriesPageMetadata(t *testing.T) {
	got := NewPaginated([]int{1, 2, 3}, 57, Page{Page: 3, PageSize: 20})

	if got.Total != 57 || got.Page != 3 || got.PageSize != 20 {
		t.Errorf("metadata = %+v, want total=57 page=3 size=20", got)
	}
	if len(got.Items) != 3 {
		t.Errorf("Items length = %d, want 3", len(got.Items))
	}
}

func TestAtoiDefault(t *testing.T) {
	tests := []struct {
		in   string
		def  int
		want int
	}{
		{"", 20, 20},
		{"abc", 20, 20},
		{"5", 20, 5},
		{"0", 20, 0},
		{"-3", 20, -3},
	}
	for _, tt := range tests {
		if got := atoiDefault(tt.in, tt.def); got != tt.want {
			t.Errorf("atoiDefault(%q, %d) = %d, want %d", tt.in, tt.def, got, tt.want)
		}
	}
}
