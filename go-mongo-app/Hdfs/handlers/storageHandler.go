package handlers

import (
	"Hdfs/storage"
	"io"
	"log"
	"net/http"
	"strings"
)

type KeyProduct struct{}

type StorageHandler struct {
	logger *log.Logger
	// NoSQL: injecting file hdfs
	store *storage.FileStorage
}

// Injecting the logger makes this code much more testable.
func NewStorageHandler(l *log.Logger, s *storage.FileStorage) *StorageHandler {
	return &StorageHandler{l, s}
}

func (s *StorageHandler) CopyFileToStorage(rw http.ResponseWriter, h *http.Request) {
	fileName := h.FormValue("fileName")

	err := s.store.CopyLocalFile(fileName, fileName)

	if err != nil {
		http.Error(rw, "File hdfs exception", http.StatusInternalServerError)
		s.logger.Println("File hdfs exception: ", err)
		return
	}
}

func (s *StorageHandler) WriteFileToStorage(rw http.ResponseWriter, h *http.Request) {
	fileName := h.FormValue("fileName")

	// NoSQL TODO: expand method so that it accepts file from request
	fileContent := "Hola Mundo!"

	err := s.store.WriteFile(fileContent, fileName)

	if err != nil {
		http.Error(rw, "File hdfs exception", http.StatusInternalServerError)
		s.logger.Println("File hdfs exception: ", err)
	}
}

func (s *StorageHandler) ReadFileFromStorage(rw http.ResponseWriter, h *http.Request) {
	fileName := h.FormValue("fileName")
	copied := h.FormValue("isCopied")
	isCopied := false
	if copied != "" {
		isCopied = true
	}

	fileContent, err := s.store.ReadFile(fileName, isCopied)

	if err != nil {
		http.Error(rw, "File hdfs exception", http.StatusInternalServerError)
		s.logger.Println("File hdfs exception: ", err)
		return
	}

	// Write content to response
	io.WriteString(rw, fileContent)
	s.logger.Printf("Content of file %s: %s\n", fileName, fileContent)
}

func (s *StorageHandler) WalkRoot(rw http.ResponseWriter, h *http.Request) {
	pathsArray := s.store.WalkDirectories()
	paths := strings.Join(pathsArray, "\n")
	io.WriteString(rw, paths)
}

func (s *StorageHandler) MiddlewareContentTypeSet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {
		s.logger.Println("Method [", h.Method, "] - Hit path :", h.URL.Path)

		rw.Header().Add("Content-Type", "application/json")

		next.ServeHTTP(rw, h)
	})
}
