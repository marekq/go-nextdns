package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/TylerBrock/colorjson"
	"github.com/spf13/viper"
)

var nextdnsProfile string
var nextdnsApiKey string
var count = 0

// NextDns create a struct to hold the response body
type NextDns struct {
	Data []struct {
		Timestamp time.Time `json:"timestamp"`
		Domain    string    `json:"domain"`
		Root      string    `json:"root"`
		Type      string    `json:"type"`
		Dnssec    bool      `json:"dnssec"`
		Encrypted bool      `json:"encrypted"`
		Protocol  string    `json:"protocol"`
		ClientIP  string    `json:"clientIp"`
		Client    string    `json:"client"`
		Device    struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Model   string `json:"model"`
			LocalIP string `json:"localIp"`
		} `json:"device"`
		Status  string `json:"status"`
		Reasons []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"reasons"`
	} `json:"data"`
	Meta struct {
		Pagination struct {
			Cursor string `json:"cursor"`
		} `json:"pagination"`
	} `json:"meta"`
}

// Check error, exit if error
func check(e error) {
	if e != nil {
		fmt.Println(e)
		panic(e)
	}
}

// Check input arguments
func checkInput(inputArg string) string {

	// Allowed input formats: -1h, 2022-09-01 and now
	match, _ := regexp.MatchString("-[0-9]{1,3}[a-z]|now|[0-9]{4}-[0-9]{2}-[0-9]{2}", inputArg)

	if !match {
		fmt.Println("Invalid input: " + inputArg)
		fmt.Println("Example: ./main stream, ./main download -1h now, ./main download -3d now")
		os.Exit(1)
	}

	return inputArg

}

func streamRequest(client http.Client, f *os.File, keyword string) {

	// Create URL string
	url := "https://api.nextdns.io/profiles/" + nextdnsProfile + "/logs/stream?raw=1"

	// Add optional search keyword to URL
	if keyword != "" {
		url += "&search=" + keyword
	}

	// Create HTTP GET request
	req, err := http.NewRequest("GET", url, nil)
	check(err)

	// Add API key to request header
	req.Header = http.Header{
		"X-Api-Key": []string{nextdnsApiKey},
	}

	// Perform request and check for errors
	res, err := client.Do(req)
	check(err)

	// Read response body
	reader := bufio.NewReader(res.Body)

	// Start indefinite loop through response records
	for {

		// Read line separated by newline
		line, err := reader.ReadBytes('\n')
		check(err)

		// Return only JSON responses
		match, _ := regexp.Compile("timestamp")
		if match.Match(line) {

			jsonstr := string(line[6 : len(line)-1])

			_, err := f.WriteString(jsonstr + ",\n")
			check(err)

			// Create color formatter with indent
			f := colorjson.NewFormatter()
			f.Indent = 4
			f.RawStrings = true

			// Format JSON
			var obj map[string]interface{}
			json.Unmarshal([]byte(string(line[6:])), &obj)

			// Marshall the colorized JSON
			s, err := f.Marshal(obj)
			check(err)

			// Print the colorized JSON
			fmt.Println(string(s))

		}
	}
}

// Get request to NextDNS API
func downloadRequest(client http.Client, cursor string, f *os.File, start string, end string) (string, int) {

	// Create URL string
	url := "https://api.nextdns.io/profiles/" + nextdnsProfile + "/logs?from=" + start + "&to=" + end + "&limit=1000&raw=1"

	// Add optional cursor to URL
	if cursor != "" && cursor != "empty" {
		url = url + "&cursor=" + cursor
	}

	// Create HTTP GET request
	req, err := http.NewRequest("GET", url, nil)
	check(err)

	// Add API key to request header
	req.Header = http.Header{
		"X-Api-Key": []string{nextdnsApiKey},
	}

	// Perform request and check for errors
	res, err := client.Do(req)
	check(err)

	// Decode response body into struct
	var p NextDns
	err1 := json.NewDecoder(res.Body).Decode(&p)
	check(err1)

	var maxTs = 0

	// Loop through response records
	for _, record := range p.Data {

		// Get max_ts timestamp
		if int(record.Timestamp.Unix()) > maxTs {
			maxTs = int(record.Timestamp.Unix())
		}

		// Marshal JSON
		line, err := json.Marshal(record)
		check(err)

		// Write to file
		_, err2 := f.WriteString(string(line) + ",\n")
		check(err2)

		// Increment counter
		count++
	}

	// Return cursor and max_ts
	returnToken := p.Meta.Pagination.Cursor
	return returnToken, maxTs
}

// Main function
func main() {

	// Read config file from .env
	viper.SetConfigFile(".env")
	viper.ReadInConfig()

	// Set API key and profile
	nextdnsApiKey = viper.GetString("nextdns_api_key")
	nextdnsProfile = viper.GetString("nextdns_profile")

	var maxTs int
	var startDt string
	var endDt string

	// Get argument length from input
	argLen := len(os.Args[1:])

	// Create HTTP client
	client := http.Client{}

	// If stream argument without keyword given
	if argLen == 1 && os.Args[1] == "stream" {

		// Create file
		f, err := os.Create("stream-all.log")
		check(err)
		defer f.Close()

		fmt.Println("streaming logs to stream-all.log ...")
		streamRequest(client, f, "")

	} else if argLen == 2 && os.Args[1] == "stream" {

		keyword := os.Args[2]

		// Create file
		f, err := os.Create("stream-" + keyword + ".log")
		check(err)
		defer f.Close()

		// If stream and keyword argument given
		fmt.Println("streaming logs with keyword: " + keyword + " to stream-" + keyword + ".log ...")
		streamRequest(client, f, keyword)

	} else if argLen == 3 && os.Args[1] == "download" {

		// If download, check input and get start and end date
		startDt = checkInput(os.Args[2])
		endDt = checkInput(os.Args[3])

		fmt.Println("download logs - start: ", startDt, " end: ", endDt+"\n")

		// Create file
		f, err := os.Create("download-output.log")
		check(err)
		defer f.Close()

		// Iterate over NextDNS API cursors
		cursor := "empty"

		// If cursor is not empty, get next cursor
		for cursor != "" {

			// Get request
			cursor, maxTs = downloadRequest(client, cursor, f, startDt, endDt)

			// Convert max_ts to date
			date := time.Unix(int64(maxTs), 0)

			// Print progress
			fmt.Printf("%v %v \n", count, date)
		}

		// Print total number of records
		fmt.Println("\nDone with " + strconv.Itoa(count) + " records")

	} else {

		// If no arguments given, return error and quit
		fmt.Println("Error: invalid input: " + strings.Join(os.Args[1:], " "))
		fmt.Println("Example: ./main stream, ./main download -1h now, ./main download -3d now")
		os.Exit(1)

	}

}
