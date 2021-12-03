package media

import (
	"bufio"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"
)

type Media struct {
	Path string
	Hash uint32
}

func (m *Media) Ext() string {
	return filepath.Ext(m.Path)
}

func New(path string) (*Media, error) {
	m := new(Media)
	buf := make([]byte, 64)

	m.Path = path

	f, err := os.Open(m.Path)
	if err != nil {
		return m, err
	}
	defer f.Close()

	_, err = f.Read(buf)
	if err != nil && err != io.EOF {
		return m, err
	}

	h := fnv.New32a()
	_, _ = h.Write(buf)
	m.Hash = h.Sum32()

	return m, nil
}

func Compare(m1, m2 *Media) (bool, error) {
	if m1.Ext() != m2.Ext() {
		return false, nil
	}

	if m1.Hash != m2.Hash {
		return false, nil
	}

	f1, err := os.Open(m1.Path)
	if err != nil {
		return false, err
	}
	defer f1.Close()

	f2, err := os.Open(m2.Path)
	if err != nil {
		return false, err
	}
	defer f2.Close()

	r := bufio.NewReader(f1)
	r2 := bufio.NewReader(f2)

	var b byte
	var b2 byte
	var err2 error
	for {
		b, err = r.ReadByte()
		b2, err2 = r2.ReadByte()

		if err == err2 && err == io.EOF {
			break
		} else if err != err2 && err == io.EOF || err2 == io.EOF {
			return false, nil
		} else if err != nil {
			return false, err
		} else if err2 != nil {
			return false, err2
		}

		if b != b2 {
			return false, nil
		}
	}

	return true, nil
}
