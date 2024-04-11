package main

import (
	"encoding/json"
	"strings"
)

type targets struct {
	Records []struct {
		UUID string `json:"uuid"`
		Name string `json:"name"`
	} `json:"records"`
}

type target struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	Container string `json:"container"`
	Cluster   struct {
		Name string `json:"name"`
	} `json:"cluster"`
}

type volumes struct {
	Records []volume `json:"records"`
}

type volume struct {
	Name   string `json:"volume"`
	UUID   string `json:"uuid"`
	Server string `json:"vserver"`
}

type objectStore struct {
	Records []btUUID `json:"records"`
}

type btUUID struct {
	OsName      string `json:"object_store_name"`
	BuftreeUUID string `json:"buftree_uuid"`
	VolUUID     string `json:"vol_uuid"`
}

func getTargets(creds, cluster string) (string, targets, error) {

	url := "https://" + cluster + "/api/cloud/targets/"

	var t targets

	resp, err := clientGET(creds, url)
	if err != nil {
		return "", t, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return "", t, err
	}

	return url, t, nil
}

func getTarget(creds, cluster string) (string, string, string, error) {

	url, ts, err := getTargets(creds, cluster)
	if err != nil {
		return "", "", "", err
	}

	var container, clusterName, name string
	for _, v := range ts.Records {
		if v.Name == "StorageAccount" {
			resp, err := clientGET(creds, url+v.UUID)
			if err != nil {
				return "", "", "", err
			}
			defer resp.Body.Close()

			var t target
			if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
				return "", "", "", err
			}
			name = v.Name
			container = t.Container
			clusterName = t.Cluster.Name
		}
	}
	return container, clusterName, name, nil
}

func getVolumes(creds, cluster string) (volumes, error) {
	url := "https://" + cluster + "/api/private/cli/volume/?fields=uuid,volume"

	var v volumes

	resp, err := clientGET(creds, url)
	if err != nil {
		return v, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return v, err
	}

	return v, nil
}

func getVolList(creds, cluster, clusterName string) ([]volume, error) {
	vols, err := getVolumes(creds, cluster)
	if err != nil {
		return nil, err
	}

	var svol []volume
	for _, v := range vols.Records {
		if v.Server == "svm_"+clusterName {
			if strings.HasPrefix(v.Name, "svm_") {
				continue
			}
			svol = append(svol, volume{
				Name:   v.Name,
				UUID:   v.UUID,
				Server: v.Server,
			})
		}
	}
	return svol, nil
}

func getObjectStore(creds, cluster string) (objectStore, error) {
	url := "https://" + cluster + "/api/private/cli/storage/aggregate/object-store/vol-btuuids?fields=buftree_uuid,vol_uuid"

	var o objectStore
	resp, err := clientGET(creds, url)
	if err != nil {
		return o, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&o); err != nil {
		return o, err
	}

	return o, nil
}

func getBtUUidList(creds, cluster, osName string) ([]btUUID, error) {

	os, err := getObjectStore(creds, cluster)
	if err != nil {
		return nil, err
	}

	var b []btUUID
	for _, v := range os.Records {
		if v.OsName == osName {
			b = append(b, btUUID{
				OsName:      v.OsName,
				BuftreeUUID: v.BuftreeUUID,
				VolUUID:     v.VolUUID,
			})
		}
	}
	return b, nil
}

func mapVolToTiering(container string, vols []volume, btus []btUUID) ([]volumeData, error) {
	var voldata []volumeData

	for _, v1 := range vols {
		for _, v2 := range btus {
			if v1.UUID == v2.VolUUID {
				size, err := getStorageSize(container, v2.BuftreeUUID)
				if err != nil {
					return nil, err
				}
				voldata = append(voldata, volumeData{
					Name: v1.Name,
					UUID: v2.BuftreeUUID,
					Size: size,
				})
			}
		}
	}
	return voldata, nil
}

func getTieringSize(creds, cluster string) ([]volumeData, error) {
	
	container, clusterName, osName, err := getTarget(creds, cluster)
	if err != nil {
		return nil, err
	}

	vols, err := getVolList(creds, cluster, clusterName)
	if err != nil {
		return nil, err
	}

	btus, err := getBtUUidList(creds, cluster, osName)
	if err != nil {
		return nil, err
	}

	v, err := mapVolToTiering(container, vols, btus)
	if err != nil {
		return nil, err
	}

	return v, nil
}
