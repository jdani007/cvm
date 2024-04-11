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
	"time"

	"github.com/xuri/excelize/v2"
)

type volumeData struct {
	Name string
	UUID string
	Size string
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
		if err := exportExcelFile(service, volData); err != nil {
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
	export := flag.Bool("export", false, "export the excel file")
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

	fmt.Printf("\nVolume Size for %v:\n\n", service)
	fmt.Printf("Size\t Volume Name\n")
	fmt.Printf("------\t --------------\n")
	for _, v := range volData {
		fmt.Printf("%v\t %v\n", v.Size, v.Name)
	}
	fmt.Println()
}

func exportExcelFile(service string, volData []volumeData) error {
	f := excelize.NewFile()
	defer func() error {
		if err := f.Close(); err != nil {
			return err
		}
		return nil
	}()
	if err := f.SetSheetName("Sheet1",service); err != nil {
		return err
	}
	if err := f.SetCellValue(service, "A1", "Server"); err != nil {
		return err
	}
	if err := f.SetCellValue(service, "B1", "Volume Name"); err != nil {
		return err
	}
	if err := f.SetCellValue(service, "C1", "Size"); err != nil {
		return err
	}
	if err := f.SetCellValue(service, "D1", "UUID"); err != nil {
		return err
	}


	for i, v := range volData {
		if err := f.SetCellValue(service,"A"+fmt.Sprint(i+2),v.Server); err != nil {
			return err
		}
		if err := f.SetCellValue(service,"B"+fmt.Sprint(i+2),v.Name); err != nil {
			return err
		}
		if err := f.SetCellValue(service,"C"+fmt.Sprint(i+2),v.Size); err != nil {
			return err
		}
		if err := f.SetCellValue(service,"D"+fmt.Sprint(i+2),v.UUID); err != nil {
			return err
		}
	}

	return f.SaveAs(service+".xlsx")
}