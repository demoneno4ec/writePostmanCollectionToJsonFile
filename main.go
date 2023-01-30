package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	os "os"
	"time"
)

const EnvVarApiKey = "POSTMAN_API_KEY"

type Config struct {
	apiKey      string
	workspaceId string
	path        string
	filename    string
	branch      string
}

type QueryParams struct {
	Key   string
	Value string
}

type RequestData struct {
	Body        io.Reader
	QueryParams []QueryParams
}

type Collections struct {
	Collections []CollectionFromCollections
}

type CollectionFromCollections struct {
	Id        string    `json:"id"`
	Name      string    `json:"name"`
	Owner     string    `json:"owner"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Uid       string    `json:"uid"`
	Fork      struct {
		Label     string    `json:"label"`
		CreatedAt time.Time `json:"createdAt"`
		From      string    `json:"from"`
	} `json:"fork"`
	IsPublic bool `json:"isPublic"`
}

func main() {
	path := flag.String("path", "", "Путь до файла, куда записать коллекцию, валидный путь с доступом на запись")
	filename := flag.String("filename", "", "Если не передан или указанный форк не существует, будет взята родительская коллекция, аля master")
	branch := flag.String("branch", "", "Если не задано, название по умолчанию default.json")
	flag.Parse()
	config := Config{
		apiKey:      os.Getenv(EnvVarApiKey),
		workspaceId: os.Getenv("POSTMAN_WORKSPACE_ID"),
		path:        *path,
		branch:      *branch,
		filename:    *filename,
	}
	validate(config)
	writeJson(config)
}

func writeJson(config Config) {
	collectionId := getCollectionId(config.workspaceId, config.branch)

	fullPath := config.path + config.filename
	writeCollectionById(collectionId, fullPath)
}

func writeCollectionById(collectionId string, fullPath string) {
	url := "https://api.getpostman.com/collections/" + collectionId
	requestData := RequestData{}
	data := getResponse(url, "GET", requestData)
	err := os.WriteFile(fullPath, data, fs.FileMode(os.O_CREATE|os.O_RDWR))
	if err != nil {
		log.Fatal(err)
	}
}

func validate(config Config) {
	var errorMessages []string
	path := config.path

	if config.apiKey == "" {
		errorMessages = append(errorMessages, "env POSTMAN_API_KEY required")
	}
	if config.workspaceId == "" {
		errorMessages = append(errorMessages, "env P~OSTMAN_WORKSPACE_ID required")
	}
	if writable(path) == false {
		errorMessages = append(errorMessages, "path - должен быть валидным путем в UNIX и открытым на запись")
	}
	if path[len(path)-1:] != "/" {
		errorMessages = append(errorMessages, "path - должен заканчиваться на /")
	}

	if len(errorMessages) == 0 {
		return
	}

	for _, message := range errorMessages {
		fmt.Println(message)
	}

	os.Exit(1)
}

func writable(path string) bool {
	return unix.Access(path, unix.W_OK) == nil
}

func getCollectionId(workspaceId string, forkName string) string {
	collections := getCollections(workspaceId)
	var collectionUid string
	var collectionStagingUid string

	for _, collection := range collections {
		if collection.Fork.Label == "staging" {
			collectionStagingUid = collection.Uid
		}
		if collection.Fork.Label == forkName {
			collectionUid = collection.Uid
		}
	}

	if collectionUid == "" && collectionStagingUid == "" {
		panic(errors.New("collection not found"))
	}
	if collectionUid == "" {
		return collectionStagingUid
	} else {
		return collectionUid
	}
}

func getCollections(workspaceId string) []CollectionFromCollections {
	url := "https://api.getpostman.com/collections"
	queryParams := []QueryParams{
		{Key: "workspace", Value: workspaceId},
	}
	requestData := RequestData{
		QueryParams: queryParams,
	}
	data := getResponse(url, "GET", requestData)
	collections := &Collections{}
	err := json.Unmarshal(data, collections)

	if err != nil {
		panic(err)
	}

	return collections.Collections
}

func getResponse(url string, method string, requestData RequestData) []byte {
	apiKey := os.Getenv(EnvVarApiKey)
	headerKey := "X-API-Key"

	client := &http.Client{}
	body := requestData.Body
	queryParams := requestData.QueryParams
	req, _ := http.NewRequest(method, url, body)
	req.Header.Set(headerKey, apiKey)

	q := req.URL.Query()
	if queryParams != nil {
		for _, queryParam := range queryParams {
			q.Add(queryParam.Key, queryParam.Value)
		}
	}
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)

	if err != nil {
		panic(err)
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(err)
		}
	}(resp.Body)

	response, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		panic(err)
	}

	return response
}
