package fixtures

import "github.com/bruli-lab/stowmark-console/internal/domain/repository"

type EntryBuilder struct {
	name  string
	isDir bool
	size  int64
}

func Entry() *EntryBuilder {
	return &EntryBuilder{name: "file.txt", size: 128}
}

func (b *EntryBuilder) WithName(name string) *EntryBuilder {
	b.name = name
	return b
}

func (b *EntryBuilder) WithSize(size int64) *EntryBuilder {
	b.size = size
	return b
}

func (b *EntryBuilder) AsDirectory() *EntryBuilder {
	b.isDir = true
	b.size = 0
	return b
}

func (b *EntryBuilder) Build() repository.Entry {
	return *repository.NewEntry(b.name, b.isDir, b.size)
}
