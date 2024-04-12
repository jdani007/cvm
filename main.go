package main

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

type volumeData struct {
	Name   string
	UUID   string
	Size   string
	Server string
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

	var volData []volumeData

	switch service {
	case "backup":
		volData, err = getBackupSize(creds, cluster)
		if err != nil {
			log.Fatal(err)
		}
	case "tiering":
		volData, err = getTieringSize(creds, cluster)
		if err != nil {
			log.Fatal(err)
		}
	}

	if export {
		if err := exportCSVFile(service, volData); err != nil {
			log.Fatal(err)
		}
	} else {
		formatOutput(service, volData)
	}

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

func clientGET(creds, url string) (*http.Response, error) {

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
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf(resp.Status)
		} else {
			return resp, nil
		}
	}
	return nil, err
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

// [Golang] Convert size in bytes to Bytes, Kilobytes, Megabytes, GB and TB
// https://gist.github.com/anikitenko/b41206a49727b83a530142c76b1cb82d?permalink_comment_id=4467913#gistcomment-4467913
func prettyByteSize(bf float64) string {

	for _, unit := range []string{"", "K", "M", "G", "T", "P", "E", "Z"} {
		if math.Abs(bf) < 1024.0 {
			return fmt.Sprintf("%3.1f%sB", bf, unit)
		}
		bf /= 1024.0
	}
	return fmt.Sprintf("%.1fYiB", bf)
}

func getFlags() (string, string, bool, error) {

	cluster := flag.String("cluster", "", "enter cluster hostname or ip")
	service := flag.String("service", "backup", "enter 'backup' or 'tiering' to retrieve cloud storage utilization for the service")
	export := flag.Bool("export", false, "export to csv file")
	flag.Parse()

	if *cluster == "" {
		return "", "", false, fmt.Errorf("enter cluster hostname or ip")
	}
	if *service == "" {
		return "", "", false, fmt.Errorf("enter 'backup' or 'tiering' to retrieve cloud storage utilization for the service")
	}
	if *service != "backup" && *service != "tiering" {
		return "", "", false, fmt.Errorf("enter 'backup' or 'tiering' to retrieve cloud storage utilization for the service")
	}

	return *cluster, *service, *export, nil
}

func formatOutput(service string, volData []volumeData) {

	w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	defer w.Flush()

	fmt.Fprintf(w,"\nVolume Size for %v:\n", strings.Title(service))
	fmt.Fprintln(w, "\nSize\tVolume Name\tUUID\t")
	fmt.Fprintln(w, "-----\t------------\t-----\t")
	for _, v := range volData {
		fmt.Fprintf(w, "%v\t%v\t%v\t\n", v.Size, v.Name, v.UUID)
	}
	fmt.Fprintln(w)
}

func exportCSVFile(service string, volData []volumeData) error {

	timeStamp := time.Now().Format("01-02-2006-150405")

	fileName := service + "-" + timeStamp + ".csv"

	f, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString("Server,Volume Name,Size,UUID\n"); err != nil {
		return err
	}
	for _, v := range volData {
		if _, err := f.WriteString(v.Server + "," + v.Name + "," + v.Size + "," + v.UUID + "\n"); err != nil {
			return err
		}
	}
	if _, err := f.WriteString("\nFile generated on " + timeStamp); err != nil {
		return err
	}
	return nil
}
