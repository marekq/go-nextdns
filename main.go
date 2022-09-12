package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

var nextdns_profile string
var nextdns_api_key string
var count = 0

// Create a struct to hold the response body
type NextDns struct {
	Data []struct {
		Timestamp time.Time     `json:"timestamp"`
		Domain    string        `json:"domain"`
		Root      string        `json:"root"`
		Tracker   string        `json:"tracker"`
		Type      string        `json:"type"`
		Dnssec    bool          `json:"dnssec"`
		Encrypted bool          `json:"encrypted"`
		Protocol  string        `json:"protocol"`
		ClientIP  string        `json:"clientIp"`
		Status    string        `json:"status"`
		Reasons   []interface{} `json:"reasons"`
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
func checkInput(input_arg string) string {

	// Allowed input formats: -1h, 2022-09-01 and now
	match, _ := regexp.MatchString("-[0-9]{1,3}[a-z]{1}|now|[0-9]{4}-[0-9]{2}-[0-9]{2}", input_arg)

	if !match {
		fmt.Println("Invalid input: " + input_arg)
		fmt.Println("Example: ./main -1h now, ./main 2022-09-01 -1h")
		os.Exit(1)
	}

	return input_arg

}

// Get request to NextDNS API
func getRequest(client http.Client, cursor string, f *os.File, start string, end string) (string, int) {

	// Create a URL
	url := "https://api.nextdns.io/profiles/" + nextdns_profile + "/logs?from=" + start + "&to=" + end + "&limit=1000&raw=1"

	// Add optional cursor to URL
	if cursor != "" && cursor != "empty" {
		url = url + "&cursor=" + cursor
	}

	// Create HTTP GET request
	req, err1 := http.NewRequest("GET", url, nil)
	check(err1)

	// Add API key to request header
	req.Header = http.Header{
		"X-Api-Key": []string{nextdns_api_key},
	}

	// Perform request and check for errors
	res, err2 := client.Do(req)
	check(err2)

	// Decode response body into struct
	var p NextDns
	err3 := json.NewDecoder(res.Body).Decode(&p)
	check(err3)

	var max_ts int = 0

	// Loop through response records
	for _, v := range p.Data {

		// Get max_ts timestamp
		if int(v.Timestamp.Unix()) > max_ts {
			max_ts = int(v.Timestamp.Unix())
		}

		// Write to file
		v, _ := json.Marshal(v)
		f.Write(v)
		f.WriteString(",\n")

		// Increment counter
		count++
	}

	// Return cursor and max_ts
	return_token := p.Meta.Pagination.Cursor
	return return_token, max_ts
}

// Main function
func main() {

	// Read config file from .env
	viper.SetConfigFile(".env")
	viper.ReadInConfig()

	// Set API key and profile
	nextdns_api_key = viper.GetString("nextdns_api_key")
	nextdns_profile = viper.GetString("nextdns_profile")

	var max_ts int
	var start_dt string
	var end_dt string

	// Get start date from user
	arg_len := len(os.Args[1:])

	// If no start date provided, error
	if arg_len != 2 {
		fmt.Println("Error: no start or end date provided")
		fmt.Println("Example: ./main -1h now, ./main -3d now")
		os.Exit(1)

	} else {

		start_dt = checkInput(os.Args[1])
		end_dt = checkInput(os.Args[2])
	}

	fmt.Println("start: ", start_dt, " end: ", end_dt)

	// Create file
	f, err := os.Create("output.log")
	check(err)
	defer f.Close()

	client := http.Client{
		Timeout: 20 * time.Second,
	}

	cursor := "empty"

	for cursor != "" {
		cursor, max_ts = getRequest(client, cursor, f, start_dt, end_dt)
		date := time.Unix(int64(max_ts), 0)
		fmt.Printf("%v %v \n", count, date)
	}

	fmt.Println("\nDone with " + strconv.Itoa(count) + " records")

}
