package repository

type Entry struct {
	name  string
	isDir bool
	size  int64
}

func (e Entry) Name() string {
	return e.name
}

func (e Entry) IsDir() bool {
	return e.isDir
}

func (e Entry) Size() int64 {
	return e.size
}

func NewEntry(name string, isDir bool, size int64) *Entry {
	return &Entry{name: name, isDir: isDir, size: size}
}
