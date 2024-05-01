package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"cloud.google.com/go/storage"
)

func main() {

	env, cluster, service, export, err := getFlags()
	if err != nil {
		log.Fatal(err)
	}

	if err := loadEnv(env); err != nil {
		log.Fatal(err)
	}

	auth, err := getAuth()
	if err != nil {
		log.Fatal(err)
	}

	client, err := getStorageClient()
	if err != nil {
		log.Fatal(err)
	}

	if err := run(cluster, service, auth, export, client); err != nil {
		log.Fatal(err)
	}
}

func run(cluster, service, auth, export string, client *storage.Client) error {
	defer client.Close()

	done := make(chan bool)

	if export == "none" {
		go printDots(service, done)
	}

	var volData []volumeData
	var bucketName string
	var err error

	switch service {
	case "backup":
		bucketName, volData, err = getBackupSize(auth, cluster, client)
		if err != nil {
			return err
		}
	case "tiering":
		bucketName, volData, err = getTieringSize(auth, cluster, client)
		if err != nil {
			return err
		}
	}

	switch export {
	case "local":
		fileName, err := createCSV(cluster, service, volData)
		if err != nil {
			return err
		}
		fmt.Print(fileName)
	case "cloud":
		fileName, err := createCSV(cluster, service, volData)
		if err != nil {
			return err
		}
		if err := uploadCSV(fileName, bucketName, client); err != nil {
			return err
		}
		fmt.Print(fileName)
	default:
		done <- true
		formatOutput(service, volData)
	}

	return nil
}

func getFlags() (string, string, string, string, error) {

	cluster := flag.String("cluster", "", "Enter cluster hostname or ip.")
	service := flag.String("service", "backup", "Enter 'backup' or 'tiering' to specify the service.")
	export := flag.String("export", "none", "Export a CSV file. Enter 'local' or 'cloud'.")
	env := flag.String("env", ".env", "Location of .env file")
	flag.Parse()

	if *cluster == "" {
		return "", "", "", "", fmt.Errorf("enter cluster hostname or ip")
	}
	*service = strings.ToLower(*service)
	if *service != "backup" && *service != "tiering" {
		return "", "", "", "", fmt.Errorf("enter 'backup' or 'tiering' to specify the service")
	}
	*export = strings.ToLower(*export)
	if *export != "none" && *export != "local" && *export != "cloud" {
		return "", "", "", "", fmt.Errorf("enter 'local' or 'cloud' to export a CSV file")
	}
	*env = strings.ToLower(*env)

	return *env, *cluster, *service, *export, nil
}
