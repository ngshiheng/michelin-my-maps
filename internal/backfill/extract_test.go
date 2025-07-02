package backfill

import "testing"

func TestExtractOriginalURL(t *testing.T) {
	cases := []struct {
		name        string
		snapshotURL string
		want        string
	}{
		{
			name:        "basic no trailing slash",
			snapshotURL: "https://web.archive.org/web/20220101000000id_/https://guide.michelin.com/en/greater-london/london/restaurant/helene-darroze-at-the-connaught",
			want:        "https://guide.michelin.com/en/greater-london/london/restaurant/helene-darroze-at-the-connaught",
		},
		{
			name:        "trailing slash",
			snapshotURL: "https://web.archive.org/web/20220101000000id_/https://guide.michelin.com/en/greater-london/london/restaurant/helene-darroze-at-the-connaught/",
			want:        "https://guide.michelin.com/en/greater-london/london/restaurant/helene-darroze-at-the-connaught",
		},
		{
			name:        "no id marker",
			snapshotURL: "https://web.archive.org/web/20220101000000/https://guide.michelin.com/en/greater-london/london/restaurant/helene-darroze-at-the-connaught",
			want:        "",
		},
		{
			name:        "root path",
			snapshotURL: "https://web.archive.org/web/20220101000000id_/https://example.com/",
			want:        "https://example.com/",
		},
		{
			name:        "mixed case scheme and host",
			snapshotURL: "https://web.archive.org/web/20220101000000id_/HTTPS://GUIDE.MICHELIN.COM/en/greater-london/london/restaurant/helene-darroze-at-the-connaught/",
			want:        "https://guide.michelin.com/en/greater-london/london/restaurant/helene-darroze-at-the-connaught",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractOriginalURL(tc.snapshotURL)
			if got != tc.want {
				t.Errorf("extractOriginalURL(%q) = %q, want %q", tc.snapshotURL, got, tc.want)
			}
		})
	}
}
