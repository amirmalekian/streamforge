package downloader

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

type Downloader interface {
	Download(item Item) (Result, error)
}
