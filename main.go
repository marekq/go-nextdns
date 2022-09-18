package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

var nextdnsProfile string
var nextdnsApiKey string
var count = 0

// Create a struct to hold the response body
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
		fmt.Println("Example: ./main -1h now, ./main 2022-09-01 -1h")
		os.Exit(1)
	}

	return inputArg

}

func streamRequest(client http.Client) {

	// Create a URL
	url := "https://api.nextdns.io/profiles/" + nextdnsProfile + "/logs/stream?raw=1"

	// Create HTTP GET request
	req, err1 := http.NewRequest("GET", url, nil)
	check(err1)

	// Add API key to request header
	req.Header = http.Header{
		"X-Api-Key": []string{nextdnsApiKey},
	}

	// Perform request and check for errors
	res, err2 := client.Do(req)
	check(err2)

	// Read response body
	reader := bufio.NewReader(res.Body)

	// Loop through response records
	for {
		line, _ := reader.ReadBytes('\n')

		// Return only JSON responses
		match, _ := regexp.Compile("timestamp")
		if match.Match(line) {

			log.Println(string(line))
		}
	}
}

// Get request to NextDNS API
func getRequest(client http.Client, cursor string, f *os.File, start string, end string) (string, int) {

	// Create a URL
	url := "https://api.nextdns.io/profiles/" + nextdnsProfile + "/logs?from=" + start + "&to=" + end + "&limit=1000&raw=1"

	// Add optional cursor to URL
	if cursor != "" && cursor != "empty" {
		url = url + "&cursor=" + cursor
	}

	// Create HTTP GET request
	req, err1 := http.NewRequest("GET", url, nil)
	check(err1)

	// Add API key to request header
	req.Header = http.Header{
		"X-Api-Key": []string{nextdnsApiKey},
	}

	// Perform request and check for errors
	res, err2 := client.Do(req)
	check(err2)

	// Decode response body into struct
	var p NextDns
	err3 := json.NewDecoder(res.Body).Decode(&p)
	check(err3)

	var maxTs = 0

	// Loop through response records
	for _, v := range p.Data {

		// Get max_ts timestamp
		if int(v.Timestamp.Unix()) > maxTs {
			maxTs = int(v.Timestamp.Unix())
		}

		// Write to file
		v, _ := json.Marshal(v)
		f.Write(v)
		f.WriteString(",\n")

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

	// Get start date from user
	argLen := len(os.Args[1:])

	// If stream argument given
	if argLen == 1 && os.Args[1] == "stream" {

		fmt.Println("streaming logs...")
		streamRequest(http.Client{})

	} else if argLen == 2 {

		// If 2 arguments given, check input and get start and end date
		startDt = checkInput(os.Args[1])
		endDt = checkInput(os.Args[2])
		fmt.Println("download logs - start: ", startDt, " end: ", endDt+"\n")

	} else {

		// If no arguments given, return error and quit
		fmt.Println("Error: invalid input: " + strings.Join(os.Args[1:], " "))
		fmt.Println("Example: ./main stream, ./main -1h now, ./main -3d now")
		os.Exit(1)

	}

	// Create file
	f, err := os.Create("output.log")
	check(err)
	defer f.Close()

	client := http.Client{
		Timeout: 20 * time.Second,
	}

	cursor := "empty"

	for cursor != "" {
		cursor, maxTs = getRequest(client, cursor, f, startDt, endDt)
		date := time.Unix(int64(maxTs), 0)
		fmt.Printf("%v %v \n", count, date)
	}

	fmt.Println("\nDone with " + strconv.Itoa(count) + " records")

}
