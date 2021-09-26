package gdrive

import (
	"fmt"
	"net/http"
	"os"
	"path"

	gocache "github.com/pmylund/go-cache"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"golang.org/x/net/webdav"
	"google.golang.org/api/drive/v3"
)

type fileSystem struct {
	client       *drive.Service
	roundTripper http.RoundTripper
	cache        *gocache.Cache
}

func (fs *fileSystem) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	log.Debugf("mkdir %v %v", name, perm)

	name = normalizePath(name)
	pID, err := fs.getFileID(ctx, name, false)
	if err != nil && err != os.ErrNotExist {
		log.Error(err)
		return err
	}
	if err == nil {
		log.Errorf("dir already exists: %v", pID)
		return os.ErrExist
	}

	parent := path.Dir(name)
	dir := path.Base(name)

	parentID, err := fs.getFileID(ctx, parent, true)
	if err != nil {
		return err
	}

	if parentID == "" {
		log.Errorf("parent not found")
		return os.ErrNotExist
	}

	f := &drive.File{
		MimeType: mimeTypeFolder,
		Name:     dir,
		Parents:  []string{parentID},
	}

	_, err = fs.client.Files.Create(f).Context(ctx).Do()
	if err != nil {
		return err
	}

	fs.invalidatePath(name)
	fs.invalidatePath(parent)

	return nil
}

func (fs *fileSystem) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	log.Debugf("openFile %v %v %v", name, flag, perm)
	name = normalizePath(name)

	if flag&os.O_RDWR != 0 {
		if flag != os.O_RDWR|os.O_CREATE|os.O_TRUNC {
			log.Panic("not implemented: fileSystem.OpenFile")
		}

		return &openWritableFile{fs: fs, name: name}, nil
	}

	if flag == os.O_RDONLY {
		file, err := fs.getFile(ctx, name, false)
		if err != nil {
			return nil, err
		}

		return &openReadonlyFile{fs: fs, file: file}, nil
	}

	return nil, fmt.Errorf("unsupported open mode: %v", flag)
}

func (fs *fileSystem) RemoveAll(ctx context.Context, name string) error {
	log.Debugf("removeAll %v", name)

	name = normalizePath(name)
	id, err := fs.getFileID(ctx, name, false)
	if err != nil {
		return err
	}

	err = fs.client.Files.Delete(id).Do()
	if err != nil {
		log.Errorf("can't delete file %v", err)
		return err
	}

	fs.invalidatePath(name)
	fs.invalidatePath(path.Dir(name))
	return nil
}

func (fs *fileSystem) Rename(ctx context.Context, oldName, newName string) error {
	f, err := fs.getFile(ctx, oldName, false)
	if err != nil {
		return err
	}

	f.Name = newName
	_, err = fs.client.Files.Update(f.Id, f).Do()
	if err != nil {
		return err
	}

	return nil
}

func (fs *fileSystem) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	log.Debugf("stat %v", name)

	f, err := fs.getFile(ctx, name, false)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	if f == nil {
		log.Debugf("can't find file %v", name)
		return nil, os.ErrNotExist
	}

	return newFileInfo(f), nil
}

func (fs *fileSystem) List(parent *drive.File, count int) ([]*drive.File, error) {
	q := fs.client.Files.List()
	q.Q(fmt.Sprintf("'%s' in parents", parent.Id))
	if count != 0 {
		q.PageSize(int64(count))
	}

	log.Tracef("query: %v", q)

	r, err := q.Do()
	if err != nil {
		log.Error(err)
		return nil, err
	}

	var files []*drive.File
	for _, file := range r.Files {
		if ignoreFile(file) {
			continue
		}

		files = append(files, file)
	}

	return files, nil
}

func (fs *fileSystem) getFileID(ctx context.Context, p string, onlyFolder bool) (string, error) {
	f, err := fs.getFile(ctx, p, onlyFolder)
	if err != nil {
		return "", err
	}

	return f.Id, nil
}

func (fs *fileSystem) getFile0(ctx context.Context, p string, onlyFolder bool) (*drive.File, error) {
	log.Tracef("getFile0 %v %v", p, onlyFolder)
	p = normalizePath(p)

	if p == "" {
		f, err := fs.client.Files.Get("root").Context(ctx).Do()
		if err != nil {
			log.Error(err)
			return nil, err
		}

		return f, nil
	}

	parent := path.Dir(p)
	base := path.Base(p)

	parentID, err := fs.getFileID(ctx, parent, true)
	if err != nil {
		log.Errorf("can't locate parent %v error: %v", parent, err)
		return nil, err
	}

	q := fs.client.Files.List()
	query := fmt.Sprintf("'%s' in parents and name='%s'", parentID, base)
	if onlyFolder {
		query += " and mimeType='" + mimeTypeFolder + "'"
	}

	q.Q(query)
	log.Tracef("query: %v", q)

	r, err := q.Do()
	if err != nil {
		log.Error(err)
		return nil, err
	}

	for _, file := range r.Files {
		if ignoreFile(file) {
			continue
		}

		return file, nil
	}

	return nil, os.ErrNotExist
}
