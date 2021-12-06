package main

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"photo-organizer/pkg/media"
	"runtime"
	"strings"
	"sync"
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

	var wg sync.WaitGroup
	var mu sync.Mutex
	allMedia := map[uint32][]*media.Media{}
	C := make(chan string)
	for i := 0; i < runtime.NumCPU(); i++ {
		go func() {
			mm := map[uint32][]*media.Media{}
			wg.Add(1)

			for {
				path, cont := <-C
				if !cont {
					break
				}

				var m *media.Media
				m, err = media.New(path)
				if err != nil {
					logrus.Errorf("failed to compute short hash for %s", path)
					continue
				}

				mm[m.Hash] = append(mm[m.Hash], m)
			}

			mu.Lock()
			for h, m := range mm {
				allMedia[h] = append(allMedia[h], m...)
			}
			mu.Unlock()

			wg.Done()
		}()
	}

	// gather all files
	logrus.Info("discovering files")
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
					C <- path
					return nil
				}
			}

			logrus.Warnf("skipping invalid extension: %s", path)

			return nil
		})
		if err != nil {
			logrus.Fatal(err)
		}
	}

	close(C)
	wg.Wait()

	var nPaths int
	for _, m := range allMedia {
		nPaths += len(m)
	}

	logrus.Infof("found %d files", nPaths)

	logrus.Info("calculating unique files")
	var uniqueMedia []*media.Media
	// find unique files
	for _, mm := range allMedia {
		if len(mm) == 1 {
			uniqueMedia = append(uniqueMedia, mm[0])
			continue
		}

		mm, err = media.Distinct(mm...)
		if err != nil {
			logrus.Fatal(err)
		}

		uniqueMedia = append(uniqueMedia, mm...)
	}

	logrus.Infof("found %d unique files", len(uniqueMedia))

	wg = sync.WaitGroup{}
	mC := make(chan *media.Media)
	for i := 0; i < runtime.NumCPU()*2; i++ {
		go func() {
			wg.Add(1)

			et, _ := exiftool.NewExiftool()

			for {
				m, cont := <-mC
				if !cont {
					break
				}

				ts := getMediaDate(et, m)
				if ts.IsZero() {
					logrus.Errorf("failed to get timestamp for media: %s", m.Path)
					continue
				}

				err := writeMedia(m, fmt.Sprintf("%s/%d/%d/%d%s", dest, ts.Year(), ts.Month(), m.Hash, m.Ext()))
				if err != nil {
					logrus.Errorf("failed to copy media: %s", m.Path)
					continue
				}

				logrus.Debugf("processed media: %s", m.Path)
			}

			wg.Done()
		}()
	}

	logrus.Info("writing files")
	for _, m := range uniqueMedia {
		mC <- m
	}

	close(mC)
	wg.Wait()
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
	} else if _date, ok = fileInfo.Fields["FileInodeChangeDate"]; ok && _date != nil {
		ts, _ = fileInfo.GetString("FileInodeChangeDate")
	} else {
		_, ts = filepath.Split(m.Path)
		ts = strings.Replace(ts, m.Ext(), "", 1)
		ts = strings.Split(ts, "_")[0]
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

	parts := strings.Split(m.Path, "/")
	if len(parts) >= 3 {
		t, err := time.Parse("2006/01", parts[len(parts)-3]+"/"+parts[len(parts)-2])
		if err == nil {
			return t
		}
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
		return err
	}
	defer src.Close()

	dest, err := os.Create(path)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, src)
	if err != nil {
		return err
	}

	return nil
}
