package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type JsonWrapper struct {
	content map[string]any
}

func makeJson(content any) JsonWrapper {
	return JsonWrapper{content.(map[string]any)}
}

func (wrapper JsonWrapper) get(key string) JsonWrapper {
	newContent, _ := wrapper.content[key].(map[string]any)
	return JsonWrapper{newContent}
}

func (wrapper JsonWrapper) getInt(key string) int {
	float, _ := wrapper.content[key].(float64)
	return int(float)
}

func (wrapper JsonWrapper) getString(key string) string {
	str, _ := wrapper.content[key].(string)
	return str
}

func (wrapper JsonWrapper) getSlice(key string) []any {
	slice, _ := wrapper.content[key].([]any)
	return slice
}

func (wrapper JsonWrapper) getAt(key string, index int) JsonWrapper {
	slice, _ := wrapper.content[key].([]any)[index].(map[string]any)
	return JsonWrapper{slice}
}

func (wrapper JsonWrapper) getStringAt(key string, index int) string {
	str, _ := wrapper.content[key].([]any)[index].(string)
	return str
}

func (wrapper JsonWrapper) getIntAt(key string, index int) int {
	float, _ := wrapper.content[key].([]any)[index].(float64)
	return int(float)
}

var sleep float64 = 1500

func Sleep() float64 {
	return sleep
}

func SetRequestSleep(newValue float64) {
	sleep = newValue
}

func DoRequest(client *http.Client, request *http.Request, printBody bool) (map[string]any, *http.Response) {
	time.Sleep(time.Duration(sleep) * time.Millisecond)
	response, err := client.Do(request)
	Check(err)
	if response.StatusCode == 429 {
		sleep = math.Min(sleep+100, 4000)
		log.Println("Rate limit hit! Increasing sleep time to:", sleep)
		time.Sleep(time.Duration(sleep) * time.Millisecond)
		return DoRequest(client, request, printBody)
	}
	if response.StatusCode >= 500 {
		log.Println("Server error! Status", response.StatusCode)
		time.Sleep(10 * time.Second)
		return DoRequest(client, request, printBody)
	}

	reponseBody := getBody(response)
	var result map[string]any
	err = json.Unmarshal(reponseBody, &result)

	if err != nil && response.StatusCode != 201 {
		log.Println("Error:", err, "Response", response)
	} else if response.StatusCode == 201 {
		log.Println("Resource created on:", request.URL)
	}

	if printBody {
		log.Println("Json:", formatJsonString(reponseBody))
	}

	return result, response
}

func getBody(response *http.Response) []byte {
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode > 299 {
		error := fmt.Sprint("Response failed with status code:", response.StatusCode, "and body:", body, ",response:", response.Request)
		log.Fatalf(error)
	}
	Check(err)
	return body
}

func formatJsonString(body []byte) string {
	var prettyJSON bytes.Buffer
	json.Indent(&prettyJSON, body, "", "  ")
	return string(prettyJSON.Bytes())
}

func Check(e error) {
	if e != nil {
		log.Println("ERROR:", e)
	}
}

// regex
func cleanString(input string) string {
	binder := `(\s-\s)`
	symbols := `|\[.+\]|[\(\)@#$%^&*\[\]:;,¿?/~\\|]`
	year := `|((20)\d{2})`
	words := `|(?i)(re-*master(ed)*|version|reissue|\strio\s|\squartet\s|\squintet\s)`
	regex := regexp.MustCompile(binder + symbols + year + words)
	var result = regex.ReplaceAllString(input, " ")
	return regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")
}

func cleanArtistName(input string) string {
	return strings.TrimSpace(regexp.MustCompile(`(?i)\s(and|&|y)\s`).ReplaceAllString(input, " "))
}

func cleanTitle(input string) string {
	return regexp.MustCompile(`(;.*)|(feat.*)|[\(\)\[\]]`).ReplaceAllString(input, "")
}

