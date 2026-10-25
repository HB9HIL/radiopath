package dem

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type dirLoader struct {
	dir string
	f   format
}

// load reads and parses the named tile; a missing file gives ErrNoTile.
func (l dirLoader) load(name string) (*tile, error) {
	raw, err := os.ReadFile(filepath.Join(l.dir, name+l.f.ext))
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%w: %s", ErrNoTile, name)
	}
	if err != nil {
		return nil, err
	}
	return parseTile(raw, name, l.f)
}

type DirSource struct {
	cachedSource
	dir string
}

// NewDirSource returns a terrain source reading .hgt tiles from dir.
func NewDirSource(dir string, maxTiles int) *DirSource {
	return &DirSource{cachedSource{newTileCache("dem", maxTiles, dirLoader{dir, hgt}.load)}, dir}
}

// Ready reports whether the tile directory can be read.
func (s *DirSource) Ready() error {
	_, err := os.ReadDir(s.dir)
	return err
}
