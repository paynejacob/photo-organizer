package main

import (
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"photo-organizer/pkg/media"
	"runtime"
)

func main() {
	if len(os.Args) <= 2 {
		log.Fatal("Usage: %s [SOURCE...] [DEST]", os.Args[0])
	}

	C := make(chan string)
	//dest := os.Args[len(os.Args) - 1]

	var wg errgroup.Group
	registries := make([]*media.Registry, runtime.NumCPU())
	for i := 0; i < runtime.NumCPU(); i++ {
		i := i
		wg.Go(func() error {
			var path string
			var cont bool
			reg := media.NewRegistry()

			for {
				path, cont = <- C
				if !cont {
					break
				}

				m, err := media.New(path)
				if err != nil {
					logrus.Errorf("failed to process file: %s\n%v", path, err)
					continue
				}

				err = reg.Add(&m)
				if err != nil {
					logrus.Errorf("failed to process file: %s\n%v", path, err)
					continue
				}
			}

			registries[i] = reg

			return nil
		})
	}

	var totalFiles int
	for _, source := range os.Args[1:len(os.Args)-1] {
		err := filepath.WalkDir(source, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			C <- path
			totalFiles += 1

			return nil
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	close(C)

	if err := wg.Wait(); err != nil {
		log.Fatal(err)
	}

	reg := media.NewRegistry()
	err := reg.Merge(registries...)
	if err != nil {
		log.Fatal(err)
	}

	var uniqueFiles int
	for _, m := range reg.UniqueMedia() {
		logrus.Info(m.Path())
		uniqueFiles += 1
	}

	logrus.Infof("total: %d unique %d", totalFiles, uniqueFiles)
}
