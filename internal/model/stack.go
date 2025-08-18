package model

type Properties struct {
	Kind Kind
	Stem string
}

type DocumentStack struct {
	Documents   []Document
	Properties  Properties
	ChangedPath string
}

func (s *DocumentStack) Push(d Document) {
	s.Documents = append(s.Documents, d)
}
