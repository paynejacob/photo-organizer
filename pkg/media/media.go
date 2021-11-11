package media

import (
	"bufio"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"
)

type Media struct {
	path string
	hash uint32
	file *os.File
}

func New(path string) (Media, error) {
	var m Media
	buf := make([]byte, 64)

	m.path = path

	_, err := m.Read(buf)
	if err != nil && err != io.EOF {
		return m, err
	}

	h := fnv.New32a()
	_, _ = h.Write(buf)
	m.hash = h.Sum32()

	return m, m.Close()
}

func (m *Media) Path() string {
	return m.path
}

func (m *Media) Ext() string {
	return filepath.Ext(m.path)
}

func (m *Media) Compare(m2 *Media) (bool, error) {
	if m.Ext() != m2.Ext() {
		return false, nil
	}

	if m.hash != m2.hash {
		return false, nil
	}

	r := bufio.NewReader(m)
	r2 := bufio.NewReader(m2)

	var b byte
	var b2 byte
	var err error
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
			_ = m.Close()
			_ = m2.Close()

			return false, nil
		}
	}

	return true, nil
}

func (m *Media) Read(p []byte) (n int, err error) {
	if m.file == nil {
		m.file, err = os.Open(m.path)
	}

	if err != nil {
		return 0, err
	}

	return m.file.Read(p)
}

func (m *Media) Close() error {
	if m.file == nil {
		return nil
	}

	err := m.file.Close()
	if err != nil {
		return err
	}

	m.file = nil

	return nil
}