package bitfog

// FileData represents all the common metadata for a file.
type FileData struct {
	Name  string `json:"name,omitempty"`
	Size  int64  `json:"size"`
	Mode  int32  `json:"mode"`
	Mtime int64  `json:"mtime"`
	Hash  uint64 `json:"hash,omitempty"`
	Dest  string `json:"linkdest,omitempty"`
}

// Equals reports whether a FileData object references the same file as another.
func (fd FileData) Equals(other FileData) bool {
	return fd.Size == other.Size &&
		fd.Hash == other.Hash &&
		fd.Dest == other.Dest
}
