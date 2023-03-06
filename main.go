package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const EnvVarApiKey = "POSTMAN_API_KEY"
const EnvVarWorkspaceID = "POSTMAN_WORKSPACE_ID"

type Config struct {
	apiKey      string
	workspaceId string
	path        string
	filename    string
	branch      string
}

func (config *Config) validate() {
	var errorMessages []string
	path := config.path

	if config.apiKey == "" {
		errorMessages = append(errorMessages, "env POSTMAN_API_KEY required")
	}
	if config.workspaceId == "" {
		errorMessages = append(errorMessages, "env P~OSTMAN_WORKSPACE_ID required")
	}
	if !writable(path) {
		errorMessages = append(errorMessages, "path - должен быть валидным путем в UNIX и открытым на запись")
	}

	if len(errorMessages) == 0 {
		return
	}

	for _, message := range errorMessages {
		fmt.Println(message)
	}

	os.Exit(1)
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
	Collections []Collection
}

type Collection struct {
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

type postmanClient struct {
	httpClient  *http.Client
	basePath    string
	apiKey      string
	workspaceId string
}

func (client *postmanClient) newRequest(method string, urlPath string, body io.Reader) (*http.Request, error) {
	url := filepath.Join(client.basePath, urlPath)
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-API-Key", client.apiKey)
	return req, nil
}

func (client *postmanClient) getCollectionId(forkName string) (string, error) {
	collections := client.getCollections()
	var collectionUid string
	var collectionStagingUid string

	for _, collection := range collections {
		if collection.Fork.Label == forkName {
			collectionUid = collection.Uid
			break
		}
		if collection.Fork.Label == "staging" {
			collectionStagingUid = collection.Uid
		}
	}
	switch {
	case collectionUid != "":
		return collectionUid, nil
	case collectionStagingUid != "":
		return collectionStagingUid, nil
	default:
		return "", fmt.Errorf("collection not found")
	}
}

func (client *postmanClient) getCollections() []Collection {
	url := "https://api.getpostman.com/collections"
	queryParams := []QueryParams{
		{Key: "workspace", Value: client.workspaceId},
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

func (client *postmanClient) getCollectionData(forkName string) ([]byte, error) {
	collectionId, err := client.getCollectionId(forkName)
	if err != nil {
		return nil, err
	}

	url := "https://api.getpostman.com/collections/" + collectionId
	requestData := RequestData{}
	return getResponse(url, "GET", requestData), nil
}

func main() {
	path := flag.String("path", "", "Путь до файла, куда записать коллекцию, валидный путь с доступом на запись")
	filename := flag.String("filename", "", "Если не передан или указанный форк не существует, будет взята родительская коллекция, аля master")
	branch := flag.String("branch", "", "Если не задано, название по умолчанию default.json")
	flag.Parse()
	config := Config{
		apiKey:      os.Getenv(EnvVarApiKey),
		workspaceId: os.Getenv(EnvVarWorkspaceID),
		path:        *path,
		branch:      *branch,
		filename:    *filename,
	}
	config.validate()
	err := writeJson(config)
	if err != nil {
		fmt.Println(err)
	}
}

func writeJson(config Config) error {
	data, err := getCollectionData(config)
	if err != nil {
		return err
	}
	fullPath := filepath.Join(config.path, config.filename)
	err = writeCollection(data, fullPath)
	return err
}

func writeCollection(collection []byte, fullPath string) error {
	err := os.WriteFile(fullPath, collection, 0644)

	if err != nil {
		return fmt.Errorf("is not write collection: %w", err)
	}

	return err
}

func getCollectionData(config Config) ([]byte, error) {
	collectionId, err := getCollectionId(config.workspaceId, config.branch)
	if err != nil {
		return nil, err
	}

	url := "https://api.getpostman.com/collections/" + collectionId
	requestData := RequestData{}
	return getResponse(url, "GET", requestData), nil
}

func writable(path string) bool {
	return unix.Access(path, unix.W_OK) == nil
}

func getCollectionId(workspaceId string, forkName string) (string, error) {
	collections := getCollections(workspaceId)
	var collectionUid string
	var collectionStagingUid string

	for _, collection := range collections {
		if collection.Fork.Label == forkName {
			collectionUid = collection.Uid
			break
		}
		if collection.Fork.Label == "staging" {
			collectionStagingUid = collection.Uid
		}
	}
	switch {
	case collectionUid != "":
		return collectionUid, nil
	case collectionStagingUid != "":
		return collectionStagingUid, nil
	default:
		return "", fmt.Errorf("collection not found")
	}
}

func getCollections(workspaceId string) []Collection {
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

	client := http.DefaultClient
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
