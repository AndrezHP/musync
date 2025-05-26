package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"time"
)

type JsonWrapper struct {
	content map[string]any
}

func (wrapper JsonWrapper) get(key string) JsonWrapper {
	newContent, _ := wrapper.content[key].(map[string]any)
	return JsonWrapper{newContent}
}

func (wrapper JsonWrapper) getString(key string) string {
	str, _ := wrapper.content[key].(string)
	return str
}

func (wrapper JsonWrapper) getSlice(key string) []any {
	slice, _ := wrapper.content[key].([]any)
	return slice
}

func (wrapper JsonWrapper) getNumber(key string) int {
	float, _ := wrapper.content[key].(float64)
	return int(float)
}

func doRequestWithRetry(client *http.Client, request *http.Request, printBody bool) (map[string]any, *http.Response) {
	time.Sleep(time.Duration(sleep) * time.Millisecond)
	response, err := client.Do(request)
	check(err)
	if response.StatusCode == 429 {
		sleep = math.Min(sleep+100, 4000)
		log.Println("Rate limit hit! Increasing sleep time to: ", sleep)
		time.Sleep(5 * time.Second)
		return doRequestWithRetry(client, request, printBody)
	}

	reponseBody := getBody(response)
	var result map[string]any
	err = json.Unmarshal(reponseBody, &result)

	if err != nil && response.StatusCode != 201 {
		log.Println("Error: ", err, "Response", response)
	} else if response.StatusCode == 201 {
		log.Println("Resource created on: ", request.URL)
	}

	if printBody {
		printJson(reponseBody)
	}

	return result, response
}

func printJson(body []byte) {
	var prettyJSON bytes.Buffer
	json.Indent(&prettyJSON, body, "", "  ")
	fmt.Println("Json: ", string(prettyJSON.Bytes()))
}

func check(e error) {
	if e != nil {
		log.Println("ERROR: ", e)
	}
}

func getBody(response *http.Response) []byte {
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode > 299 {
		log.Fatalf("Response failed with status code: %d and\nbody: %s\n", response.StatusCode, body)
	}
	check(err)
	return body
}

func startLogging() {
	file, err := os.OpenFile("log.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}

	multiWriter := io.MultiWriter(os.Stdout, file)
	log.SetOutput(multiWriter)
}
