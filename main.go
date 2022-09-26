package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

type BucketInfo struct {
	spec BucketInfoSpec "json: spec"
}

type BucketInfoSpec struct {
	accessSecretKey string "json: accessSecretKey"
}

func handler(w http.ResponseWriter, r *http.Request) {
	secret, err := getSecret("/cosi/bucket1")
	if err != nil {
		fmt.Printf("secret: %s", err.Error())
		return
	}

	containerClient, err := azblob.NewContainerClientWithNoCredential(secret, nil)
	if err != nil {
		fmt.Printf("containerClient: %s", err.Error())
		return
	}

	blobClient, err := containerClient.NewBlobClient("LoremIpsum.txt")
	if err != nil {
		fmt.Printf("blobClient: %s", err.Error())
		return
	}

	file, err := os.Create("data.txt")
	if err != nil {
		fmt.Printf("create file: %s", err.Error())
		return
	}
	defer file.Close()
	defer os.Remove("data.txt")

	err = blobClient.DownloadToFile(context.TODO(), 0, 0, file, azblob.DownloadOptions{})
	if err != nil {
		fmt.Printf("download data: %s", err.Error())
		return
	}

	data, err := os.ReadFile(file.Name())
	if err != nil {
		fmt.Printf("read data: %s", err.Error())
		return
	}

	fmt.Fprint(w, "Cosi test web app: "+string(data))
}

func getSecret(mntPath string) (string, error) {
	secretFile, err := os.Open("/cosi/bucket1")
	if err != nil {
		return "", err
	}
	defer secretFile.Close()

	var bucketInfo BucketInfo
	jsonData, _ := ioutil.ReadAll(secretFile)
	json.Unmarshal(jsonData, &bucketInfo)
	return bucketInfo.spec.accessSecretKey, nil
}

func main() {
	http.HandleFunc("/", handler)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
		return
	}
}
