package gdrive

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"
	"google.golang.org/api/drive/v3"
)

type openReadonlyFile struct {
	fs            *fileSystem
	file          *drive.File
	content       []byte
	size          int64
	pos           int64
	contentReader io.Reader
}

func (f *openReadonlyFile) Write(p []byte) (int, error) {
	log.Panic("not implemented: openReadonlyFile.Write")
	return -1, nil
}

func (f *openReadonlyFile) Readdir(count int) ([]os.FileInfo, error) {
	files, err := f.fs.List(f.file, count)
	if err != nil {
		return nil, err
	}

	var fileInfos []os.FileInfo
	for _, file := range files {
		fileInfos = append(fileInfos, newFileInfo(file))
	}

	return fileInfos, nil
}

func (f *openReadonlyFile) Stat() (os.FileInfo, error) {
	return newFileInfo(f.file), nil
}

func (f *openReadonlyFile) Close() error {
	f.content = nil
	f.contentReader = nil
	return nil
}

func (f *openReadonlyFile) initContent() error {
	if f.content != nil {
		return nil
	}

	resp, err := f.fs.client.Files.Get(f.file.Id).Download()
	if err != nil {
		log.Error(err)
		return err
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return err
	}

	err = resp.Body.Close()
	if err != nil {
		log.Error(err)
		return err
	}

	f.size = int64(len(content))
	f.content = content
	f.contentReader = bytes.NewBuffer(content)
	return nil
}

func (f *openReadonlyFile) Read(p []byte) (n int, err error) {
	log.Debugf("read %d", len(p))

	err = f.initContent()
	if err != nil {
		log.Error(err)
		return 0, err
	}

	n, err = f.contentReader.Read(p)
	if err != nil {
		log.Error(err)
		return 0, err
	}

	f.pos += int64(n)
	return n, err
}

func (f *openReadonlyFile) Seek(offset int64, whence int) (int64, error) {
	log.Debugf("seek %d %d", offset, whence)

	if whence == 0 {
		// io.SeekStart
		if f.content != nil {
			f.pos = 0
			f.contentReader = bytes.NewBuffer(f.content)
			return 0, nil
		}
		return f.pos, nil
	}

	if whence == 2 {
		// io.SeekEnd
		err := f.initContent()
		if err != nil {
			return 0, err
		}
		f.contentReader = &bytes.Buffer{}
		f.pos = f.size
		return f.pos, nil
	}

	log.Panic("not implemented: openReadonlyFile.Seek")
	return 0, nil
}
