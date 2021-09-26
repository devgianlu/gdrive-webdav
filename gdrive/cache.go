package gdrive

import (
	"google.golang.org/api/drive/v3"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	cacheKeyFile = "file:"
)

func (fs *fileSystem) invalidatePath(p string) {
	log.Tracef("invalidatePath %v", p)
	fs.cache.Delete(cacheKeyFile + p)
}

type fileLookupResult struct {
	fp  *drive.File
	err error
}

func (fs *fileSystem) getFile(p string, onlyFolder bool) (*drive.File, error) {
	log.Tracef("getFile %v %v", p, onlyFolder)
	key := cacheKeyFile + p

	if lookup, found := fs.cache.Get(key); found {
		log.Tracef("reusing cached file: %v", p)
		result := lookup.(*fileLookupResult)
		return result.fp, result.err
	}

	fp, err := fs.getFile0(p, onlyFolder)
	lookup := &fileLookupResult{fp: fp, err: err}
	if err != nil {
		fs.cache.Set(key, lookup, time.Minute)
	}

	return lookup.fp, lookup.err
}
