package model

import "time"

type FileInfo struct {
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
	Hash    string    `json:"hash"`
}

type Document struct {
	FileInfo FileInfo
	Metadata map[string]string
	Content  []byte
}
