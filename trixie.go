package main

import (
	"os"
	"syscall"
	"sync"
)

type Trixie struct{
	Trie *Trie
	path string
	mu sync.RWMutex
}

var (
	magic = []byte("Trixie :3")
)

func NewTrixie(path string) *Trixie {
	return &Trixie{
		Trie: NewTrie(),
		path: path,
	}
}

func (t *Trixie) Save() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	file, err := os.OpenFile(t.path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer file.Close()

	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX)
	if err != nil {
		return err
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)

	return t.Trie.Serialize(file)
}
