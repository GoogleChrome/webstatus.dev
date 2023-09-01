package targz

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"path/filepath"
	"strings"
)

const (
	defaultStripComponents uint = 1
	minStripComonents      int  = 0
	maxStripComonents      int  = 1
)

type ArchiveIteartor struct {
	gzipReader      *gzip.Reader
	tarReader       *tar.Reader
	stripComponents uint
}

type ArchiveFile struct {
	data io.Reader
	name string
}

func (f ArchiveFile) GetName() string {
	return f.name
}

func (f *ArchiveFile) GetData() io.Reader {
	return f.data
}

func (i *ArchiveIteartor) Next() (*ArchiveFile, error) {
	// Currently, we only care about the files and not the directories
	filename := ""
	for {
		header, err := i.tarReader.Next()
		if err != nil {
			return nil, err
		}
		if header.Typeflag == tar.TypeReg {
			// Big assumption that the OS is linux based.
			paths := strings.Split(header.Name, "/")

			filename = filepath.Join(paths[i.stripComponents:]...)

			break
		}

	}

	return &ArchiveFile{
		data: i.tarReader,
		name: filename,
	}, nil
}

func (i *ArchiveIteartor) Close() error {
	return i.gzipReader.Close()
}

func NewTarGzArchiveIterator(reader io.ReadCloser, stripComponentsInput *int) (*ArchiveIteartor, error) {
	gzip, err := gzip.NewReader(reader)
	if err != nil {
		return nil, err
	}
	stripComponents := defaultStripComponents
	if stripComponentsInput != nil &&
		(*stripComponentsInput >= minStripComonents &&
			*stripComponentsInput <= maxStripComonents) {
		stripComponents = uint(*stripComponentsInput)
	}
	tarReader := tar.NewReader(gzip)
	return &ArchiveIteartor{
		gzipReader:      gzip,
		tarReader:       tarReader,
		stripComponents: stripComponents,
	}, nil
}
