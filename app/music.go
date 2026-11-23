package app

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// supported audio extension -> content type
var audioContentTypes = map[string]string{
	".mp3": "audio/mpeg",
}

var (
	ErrMusicNotFound = errors.New("music not found")
	ErrInvalidName   = errors.New("invalid music name")
)

type Music struct {
	Data        *os.File
	Offset      int64
	Size        int64
	ModTime     time.Time
	ContentType string
}

// NewMusicPercent opens the music and records the offset computed from percent.
// The file is not seeked here: the handler serves [Offset, Size) as one
// continuous resource so that Range requests stay consistent with percent.
//
// NOTE: The percent is int64, 42 means 42% instead of 4200%
func NewMusicPercent(name string, percent int64) (*Music, error) {
	// Only plain file names inside DataDir are allowed: this rejects
	// path traversal (../), absolute paths and sub directories.
	if !filepath.IsLocal(name) || filepath.Base(name) != name {
		return nil, fmt.Errorf("%w: %q", ErrInvalidName, name)
	}

	contentType, ok := audioContentTypes[strings.ToLower(filepath.Ext(name))]
	if !ok {
		return nil, fmt.Errorf("%w: %q is not a supported audio file", ErrInvalidName, name)
	}

	f, err := os.OpenFile(filepath.Join(DataDir, name), os.O_RDONLY, 0o644)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s, please visit /list for all available musics", ErrMusicNotFound, name)
		}
		return nil, fmt.Errorf("server internal error: %w", err)
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("error while getting file information: %w", err)
	}

	size := info.Size()
	offset := size * percent / 100
	if offset < 0 {
		offset = 0
	}
	if offset > size {
		offset = size
	}

	slog.Debug("preparing music", "name", name, "size", size, "percent", percent, "offset", offset)

	return &Music{
		Data:        f,
		Offset:      offset,
		Size:        size,
		ModTime:     info.ModTime(),
		ContentType: contentType,
	}, nil
}
