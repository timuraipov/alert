package filestorage

import (
	"bufio"
	"os"
)

type Storage struct {
	file   *os.File
	writer *bufio.Writer
	reader *bufio.Reader
}

func NewStorage(filename string) (*Storage, error) {
	flag := os.O_WRONLY | os.O_CREATE
	file, err := os.OpenFile(filename, flag, 0666)
	if err != nil {
		return nil, err
	}
	return &Storage{file: file, writer: bufio.NewWriter(file), reader: bufio.NewReader(file)}, nil
}
func (s *Storage) Write(data []byte) error {
	if _, err := s.writer.Write(data); err != nil {
		return err
	}
	return s.writer.Flush()
}
func (s *Storage) Read() ([]byte, error) {
	data, err := s.reader.ReadBytes('\n') // как считать весь файл?
	return data, err
}
