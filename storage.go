package main

import (
	"context"
	"fmt"
	"math"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

func getStorageClient() (*storage.Client, error) {
	c, err := storage.NewClient(context.Background())
	if err != nil {
		return nil, err
	}
	return c, nil
}

func getStorageSize(container, uuid string, client *storage.Client) (string, error) {

	bucket := client.Bucket(container)

	objects := bucket.Objects(context.Background(), &storage.Query{
		Prefix: uuid + "/",
	})

	var size int64
	for {
		attrs, err := objects.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return "", err
		}
		size += attrs.Size
	}
	return prettyByteSize(float64(size)), nil
}

// [Golang] Convert size in bytes to Bytes, Kilobytes, Megabytes, GB and TB
// https://gist.github.com/anikitenko/b41206a49727b83a530142c76b1cb82d?permalink_comment_id=4467913#gistcomment-4467913
func prettyByteSize(bf float64) string {

	for _, unit := range []string{"", "K", "M", "G", "T", "P", "E", "Z"} {
		if math.Abs(bf) < 1024.0 {
			return fmt.Sprintf("%3.2f %sB", bf, unit)
		}
		bf /= 1024.0
	}
	return fmt.Sprintf("%.1f YiB", bf)
}
