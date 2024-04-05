package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type relationships struct {
	Records []struct {
		UUID        string      `json:"uuid"`
		Destination destination `json:"destination"`
	} `json:"records"`
}

type relationship struct {
	UUID   string `json:"uuid"`
	Source struct {
		Path string `json:"path"`
	} `json:"source"`
	Destination destination `json:"destination"`
}

type destination struct {
	Path string `json:"path"`
	UUID string `json:"uuid"`
}

func main() {
	cluster := flag.String("c", "", "enter cluster hostname or ip")
	flag.Parse()
	
	if *cluster == "" {
		fmt.Println("enter a cluster name")
		os.Exit(1)
	}

	creds := os.Getenv("CREDS")
	url := "https://" + *cluster + "/api/snapmirror/relationships/"

	container, rec := getRelationships(creds, url)

	volData := make(map[string]string)
	for _, v := range rec.Records {
		if strings.HasPrefix(v.Destination.Path, container) {
			rel := getRelationship(creds, url, v.UUID)
			if rel.UUID == v.UUID {
				volume := strings.Split(rel.Source.Path, ":")
				volData[volume[1]] = rel.Destination.UUID
			}
		}
	}
	for k, v := range volData {
		fmt.Println(k, v)
	}

	folders, err := getFolders(container)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	for i, v := range folders {
		fmt.Println(i+1, v)
	}

}

func clientGET(creds, url string) *http.Response {

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	client := &http.Client{
		Timeout:   time.Second * 10,
		Transport: transport,
	}

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	request.Header.Set("Authorization", "Basic "+creds)
	resp, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	return resp
}


func getFolders(container string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	bucket := "gs://test-" + container

	cmd := exec.CommandContext(ctx, "gsutil", "ls", bucket)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	scanner.Split(bufio.ScanLines)

	var ss []string
	for scanner.Scan() {
		ss = append(ss, scanner.Text())
	}
	return ss, nil
}

func getRelationships(creds, url string) (string, relationships) {
	resp := clientGET(creds, url)
	defer resp.Body.Close()

	var r relationships
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		log.Fatal(err)
	}

	var once sync.Once
	var container string
	for _, v := range r.Records {
		if strings.HasPrefix(v.Destination.Path, "netapp-backup") {
			once.Do(func(){
				path := strings.Split(v.Destination.Path, ":")
				container = path[0]
			})
		}
	}
	return container, r
}

func getRelationship(creds, url, uuid string) relationship {
	url = url + uuid
	resp := clientGET(creds, url)
	defer resp.Body.Close()

	var r relationship
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		log.Fatal(err)
	}
	return r
}

