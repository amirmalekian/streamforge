package downloader

import (
	"context"
)

type Item struct {
	ID    string
	URL   string
	Title string
}

type Result struct {
	ID       string
	FilePath string
	Size     int64
}

type Metadata struct {
	ID       string
	Title    string
	Duration int
	Format   string
	Ext      string
	Size     int64
}

type PlaylistEntry struct {
	ID    string
	URL   string
	Title string
}

type Playlist struct {
	ID      string
	Title   string
	Entries []PlaylistEntry
}

type Downloader interface {
	Download(item Item) (Result, error)
	GetPlaylist(ctx context.Context, url string) (*Playlist, error)
}
