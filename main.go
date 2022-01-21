package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	filepath "path"
	"strings"
	"text/template"
)

const maxSize = 5e6

func writeImage(r *http.Request) error {
	r.Body = http.MaxBytesReader(nil, r.Body, maxSize)
	src, _, err := r.FormFile("image")
	if err != nil {
		return err
	}
	defer src.Close()
	b, err := io.ReadAll(src)
	if err != nil {
		return err
	}
	ext := strings.TrimPrefix(http.DetectContentType(b), "image/")
	if ext != "jpeg" && ext != "png" {
		return errors.New("invalid file type, expected JPEG or PNG")
	}
	hash := sha256.Sum256(b)
	filename := filepath.Join("images", hex.EncodeToString(hash[:])+"."+ext)
	return os.WriteFile(filename, b, 0666)
}

func writeIndex(rw http.ResponseWriter) error {
	files, err := os.ReadDir("images")
	if err != nil {
		return err
	}
	var images []string
	for _, file := range files {
		if file.Type().IsRegular() {
			switch filepath.Ext(file.Name()) {
			case ".jpeg", ".png":
				images = append(images, file.Name())
			}
		}
	}
	tmpl, err := template.ParseFiles("index.html.tmpl")
	if err != nil {
		return err
	}
	rw.Header().Add("content-type", "text/html")
	return tmpl.Execute(rw, images)
}

func handleIndex(rw http.ResponseWriter, _ *http.Request) {
	if err := writeIndex(rw); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		log.Println(err)
	}
}

func handleUpload(rw http.ResponseWriter, r *http.Request) {
	if err := writeImage(r); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		log.Println(err)
	} else {
		rw.Header().Set("Location", "..")
		rw.WriteHeader(http.StatusFound)
	}
}

func handleImage(rw http.ResponseWriter, r *http.Request) {
	switch filepath.Ext(r.URL.Path) {
	case ".jpeg", ".png":
		http.ServeFile(rw, r, "."+r.URL.Path)
	default:
		http.NotFound(rw, r)
	}
}

func main() {
	os.Mkdir("images", 0755)
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/upload/", handleUpload)
	http.HandleFunc("/images/", handleImage)
	port := flag.String("port", "8080", "Specify alternate port")
	flag.Parse()
	fmt.Println("listening to", *port)
	if err := http.ListenAndServe(":"+*port, nil); err != http.ErrServerClosed {
		panic(err)
	}
}
