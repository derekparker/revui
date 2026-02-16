package comment

import "github.com/deparker/revui/internal/git"

// Comment represents an inline review comment on a diff.
type Comment struct {
	FilePath    string
	StartLine   int
	EndLine     int
	LineType    git.LineType
	Body        string
	CodeSnippet string
}

// Store holds comments in memory.
type Store struct {
	comments []Comment
}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) Add(c Comment) {
	s.comments = append(s.comments, c)
}

func (s *Store) Delete(filePath string, startLine int) {
	for i, c := range s.comments {
		if c.FilePath == filePath && c.StartLine == startLine {
			s.comments = append(s.comments[:i], s.comments[i+1:]...)
			return
		}
	}
}

func (s *Store) Get(filePath string, line int) *Comment {
	for i := range s.comments {
		if s.comments[i].FilePath == filePath && s.comments[i].StartLine == line {
			return &s.comments[i]
		}
	}
	return nil
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
	return s.Get(filePath, line) != nil
}