// Distance/matching
func relativeDistance(str1, str2 string) float64 {
	clean1 := strings.TrimSpace(strings.ToLower(cleanString(str1)))
	clean2 := strings.TrimSpace(strings.ToLower(cleanString(str2)))
	dist := float64(levenshteinDistance(clean1, clean2))
	maxLength := float64(max(len(clean1), len(clean2)))
	log.Println(str1, "->", clean1)
	log.Println(str2, "->", clean2)
	log.Println(dist, dist/maxLength)
	return dist / maxLength
}

func approximateMatch(str1, str2 string, approx float64) bool {
	approximation := relativeDistance(str1, str2)
	return approximation < approx
}

func stringMatch(str1, str2 string) bool {
	clean1 := strings.TrimSpace(strings.ToLower(cleanString(str1)))
	clean2 := strings.TrimSpace(strings.ToLower(cleanString(str2)))
	log.Println(str1, "->", clean1)
	log.Println(str2, "->", clean2)
	match := clean1 == clean2
	if !match {
		log.Println(clean1, "=", clean2, ": Did not match")
	}
	return match
}

func similarity(str1, str2 string) float64 {
	clean1 := strings.TrimSpace(strings.ToLower(cleanString(str1)))
	clean2 := strings.TrimSpace(strings.ToLower(cleanString(str2)))
	similarity := float64(jaroWinklerSimilarity(clean1, clean2))
	return similarity
}

func levenshteinDistance(str1, str2 string) int {
	strLen1 := len(str1)
	strLen2 := len(str2)

	if strLen1 == 0 {
		return strLen2
	} else if strLen2 == 0 {
		return strLen1
	} else if str1 == str2 {
		return 0
	}

	column := make([]int, strLen1+1)
	for y := 1; y <= strLen1; y++ {
		column[y] = y
	}

	for x := 1; x <= strLen2; x++ {
		column[0] = x
		lastkey := x - 1
		for y := 1; y <= strLen1; y++ {
			oldkey := column[y]
			var i int
			if str1[y-1] != str2[x-1] {
				i = 1
			}
			m := min(column[y]+1, column[y-1]+1)
			column[y] = min(m, lastkey+i)
			lastkey = oldkey
		}
	}

	return column[strLen1]
}

func jaroWinklerSimilarity(str1, str2 string) float32 {
	jaroSim := jaroSimilarity(str1, str2)
	if jaroSim != 0.0 && jaroSim != 1.0 {
		str1len := len(str1)
		str2len := len(str2)

		var prefix int
		for i := range min(str1len, str2len) {
			if str1[i] == str2[i] {
				prefix++
			} else {
				break
			}
		}

		prefix = min(prefix, 4)
		return jaroSim + 0.1*float32(prefix)*(1-jaroSim)
	}
	return jaroSim
}

func jaroSimilarity(str1, str2 string) float32 {
	str1len := len(str1)
	str2len := len(str2)
	if str1len == 0 || str2len == 0 {
		return 0.0
	} else if str1 == str2 {
		return 1.0
	}

	var match int
	maxDist := max(str1len, str2len)/2 - 1
	str1Table := make([]int, str1len)
	str2Table := make([]int, str2len)
	for i := range str1len {
		for j := max(0, i-maxDist); j < min(str2len, i+maxDist+1); j++ {
			if str1[i] == str2[j] && str2Table[j] == 0 {
				str1Table[i] = 1
				str2Table[j] = 1
				match++
				break
			}
		}
	}
	if match == 0 {
		return 0.0
	}

	var t float32
	var p int
	for i := range str1len {
		if str1Table[i] == 1 {
			for str2Table[p] == 0 {
				p++
			}
			if str1[i] != str2[p] {
				t++
			}
			p++
		}
	}
	t /= 2

	return (float32(match)/float32(str1len) +
		float32(match)/float32(str2len) +
		(float32(match)-t)/float32(match)) / 3.0
}
