package handlers

import (
	"Hdfs/storage"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type KeyProduct struct{}

type StorageHandler struct {
	logger *log.Logger
	// NoSQL: injecting file hdfs
	store *storage.FileStorage
	// Environment variables
	defaultFilePath    string
	defaultFileContent string
}

// Injecting the logger makes this code much more testable.
func NewStorageHandler(l *log.Logger, s *storage.FileStorage) *StorageHandler {
	// Učitavamo vrednosti iz okruženja (ako ne postoje, koristićemo default vrednosti)
	defaultFilePath := os.Getenv("DEFAULT_FILE_PATH")
	if defaultFilePath == "" {
		defaultFilePath = "/tmp" // default
	}

	defaultFileContent := os.Getenv("DEFAULT_FILE_CONTENT")
	if defaultFileContent == "" {
		defaultFileContent = "Hola Mundo!" // default
	}

	return &StorageHandler{
		logger:             l,
		store:              s,
		defaultFilePath:    defaultFilePath,
		defaultFileContent: defaultFileContent,
	}
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

	// Koristi podrazumevani sadržaj iz environment varijable
	fileContent := s.defaultFileContent

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
