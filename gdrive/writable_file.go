package gdrive

import (
	"bytes"
	"golang.org/x/net/context"
	"os"
	"path"

	log "github.com/sirupsen/logrus"
	"google.golang.org/api/drive/v3"
)

type openWritableFile struct {
	ctx    context.Context
	fs     *fileSystem
	buffer bytes.Buffer
	size   int64
	name   string
}

func (f *openWritableFile) Write(p []byte) (int, error) {
	n, err := f.buffer.Write(p)
	f.size += int64(n)
	return n, err
}

func (f *openWritableFile) Readdir(_ int) ([]os.FileInfo, error) {
	log.Panic("not supported: openWritableFile.Readdir")
	return nil, nil
}

func (f *openWritableFile) Stat() (os.FileInfo, error) {
	return &fileInfo{
		isDir: false,
		size:  f.size,
	}, nil
}

func (f *openWritableFile) Close() error {
	log.Debugf("close %v", f.name)

	fs := f.fs
	fileID, err := fs.getFileID(f.ctx, f.name, false)
	if err != nil && err != os.ErrNotExist {
		log.Error(err)
		return err
	}

	if fileID != "" {
		err = os.ErrExist
		log.Error(err)
		return err
	}

	parent := path.Dir(f.name)
	base := path.Base(f.name)

	parentID, err := fs.getFileID(f.ctx, parent, true)
	if err != nil {
		log.Error(err)
		return err
	}

	if parentID == "" {
		err = os.ErrNotExist
		log.Error(err)
		return err
	}

	file := &drive.File{
		Name:    base,
		Parents: []string{parentID},
	}

	_, err = fs.client.Files.Create(file).Media(&f.buffer).Context(f.ctx).Do()
	if err != nil {
		log.Error(err)
		return err
	}

	fs.invalidatePath(f.name)
	fs.invalidatePath(parent)

	log.Debugf("close successful %s", f.name)
	return nil
}

func (f *openWritableFile) Read(_ []byte) (n int, err error) {
	log.Panic("not supported: openWritableFile.Read")
	return -1, nil
}

func (f *openWritableFile) Seek(_ int64, _ int) (int64, error) {
	log.Panic("not supported: openWritableFile.Seek")
	return -1, nil
}
