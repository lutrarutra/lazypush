package version_test

import (
	"testing"

	"github.com/lutrarutra/lazypush/internal/version"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input   string
		want    version.Version
		wantErr bool
	}{
		{"v0.9.1", version.Version{Major: 0, Minor: 9, Patch: 1}, false},
		{"0.9.1", version.Version{Major: 0, Minor: 9, Patch: 1}, false},
		{"v1.2.3-rc1", version.Version{Major: 1, Minor: 2, Patch: 3, PreRelease: "rc1"}, false},
		{"v1.0.0", version.Version{Major: 1, Minor: 0, Patch: 0}, false},
		{"", version.Version{}, true},
		{"not-a-version", version.Version{}, true},
	}

	for _, tt := range tests {
		got, err := version.Parse(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("Parse(%q) expected error", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("Parse(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("Parse(%q) = %+v, want %+v", tt.input, got, tt.want)
		}
	}
}

func TestVersionBump(t *testing.T) {
	v := version.Version{Major: 0, Minor: 9, Patch: 1}

	patch := v.Bump(version.Patch)
	if patch.String() != "v0.9.2" {
		t.Errorf("Bump(Patch) = %s, want v0.9.2", patch.String())
	}

	minor := v.Bump(version.Minor)
	if minor.String() != "v0.10.0" {
		t.Errorf("Bump(Minor) = %s, want v0.10.0", minor.String())
	}

	major := v.Bump(version.Major)
	if major.String() != "v1.0.0" {
		t.Errorf("Bump(Major) = %s, want v1.0.0", major.String())
	}
}

func TestVersionString(t *testing.T) {
	tests := []struct {
		v    version.Version
		want string
	}{
		{version.Version{Major: 0, Minor: 9, Patch: 1}, "v0.9.1"},
		{version.Version{Major: 1, Minor: 0, Patch: 0}, "v1.0.0"},
		{version.Version{Major: 1, Minor: 2, Patch: 3, PreRelease: "rc1"}, "v1.2.3-rc1"},
	}
	for _, tt := range tests {
		if got := tt.v.String(); got != tt.want {
			t.Errorf("(%+v).String() = %q, want %q", tt.v, got, tt.want)
		}
	}
}
