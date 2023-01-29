package main

import (
	"flag"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
)

type Config struct {
	apiKey      string
	workspaceId string
	path        string
	filename    string
	branch      string
}

func main() {
	path := flag.String("path", "", "Путь до файла, куда записать коллекцию, валидный путь с доступом на запись")
	filename := flag.String("filename", "", "Если не передан или указанный форк не существует, будет взята родительская коллекция, аля master")
	branch := flag.String("branch", "", "Если не задано, название по умолчанию default.json")
	flag.Parse()
	config := Config{
		apiKey:      os.Getenv("POSTMAN_API_KEY"),
		workspaceId: os.Getenv("POSTMAN_WORKSPACE_ID"),
		path:        *path,
		branch:      *branch,
		filename:    *filename,
	}
	validate(config)
	fmt.Printf("my cmd: \"%v\"\n", *path)
	fmt.Printf("my cmd: \"%v\"\n", *filename)
	fmt.Printf("my cmd: \"%v\"\n", *branch)
}

func validate(config Config) {
	if config.apiKey == "" {
		fmt.Println("env POSTMAN_API_KEY required")
		os.Exit(1)
	}
	if config.workspaceId == "" {
		fmt.Println("env P~OSTMAN_WORKSPACE_ID required")
		os.Exit(1)
	}

	test := writable(config.path)
	fmt.Println("flagvar has value ", test)
}

func writable(path string) bool {
	return unix.Access(path, unix.W_OK) == nil
}
