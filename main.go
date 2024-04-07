package main

import (
	"context"
	"crypto/tls"
	"encoding/base64"
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

	cluster, service, err := getFlags()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	creds := getCreds()

	var volData []volume

	switch service {
	case "backup":
		volData, err = getBackupSize(creds, cluster)
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println()
	for _, v := range volData {
		fmt.Printf("%v\t%v\n", v.Size, v.Name)
	}
	fmt.Println()
}

func getCreds() string {

	user, ok := os.LookupEnv("netapp_user")
	if !ok {
		fmt.Println("credentials missing from environment variable 'netapp_user'")
		os.Exit(1)
	}
	pass, ok := os.LookupEnv("netapp_pass")
	if !ok {
		fmt.Println("credentials missing from environment variable 'netapp_pass'")
		os.Exit(1)
	}

	return base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
}

func getRelationships(creds, cluster string) (string, string, relationships, error) {

	url := "https://" + cluster + "/api/snapmirror/relationships/"

	var rel relationships

	resp, err := clientGET(creds, url)
	if err != nil {
		return "", "", rel, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", "", rel, err
	}

	var container string
	for _, v := range rel.Records {
		if strings.HasPrefix(v.Destination.Path, "netapp-backup") {
			path := strings.Split(v.Destination.Path, ":")
			container = path[0]
			break
		}
	}
	return container, url, rel, nil
}

func getRelationship(creds, url, uuid string) (relationship, error) {

	var r relationship

	resp, err := clientGET(creds, url+uuid)
	if err != nil {
		return r, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return r, err
	}
	return r, nil
}

func clientGET(creds, url string) (*http.Response, error) {

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	client := &http.Client{
		Timeout:   time.Second * 10,
		Transport: transport,
	}

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Basic "+creds)
	resp, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	return resp, nil
}

func mapVolToBackup(creds, container, url string, rel relationships) ([]volume, error) {
	var volData []volume
	for _, v := range rel.Records {
		if strings.HasPrefix(v.Destination.Path, container) {
			r, err := getRelationship(creds, url, v.UUID)
			if err != nil {
				return nil, err
			}
			if r.UUID == v.UUID {
				source := strings.Split(r.Source.Path, ":")
				size, err := getStorageSize(container, r.Destination.UUID)
				if err != nil {
					return nil, err
				}
				volData = append(volData, volume{
					Name: source[1],
					UUID: r.Destination.UUID,
					Size: size,
				})
			}
		}
	}

	return volData, nil
}

func getStorageSize(container, uuid string) (string, error) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	url := "gs://test-" + container + "/" + uuid

	args := []string{"storage", "du", url, "--summarize"}

	cmd := exec.CommandContext(ctx, "gcloud", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	result := strings.Split(string(output), " ")
	size, err := strconv.ParseFloat(result[0], 64)
	if err != nil {
		return "", err
	}
	return prettyByteSize(size), nil
}

func prettyByteSize(bf float64) string {
	// [Golang] Convert size in bytes to Bytes, Kilobytes, Megabytes, GB and TB
	// https://gist.github.com/anikitenko/b41206a49727b83a530142c76b1cb82d?permalink_comment_id=4467913#gistcomment-4467913

	for _, unit := range []string{"", "Ki", "Mi", "Gi", "Ti", "Pi", "Ei", "Zi"} {
		if math.Abs(bf) < 1024.0 {
			return fmt.Sprintf("%3.1f%sB", bf, unit)
		}
		bf /= 1024.0
	}
	return fmt.Sprintf("%.1fYiB", bf)
}

func getFlags() (string, string, error) {

	cluster := flag.String("cluster", "", "enter cluster hostname or ip")
	service := flag.String("service", "backup", "enter 'backup' or 'tiering' to retrieve cloud storage utilization for the service")
	flag.Parse()

	if *cluster == "" {
		return "", "", fmt.Errorf("enter cluster hostname or ip")
	}
	if *service == "" {
		return "", "", fmt.Errorf("enter 'backup' or 'tiering' to retrieve cloud storage utilization for the service")
	}

	if *service != "backup" && *service != "tiering" {
		return "", "", fmt.Errorf("enter 'backup' or 'tiering' to retrieve cloud storage utilization for the service")
	}

	return *cluster, *service, nil
}

func getBackupSize(creds, cluster string) ([]volume, error) {

	container, url, rel, err := getRelationships(creds, cluster)
	if err != nil {
		return nil, err
	}

	v, err := mapVolToBackup(creds, container, url, rel)
	if err != nil {
		return nil, err
	}

	return v, nil
}
