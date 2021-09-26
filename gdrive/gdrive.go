package gdrive

import (
	"strings"
	"time"

	gocache "github.com/pmylund/go-cache"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"golang.org/x/net/webdav"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const (
	mimeTypeFolder = "application/vnd.google-apps.folder"
)

// NewFS creates new gdrive file system.
func NewFS(ctx context.Context, clientID string, clientSecret string) webdav.FileSystem {
	client, err := drive.NewService(ctx, option.WithHTTPClient(newHTTPClient(ctx, clientID, clientSecret)))
	if err != nil {
		log.Errorf("An error occurred creating Drive client: %v\n", err)
		panic(-3)
	}

	return &fileSystem{
		client: client,
		cache:  gocache.New(5*time.Minute, 30*time.Second),
	}
}

// NewLS creates new GDrive locking system
func NewLS() webdav.LockSystem {
	return webdav.NewMemLS()
}

func getName(file *drive.File) string {
	if file.OriginalFilename != "" {
		return file.OriginalFilename
	} else {
		return file.Name
	}
}

func getModTime(file *drive.File) (time.Time, error) {
	modifiedTime := file.ModifiedTime
	if modifiedTime == "" {
		modifiedTime = file.CreatedTime
	}
	if modifiedTime == "" {
		return time.Time{}, nil
	}

	modTime, err := time.Parse(time.RFC3339, modifiedTime)
	if err != nil {
		return time.Time{}, err
	}

	return modTime, nil
}

func ignoreFile(f *drive.File) bool {
	return f.Trashed
}

func normalizePath(p string) string {
	return strings.TrimRight(p, "/")
}
