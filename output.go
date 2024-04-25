package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"
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

func printDots(service string, done chan bool) {
	fmt.Printf("\nGetting Cloud Storage size for %v", strings.Title(service))
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
