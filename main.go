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
	"time"
)

type record struct {
	Records []struct {
		UUID        string      `json:"uuid"`
		Destination destination `json:"destination"`
	} `json:"records"`
}

type relationship struct {
	UUID        string      `json:"uuid"`
	Source      Source      `json:"source"`
	Destination destination `json:"destination"`
}

type destination struct {
	Path string `json:"path"`
	UUID string `json:"uuid"`
}

type Source struct {
	Path string `json:"path"`
}

func main() {

	cluster, bucket, err := getFlags()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	out, err := listFolder(bucket)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	for i, v := range out {
		fmt.Println(i+1, v)
	}

	creds := os.Getenv("CREDS")
	url := "https://" + cluster + "/api/snapmirror/relationships/"

	rec := getRecords(creds, url)
	for _, v := range rec.Records {
		if strings.HasPrefix(v.Destination.Path, "netapp-backup") {
			rel := getRelationship(creds, url, v.UUID)
			if rel.UUID == v.UUID {
				volume := strings.Split(rel.Source.Path, ":")
				fmt.Println(rel.Destination.UUID, volume[1])
			}
		}
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

func getFlags() (string, string, error) {
	cluster := flag.String("c", "", "enter cluster hostname or ip")
	bucket := flag.String("b", "", "enter the bucket the cluster is running in")
	flag.Parse()

	if *cluster == "" {
		return "", "", fmt.Errorf("enter a cluster name")
	}
	if *bucket == "" {
		return "", "", fmt.Errorf("enter a bucket name")
	}
	return *cluster, *bucket, nil
}

func listFolder(bucket string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gsutil", "ls", bucket)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	scanner.Split(bufio.ScanLines)

	var list []string
	for scanner.Scan() {
		list = append(list, scanner.Text())
	}
	return list, nil
}

func getRecords(creds, url string) record {
	resp := clientGET(creds, url)
	defer resp.Body.Close()

	var r record
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		log.Fatal(err)
	}
	return r
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
