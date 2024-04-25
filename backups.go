package main

import (
	"encoding/json"
	"strings"

	"cloud.google.com/go/storage"
)

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

func getRelationships(creds, cluster string) (string, string, relationships, error) {

	url := "https://" + cluster + "/api/snapmirror/relationships/"

	var rel relationships

	resp, err := getHTTPClient(creds, url)
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

	resp, err := getHTTPClient(creds, url+uuid)
	if err != nil {
		return r, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return r, err
	}
	return r, nil
}

func mapVolToBackup(creds, container, url string, rel relationships, client *storage.Client) ([]volumeData, error) {

	var volData []volumeData

	for _, v := range rel.Records {
		if strings.HasPrefix(v.Destination.Path, container) {
			r, err := getRelationship(creds, url, v.UUID)
			if err != nil {
				return nil, err
			}
			if r.UUID == v.UUID {
				source := strings.Split(r.Source.Path, ":")
				size, err := getStorageSize(container, r.Destination.UUID, client)
				if err != nil {
					return nil, err
				}
				volData = append(volData, volumeData{
					Name:   source[1],
					UUID:   r.Destination.UUID,
					Size:   size,
					Server: source[0],
					Bucket: container,
				})
			}
		}
	}
	return volData, nil
}

func getBackupSize(creds, cluster string, client *storage.Client) ([]volumeData, error) {

	container, url, rel, err := getRelationships(creds, cluster)
	if err != nil {
		return nil, err
	}

	v, err := mapVolToBackup(creds, container, url, rel, client)
	if err != nil {
		return nil, err
	}

	return v, nil
}
