package comment

import "github.com/deparker/revui/internal/git"

// Comment represents an inline review comment on a diff.
type Comment struct {
	FilePath    string
	StartLine   int
	EndLine     int
	LineType    git.LineType
	Body        string
}

type commentKey struct {
	filePath  string
	startLine int
}

// Store holds comments in memory.
type Store struct {
	comments []Comment
	byKey    map[commentKey]int // maps key to index in comments slice
}

func NewStore() *Store {
	return &Store{
		byKey: make(map[commentKey]int),
	}
}

func (s *Store) Add(c Comment) {
	key := commentKey{c.FilePath, c.StartLine}
	if idx, ok := s.byKey[key]; ok {
		s.comments[idx] = c
		return
	}
	s.byKey[key] = len(s.comments)
	s.comments = append(s.comments, c)
}

func (s *Store) Delete(filePath string, startLine int) {
	key := commentKey{filePath, startLine}
	idx, ok := s.byKey[key]
	if !ok {
		return
	}
	last := len(s.comments) - 1
	if idx != last {
		s.comments[idx] = s.comments[last]
		movedKey := commentKey{s.comments[idx].FilePath, s.comments[idx].StartLine}
		s.byKey[movedKey] = idx
	}
	s.comments = s.comments[:last]
	delete(s.byKey, key)
}

func (s *Store) Get(filePath string, line int) *Comment {
	key := commentKey{filePath, line}
	idx, ok := s.byKey[key]
	if !ok {
		return nil
	}
	return &s.comments[idx]
}

func (s *Store) All() []Comment {
	return s.comments
}

func (s *Store) ForFile(filePath string) []Comment {
	var result []Comment
	for _, c := range s.comments {
		if c.FilePath == filePath {
			result = append(result, c)
		}
	}
	return result
}

func (s *Store) HasComment(filePath string, line int) bool {
	_, ok := s.byKey[commentKey{filePath, line}]
	return ok
}
