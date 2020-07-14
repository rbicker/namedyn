package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	username, ok := os.LookupEnv("USERNAME")
	if !ok {
		log.Fatalf("environment variable USERNAME is undefined, aborting...")
	}
	token, ok := os.LookupEnv("TOKEN")
	if !ok {
		log.Fatalf("environment variable TOKEN is undefined, aborting...")
	}
	host, ok := os.LookupEnv("HOST")
	if !ok {
		log.Fatalf("environment variable HOST is undefined, aborting...")
	}
	domain, ok := os.LookupEnv("DOMAIN")
	if !ok {
		log.Fatalf("environment variable DOMAIN is undefined, aborting...")
	}
	for {
		run(username, token, host, domain)
		time.Sleep(10 * time.Second)
	}

}

// NameRecord represents the record type from the name.com api
// (https://www.name.com/api-docs/types/record).
type NameRecord struct {
	Id     int32  `json:"id"`
	Host   string `json:"host"`
	Type   string `json:"type"`
	Answer string `json:"answer"`
	TTL    int32  `json:"ttl"`
}

// NameListRecordsReply represents the reply while listing
// records using the name.com api.
type NameListRecordsReply struct {
	Records []NameRecord `json:"records"`
}

// findRecord searches for the host A record.
func findRecord(username, token, host, domain string) (*NameRecord, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://api.name.com/v4/domains/%s/records", domain), nil)
	if err != nil {
		return nil, fmt.Errorf("error while creating request to list dns records using name.com api: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(username, token)
	cli := &http.Client{}
	res, err := cli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error while querying list of dns records using name.com api: %s", err)
	}
	defer res.Body.Close()
	var listReply NameListRecordsReply
	err = json.NewDecoder(res.Body).Decode(&listReply)
	if err != nil {
		return nil, fmt.Errorf("could not decode the reply while listing name.com records: %s", err)
	}
	if res.StatusCode != 200 {
		b, _ := ioutil.ReadAll(res.Body)
		return nil, fmt.Errorf("unexpected status code %v while listing dns record using name.com api: %s", res.StatusCode, string(b))
	}
	// search for dns
	for _, r := range listReply.Records {
		if r.Host == host && r.Type == "A" {
			return &r, nil
		}
	}
	return nil, nil
}

// run creates or updates the dynamic record if necessary.
func run(username, token, host, domain string) {
	hostname := fmt.Sprintf("%s.%s", host, domain)
	// query current record
	r, err := findRecord(username, token, host, domain)
	if err != nil {
		log.Printf("ERROR: error while looking for existing record: %s", err)
		return
	}
	// check own public ip
	res, err := http.Get("https://api.ipify.org?format=text")
	if err != nil {
		log.Printf("ERROR: error while querying ipify api to lookup own ip: %s", err)
		return
	}
	defer res.Body.Close()
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("ERROR: error while reading response body from ipify api: %s", err)
		return
	}
	if res.StatusCode != 200 {
		log.Printf("ERROR: unexpected status code %v while looking up own ip: %s", res.StatusCode, err)
		return
	}
	ip := string(b)
	// if record does not exist
	if r == nil {
		// create record
		r := NameRecord{
			Host:   host,
			Type:   "A",
			Answer: ip,
			TTL:    300, // minimum TTL unfortunately
		}
		body, err := json.Marshal(r)
		if err != nil {
			log.Printf("ERROR: error while creating request body to add dns record using name.com api: %s", err)
			return
		}
		req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://api.name.com/v4/domains/%s/records", domain), bytes.NewBuffer(body))
		if err != nil {
			log.Printf("ERROR: error while creating request to add dns record using name.com api: %s", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.SetBasicAuth(username, token)
		cli := &http.Client{}
		res, err := cli.Do(req)
		if err != nil {
			log.Printf("ERROR: error while creating dns record using name.com api: %s", err)
			return
		}
		if res.StatusCode != 200 {
			b, _ := ioutil.ReadAll(res.Body)
			log.Printf("ERROR: unexpected status code %v while creating dns record using name api: %s", res.StatusCode, string(b))
			return
		}
		log.Printf("INFO: created host A record %s with ip %s", hostname, ip)
		return
	}
	// record exists
	if r.Answer != ip {
		oldIp := r.Answer
		// ip has changed and needs to be updated
		r.Answer = ip
		body, err := json.Marshal(r)
		if err != nil {
			log.Printf("ERROR: error while creating request body to update dns record using name api: %s", err)
		}
		req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("https://api.name.com/v4/domains/%s/records/%v", domain, r.Id), bytes.NewBuffer(body))
		if err != nil {
			log.Printf("ERROR: error while creating request to update dns record using name api: %s", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.SetBasicAuth(username, token)
		cli := &http.Client{}
		res, err := cli.Do(req)
		if err != nil {
			log.Printf("ERROR: error while updating dns record using name api: %s", err)
			return
		}
		if res.StatusCode != 200 {
			b, _ := ioutil.ReadAll(res.Body)
			log.Printf("ERROR: unexpected status code %v while updating dns record using name api: %s", res.StatusCode, string(b))
			return
		}
		log.Printf("INFO: updated host A record %s, changed ip from %s to %s", hostname, oldIp, ip)
		return
	}

}
