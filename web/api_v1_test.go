package web

import "testing"

func TestIsV1APIPathHonorsBasePath(t *testing.T) {
	tests := []struct {
		name     string
		basePath string
		path     string
		want     bool
	}{
		{name: "root live path", basePath: "/", path: "/api/v1/health/live", want: true},
		{name: "root exact path", basePath: "/", path: "/api/v1", want: true},
		{name: "root avoids false prefix", basePath: "/", path: "/api/v10", want: false},
		{name: "base path live path", basePath: "/super/", path: "/super/api/v1/health/live", want: true},
		{name: "base path exact path", basePath: "/super/", path: "/super/api/v1", want: true},
		{name: "base path avoids sibling", basePath: "/super/", path: "/api/v1/health/live", want: false},
		{name: "base path avoids false prefix", basePath: "/super/", path: "/super/api/v10", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isV1APIPath(tt.path, tt.basePath); got != tt.want {
				t.Fatalf("isV1APIPath(%q, %q) = %v, want %v", tt.path, tt.basePath, got, tt.want)
			}
		})
	}
}
