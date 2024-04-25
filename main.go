package main

import (
	"crypto/tls"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/storage"
)

type volumeData struct {
	Name   string
	UUID   string
	Size   string
	Server string
	Bucket string
}

func main() {

	cluster, service, export, err := getFlags()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	creds, err := getCredentials()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	client, err := getStorageClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := run(cluster, service, creds, export, client); err != nil {
		log.Fatal(err)
	}
}

func run(cluster, service, creds, export string, client *storage.Client) error {

	done := make(chan bool)

	if export == "none" {
		go printDots(service, done)
	}

	var volData []volumeData
	var err error

	switch service {
	case "backup":
		volData, err = getBackupSize(creds, cluster, client)
		if err != nil {
			return err
		}
	case "tiering":
		volData, err = getTieringSize(creds, cluster, client)
		if err != nil {
			return err
		}
	}

	switch export {
	case "local":
		if err := exportCSVFile(service, volData); err != nil {
			return err
		}
	case "cloud":

	default:
		done <- true
		formatOutput(service, volData)
	}

	return nil
}

func getCredentials() (string, error) {

	user, ok := os.LookupEnv("netapp_user")
	if !ok {
		return "", fmt.Errorf("missing environment variable for 'netapp_user'")
	}
	pass, ok := os.LookupEnv("netapp_pass")
	if !ok {
		return "", fmt.Errorf("missing environment variable for 'netapp_pass'")
	}

	return base64.StdEncoding.EncodeToString([]byte(user + ":" + pass)), nil
}

func getHTTPClient(creds, url string) (*http.Response, error) {

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	client := &http.Client{
		Timeout:   time.Second * 10,
		Transport: transport,
	}

	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Basic "+creds)
	resp, err := client.Do(request)
	if err == nil {
		if resp.StatusCode == http.StatusOK {
			return resp, nil
		} else {
			return nil, fmt.Errorf(resp.Status)
		}
	}
	return nil, err
}

func getFlags() (string, string, string, error) {

	cluster := flag.String("cluster", "", "Enter cluster hostname or ip")
	service := flag.String("service", "backup", "Enter 'backup' or 'tiering' to retrieve cloud storage utilization for the service")
	export := flag.String("export", "none", "Export .csv file. Enter 'local' or 'cloud', default 'none'")
	flag.Parse()

	if *cluster == "" {
		return "", "", "", fmt.Errorf("enter cluster hostname or ip")
	}
	if *service == "" {
		return "", "", "", fmt.Errorf("enter 'backup' or 'tiering' to retrieve cloud storage utilization for the service")
	}
	if *service != "backup" && *service != "tiering" {
		return "", "", "", fmt.Errorf("enter 'backup' or 'tiering' to retrieve cloud storage utilization for the service")
	}
	if *export != "none" && *export != "local" && *export != "cloud" {
		return "", "", "", fmt.Errorf("enter 'local' or 'cloud' to export .csv file")
	}

	return *cluster, *service, *export, nil
}
