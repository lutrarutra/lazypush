package version

import (
	"fmt"
	"strconv"
	"strings"
)

type BumpKind int

const (
	Patch BumpKind = iota
	Minor
	Major
)

type Version struct {
	Major      int
	Minor      int
	Patch      int
	PreRelease string
}

func Parse(tag string) (Version, error) {
	tag = strings.TrimPrefix(tag, "v")
	parts := strings.SplitN(tag, "-", 2)
	nums := strings.Split(parts[0], ".")
	if len(nums) != 3 {
		return Version{}, fmt.Errorf("invalid semver: %q", tag)
	}
	major, err := strconv.Atoi(nums[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major version %q: %w", nums[0], err)
	}
	minor, err := strconv.Atoi(nums[1])
	if err != nil {
		return Version{}, fmt.Errorf("invalid minor version %q: %w", nums[1], err)
	}
	patch, err := strconv.Atoi(nums[2])
	if err != nil {
		return Version{}, fmt.Errorf("invalid patch version %q: %w", nums[2], err)
	}
	prerelease := ""
	if len(parts) > 1 {
		prerelease = parts[1]
	}
	return Version{Major: major, Minor: minor, Patch: patch, PreRelease: prerelease}, nil
}

func (v Version) Bump(kind BumpKind) Version {
	switch kind {
	case Major:
		return Version{Major: v.Major + 1, Minor: 0, Patch: 0}
	case Minor:
		return Version{Major: v.Major, Minor: v.Minor + 1, Patch: 0}
	default:
		return Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch + 1}
	}
}

func (v Version) String() string {
	s := fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.PreRelease != "" {
		s += "-" + v.PreRelease
	}
	return s
}
