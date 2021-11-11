package media

import "sync"

type Registry struct {
	mu sync.Mutex

	hashes map[uint32]*Media
	uniqueMedia []*Media
}

func NewRegistry() *Registry {
	return &Registry{
		hashes:      make(map[uint32]*Media),
	}
}

func (r *Registry) UniqueMedia() []*Media {
	return r.uniqueMedia
}

func (r *Registry) Merge(registries ...*Registry) error {
	r.mu.Lock()

	for _, reg := range registries {
		for _, m := range reg.uniqueMedia {
			if err := r.add(m); err != nil {
				r.mu.Unlock()
				return err
			}
		}
	}

	r.mu.Unlock()
	return nil
}

func (r *Registry) Add(m *Media) error {
	r.mu.Lock()

	err := r.add(m)

	r.mu.Unlock()

	return err
}

func (r *Registry) add(m *Media) error {
	m2 := r.hashes[m.hash]
	if m2 != nil {
		match, err := m.Compare(m2)
		if err != nil || match {
			return err
		}
	}

	r.hashes[m.hash] = m
	r.uniqueMedia = append(r.uniqueMedia, m)

	return nil
}
