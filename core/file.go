package core

import (
	"gin-core/common"
	"io"
	"mime/multipart"
	"os"
	"path"
	"sync"
)

type FileStorage interface {
	Save(fileHeader *multipart.FileHeader) (string, error)
}

var _ FileStorage = &LocalFileStorage{}

type LocalFileStorage struct {
	MediaRoot string
	lock      sync.RWMutex
}

func fileExists(filePath string) (bool, error) {
	if _, err := os.Stat(filePath); err != nil {
		return os.IsNotExist(err), nil
	}

	return true, nil
}

func (l *LocalFileStorage) SetMediaRouter(mediaRoot string) {
	l.MediaRoot = mediaRoot
}

// getAlternativeName 随机命名文件
func (l *LocalFileStorage) getAlternativeName(filename string) (string, error) {
	l.lock.RLock()
	defer l.lock.Unlock()

	for {
		var dst = path.Join(l.MediaRoot, filename)
		var exist, err = fileExists(dst)
		if err != nil {
			return "", err
		}

		if !exist {
			return dst, err
		}

		filename = common.GetRandomString(7) + filename
	}
}

func (l *LocalFileStorage) Save(fileHeader *multipart.FileHeader) (string, error) {
	var src, err = fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	if len(l.MediaRoot) > 0 {
		if _, err = os.Stat(l.MediaRoot); err != nil && os.IsExist(err) {
			if err = os.MkdirAll(l.MediaRoot, 0666); err != nil {
				return "", err
			}
		}
	}

	dst, err := l.getAlternativeName(fileHeader.Filename)
	if err != nil {
		return "", err
	}

	l.lock.Lock()
	defer l.lock.Unlock()

	out, err := os.Create(dst)
	if err != nil {
		return "", err
	}

	defer out.Close()

	_, err = io.Copy(out, src)

	return dst, err
}
