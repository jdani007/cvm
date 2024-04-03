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
		Name string `json:"name"`
		UUID string `json:"uuid"`
	} `json:"records"`
}

type configuration struct {
	Credentials string
}

func main() {

	cluster, bucket, err := getFlags()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	out, err := listBuckets(bucket)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	for i, v := range out {
		fmt.Println(i+1, v)
	}

	// data := unmarshFile("records.json")
	data := unmarshalURL(cluster)
	for _, v := range data.Records {
		fmt.Printf("%+v\n", v)
	}
}

func clientGET(url, credentials string) *http.Response {

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	client := &http.Client{
		Timeout: time.Second * 10,
		Transport: transport,
	}

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	request.Header.Set("Authorization", "Basic "+ credentials)
	resp, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	return resp
}

func unmarshFile(file string) record {
	body, err := os.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}
	var data record
	if err = json.Unmarshal(body, &data); err != nil {
		log.Fatal(err)
	}

	return data
}


func unmarshalURL(cluster string) record {
	creds := getCreds("config.json")
	url := "https://"+cluster+"/api/storage/volumes"
	resp := clientGET(url, creds); defer resp.Body.Close()

	var data record
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Fatal(err)
	}
	return data
}

func getCreds(filename string) string {
	d, err := os.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	var config configuration
	if err = json.Unmarshal(d, &config); err != nil {
		log.Fatal(err)
	}

	return config.Credentials
}

func getFlags() (string, string, error) {
	cluster := flag.String("c", "", "enter cluster hostname or ip")
	bucket := flag.String("b", "", "enter the project the cluster is running in")
	flag.Parse()

	if *cluster == "" {
		return "", "", fmt.Errorf("enter a cluster name")
	}
	if *bucket == "" {
		return "", "", fmt.Errorf("enter a project name")
	}
	return *cluster, *bucket, nil
}

func listBuckets(bucket string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 10)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gsutil", "ls", bucket)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	scanner.Split(bufio.ScanLines)

	var list []string
	for scanner.Scan(){
		list = append(list, scanner.Text())
	}
	return list, nil
}