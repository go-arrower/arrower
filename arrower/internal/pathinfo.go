package internal

import (
	"fmt"
	"path/filepath"
	"strings"
)

// PathInfo holds precomputed information about a watched path.
// All fields are public for direct access.
type PathInfo struct {
	AbsPath      string // Absolute filesystem path
	RelativePath string // Normalised relative path ("" if original was ".")
	DisplayPath  string // Path for display ("." if RelativePath is "")
}

// NewPathInfo creates a PathInfo with all values precomputed.
func NewPathInfo(relativePath string) (PathInfo, error) {
	absPath, err := filepath.Abs(relativePath)
	if err != nil {
		return PathInfo{}, fmt.Errorf("invalid path %q: %w", relativePath, err)
	}

	// "." becomes "" for no prefix.
	normalisedRel := relativePath
	if normalisedRel == "." {
		normalisedRel = ""
	}

	// show "." instead of empty string.
	displayPath := normalisedRel
	if displayPath == "" {
		displayPath = "."
	}

	return PathInfo{
		AbsPath:      absPath,
		RelativePath: normalisedRel,
		DisplayPath:  displayPath,
	}, nil
}

// NewPathInfos creates multiple PathInfos from relative paths.
// If any path is invalid, it returns an error indicating which path failed.
func NewPathInfos(relativePaths []string) (PathInfos, error) {
	pathInfos := make(PathInfos, 0, len(relativePaths))
	for _, relPath := range relativePaths {
		pi, err := NewPathInfo(relPath)
		if err != nil {
			return PathInfos{}, fmt.Errorf("failed to create PathInfo for %q: %w", relPath, err)
		}

		pathInfos = append(pathInfos, pi)
	}

	return pathInfos, nil
}

// PathInfos is a slice of PathInfo with helper methods.
type PathInfos []PathInfo

// FormatFilePath converts an absolute file path to a nice display path.
// It finds which PathInfo the file belongs to and formats it accordingly.
// Example: "/home/user/arrower/contexts/auth/..." → "../arrower/contexts/auth/...".
func (pis PathInfos) FormatFilePath(absFilePath string) string {
	for _, pi := range pis {
		if strings.HasPrefix(absFilePath, pi.AbsPath+"/") {
			rel := strings.TrimPrefix(absFilePath, pi.AbsPath+"/")
			if pi.RelativePath != "" {
				return pi.RelativePath + "/" + rel
			}

			return rel
		}
	}

	return absFilePath
}
