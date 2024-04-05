package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type volume struct {
	Name string
	UUID string
	Size string
}

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
	cluster := flag.String("cluster", "", "enter cluster hostname or ip")
	flag.Parse()
	
	if *cluster == "" {
		fmt.Println("enter cluster hostname or ip")
		os.Exit(1)
	}
	
	creds, ok := os.LookupEnv("CREDS")
	if !ok {
		fmt.Println("credentials missing from environment variable 'CREDS'")
		os.Exit(1)
	}

	container, url, rel := getRelationships(creds, *cluster)

	var volData []volume
	for _, v := range rel.Records {
		if strings.HasPrefix(v.Destination.Path, container) {
			r := getRelationship(creds, url, v.UUID)
			if r.UUID == v.UUID {
				vol := strings.Split(r.Source.Path, ":")
				size, err := getSize(container, r.Destination.UUID)
				if err != nil {
					log.Fatal(err)
				}
				volData = append(volData, volume{
					Name: vol[1],
					UUID: r.Destination.UUID,
					Size: size,
				})
			}
		}
	}

	fmt.Println()
	for _, v := range volData {
		fmt.Println(v.Size, v.UUID, v.Name)
	}
	fmt.Println()

}


func getSize(container, uuid string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	url := "gs://test-"+container+"/"+uuid

	args := []string{"storage","du",url,"--summarize"}
	
	cmd := exec.CommandContext(ctx, "gcloud", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	result := strings.Split(string(output), " ")
	size, err := strconv.ParseFloat(result[0], 64)
	if err != nil {
		log.Fatal(err)
	}
	return prettyByteSize(size), nil
}

func getRelationships(creds, cluster string) (string, string, relationships) {

	url := "https://" + cluster + "/api/snapmirror/relationships/"

	resp := clientGET(creds, url)
	defer resp.Body.Close()

	var rel relationships
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		log.Fatal(err)
	}

	var once sync.Once
	var container string
	
	for _, v := range rel.Records {
		if strings.HasPrefix(v.Destination.Path, "netapp-backup") {
			once.Do(func(){
				path := strings.Split(v.Destination.Path, ":")
				container = path[0]
			})
		}
	}
	return container, url, rel
}

func getRelationship(creds, url, uuid string) relationship {

	resp := clientGET(creds, url + uuid)
	defer resp.Body.Close()

	var r relationship
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		log.Fatalln(err)
	}
	return r
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
	request.Header.Set("Authorization", "Basic " + creds)
	resp, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	return resp
}

// [Golang] Convert size in bytes to Bytes, Kilobytes, Megabytes, GB and TB
// https://gist.github.com/anikitenko/b41206a49727b83a530142c76b1cb82d?permalink_comment_id=4467913#gistcomment-4467913
func prettyByteSize(bf float64) string {
	for _, unit := range []string{"", "Ki", "Mi", "Gi", "Ti", "Pi", "Ei", "Zi"} {
		if math.Abs(bf) < 1024.0 {
			return fmt.Sprintf("%3.1f%sB", bf, unit)
		}
		bf /= 1024.0
	}
	return fmt.Sprintf("%.1fYiB", bf)
}