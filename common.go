package bitfog

type FileData struct {
	Name  string `json:"name",omitempty`
	Size  int64  `json:"size"`
	Mode  int32  `json:"mode"`
	Mtime int64  `json:"mtime"`
	Hash  uint64 `json:"hash,omitempty"`
	Dest  string `json:"linkdest,omitempty"`
}

func (fd FileData) Equals(other FileData) bool {
	return fd.Size == other.Size &&
		fd.Hash == other.Hash &&
		fd.Dest == other.Dest
}
