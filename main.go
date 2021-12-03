package main

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"photo-organizer/pkg/media"
	"strings"
	"time"

	"github.com/barasher/go-exiftool"
	"github.com/sirupsen/logrus"
)

var allowedExtensions = []string{
	".jpg",
	".png",
	".mp4",
	".gif",
	".JPG",
	".mpeg",
	".mov",
	".MP4",
	".svg",
}

func main() {
	var dest string
	var err error

	if len(os.Args) <= 2 {
		log.Fatalf("Usage: %s [SOURCE...] [DEST]", os.Args[0])
	}

	// get destination fs
	dest = os.Args[len(os.Args)-1]

	// clean destination
	err = os.RemoveAll(dest)
	if err != nil {
		logrus.Fatal("failed to cleanup destination FS")
	}

	// gather all files
	var allMedia []*media.Media
	for _, source := range os.Args[1 : len(os.Args)-1] {
		err = filepath.WalkDir(source, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			for _, ext := range allowedExtensions {
				if filepath.Ext(path) == ext {
					var m *media.Media
					m, err = media.New(path)
					if err != nil {
						return err
					}

					allMedia = append(allMedia, m)

					return nil
				}
			}

			return nil
		})
		if err != nil {
			logrus.Fatal(err)
		}
	}

	// find unique media
	et, _ := exiftool.NewExiftool()
	uniqueMedia := map[uint32]*media.Media{}
	for _, m := range allMedia {
		if uniqueMedia[m.Hash] != nil {
			ok, err := media.Compare(m, uniqueMedia[m.Hash])
			if err != nil {
				logrus.Fatalf("failed to compare files: %s %s", m.Path, uniqueMedia[m.Hash].Path)
			}

			if ok {
				continue
			}
		}

		uniqueMedia[m.Hash] = m

		ts := getMediaDate(et, m)
		err = writeMedia(m, filepath.Join(dest, fmt.Sprintf("/%d/%d/%d%s", ts.Year(), ts.Month(), m.Hash, m.Ext())))
		if err != nil {
			logrus.Errorf("failed to write media for %s", m.Path)
		}
	}
}

func getMediaDate(et *exiftool.Exiftool, m *media.Media) time.Time {
	fileInfo := et.ExtractMetadata(m.Path)[0]
	if fileInfo.Err != nil {
		return time.Time{}
	}

	var ts string
	if _date, ok := fileInfo.Fields["DateTimeOriginal"]; ok && _date != nil {
		ts, _ = fileInfo.GetString("DateTimeOriginal")
	} else if _date, ok = fileInfo.Fields["MediaCreateDate"]; ok && _date != nil {
		ts, _ = fileInfo.GetString("MediaCreateDate")
	} else if _date, ok = fileInfo.Fields["FileModifyDate"]; ok && _date != nil {
		ts, _ = fileInfo.GetString("FileModifyDate")
	} else {
		_, ts = filepath.Split(m.Path)
		ts = strings.Replace(ts, m.Ext(), "", 1)
		ts = ts[:17]
	}

	var t time.Time
	var err error
	if t, err = time.Parse("2006:01:02 15:04:05-07:00", ts); err == nil {
		return t
	} else if t, err = time.Parse("2006:01:02 15:04:05", ts); err == nil {
		return t
	} else if t, err = time.Parse("06-01-02 15-04-05", ts); err == nil {
		return t
	}

	return time.Time{}
}

func writeMedia(m *media.Media, path string) error {
	dir, _ := filepath.Split(path)

	err := os.MkdirAll(dir, 0700)
	if err != nil {
		return err
	}

	src, err := os.Open(m.Path)
	if err != nil {
		logrus.Fatal(err)
	}
	defer src.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, src)
	if err != nil {
		logrus.Fatal(err)
	}

	err = os.WriteFile(path, buf.Bytes(), 0700)
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Infof("wrote %s", path)

	return nil
}
