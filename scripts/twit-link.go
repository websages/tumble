/*************************
  This is a quick utility designed to take a uri to a tweet and render it as a twitter-looking tweet.

  Author: stahnma
  Email: stahnma@websages.com
  License: Apache 2

  Notes:
    Takes a uri like https://twitter.com/stahnma/status/452133159329476608
*************************/
package main

import "encoding/json"
import "fmt"
import "io/ioutil"
import "net/http"
import "regexp"
import "os"
import "strings"

func main() {
	if len(os.Args) <= 1 {
    // No argument passed
		os.Exit(1)
	}
	input := os.Args[1]
	matched, err := regexp.MatchString("twitter.com*", input)
	if matched == false {
    // Not a twitter uri
		os.Exit(2)
	}
	parts := strings.Split(input, "/")
	id := parts[len(parts)-1]
	var f interface{}

	uri := "https://api.twitter.com/1/statuses/oembed.json?id=" + id
	resp, err := http.Get(uri)
	if err != nil {
		fmt.Println("error:", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, &f)
	m := f.(map[string]interface{})
	html2 := m["html"].(string)
	fmt.Println(html2)
}
