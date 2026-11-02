package pagination

import "testing"

func TestPageRequestOffsetSkipsEarlierPages(t *testing.T) {
	cases := []struct {
		name    string
		request PageRequest
		want    int
	}{
		{name: "first page skips nothing", request: PageRequest{Page: 1, Limit: 20}, want: 0},
		{name: "second page skips one page", request: PageRequest{Page: 2, Limit: 20}, want: 20},
		{name: "later page skips every earlier one", request: PageRequest{Page: 5, Limit: 3}, want: 12},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := testCase.request.Offset(); got != testCase.want {
				t.Errorf("%+v.Offset() = %d, want %d", testCase.request, got, testCase.want)
			}
		})
	}
}

func TestNewPageMetadataRoundsTotalPagesUp(t *testing.T) {
	cases := []struct {
		name  string
		total int
		want  int
	}{
		{name: "empty list", total: 0, want: 0},
		{name: "partial page", total: 1, want: 1},
		{name: "exact pages", total: 20, want: 2},
		{name: "trailing partial page", total: 21, want: 3},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			metadata := NewPageMetadata(PageRequest{Page: 2, Limit: 10}, testCase.total)

			want := PageMetadata{Total: testCase.total, Page: 2, Limit: 10, TotalPages: testCase.want}
			if metadata != want {
				t.Errorf("metadata = %+v, want %+v", metadata, want)
			}
		})
	}
}
