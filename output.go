package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"cloud.google.com/go/storage"
)

func formatOutput(service string, volData []volumeData) {

	w := tabwriter.NewWriter(os.Stdout, 1, 1, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintf(w, "\n\nCloud Storage Size for %v:\n", strings.Title(service))
	fmt.Fprintln(w, "\n\tSize\tVolume Name\tUUID\t")
	fmt.Fprintln(w, "\t-----\t------------\t-----\t")
	for i, v := range volData {
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\t\n", i+1, v.Size, v.Name, v.UUID)
	}
	fmt.Fprintln(w)
}

func uploadCSV(fileName, bucketName string, client *storage.Client) error {

	bucket := client.Bucket(bucketName)
	fileData, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer fileData.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*50)
	defer cancel()

	object := bucket.Object("reports/" + fileName)
	object = object.If(storage.Conditions{DoesNotExist: true})

	writer := object.NewWriter(ctx)
	if _, err := io.Copy(writer, fileData); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	return nil
}

func createCSV(cluster, service string, volData []volumeData) (string, error) {

	timeStamp := time.Now().Format("01-02-2006-150405")

	fileName := fmt.Sprintf("%v-%v-%v.csv", cluster, service, timeStamp)

	f, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, err = f.WriteString("Server,Volume Name,Size,Bucket,Folder (UUID)\n")
	if err != nil {
		return "", err
	}

	for _, v := range volData {
		data := fmt.Sprintf("%v,%v,%v,%v,%v\n", v.Server, v.Name, v.Size, v.Bucket, v.UUID)
		_, err := f.WriteString(data)
		if err != nil {
			return "", err
		}
	}

	return fileName, nil
}

func printDots(service string, done chan bool) {
	fmt.Printf("\nGetting Cloud Storage size for Cloud %v", strings.Title(service))
	for {
		select {
		case <-done:
			return
		default:
			fmt.Print(".")
			time.Sleep(time.Second * 1)
		}
	}
}
