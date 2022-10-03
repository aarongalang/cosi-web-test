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
	cosiapi "sigs.k8s.io/container-object-storage-interface-api/apis"
	"k8s.io/klog/v2"
)

func handler(w http.ResponseWriter, r *http.Request) {
	secret, err := getSecret("/cosi/bucket1")
	if err != nil {
		klog.Infof("secret: %s", err.Error())
		return
	}

	containerClient, err := azblob.NewContainerClientWithNoCredential(secret, nil)
	if err != nil {
		klog.Infof("containerClient: %s", err.Error())
		return
	}

	blobClient, err := containerClient.NewBlobClient("LoremIpsum.txt")
	if err != nil {
		klog.Infof("blobClient: %s", err.Error())
		return
	}

	file, err := os.Create("data.txt")
	if err != nil {
		klog.Infof("create file: %s", err.Error())
		return
	}
	defer file.Close()
	defer os.Remove("data.txt")

	err = blobClient.DownloadToFile(context.TODO(), 0, 0, file, azblob.DownloadOptions{})
	if err != nil {
		klog.Infof("download data: %s", err.Error())
		return
	}

	data, err := os.ReadFile(file.Name())
	if err != nil {
		klog.Infof("read data: %s", err.Error())
		return
	}

	fmt.Fprint(w, "Cosi test web app: "+string(data))
}

func getSecret(mntPath string) (string, error) {
	secretFile, err := os.Open("/cosi/bucket1/BucketInfo")
	if err != nil {
		return "", err
	}
	defer secretFile.Close()

	var bucketInfo cosiapi.BucketInfo
	jsonData, _ := ioutil.ReadAll(secretFile)
	json.Unmarshal(jsonData, &bucketInfo)
	klog.Infof("getSecret: %+v", bucketInfo)
	return bucketInfo.Spec.Azure.AccessToken, nil
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)
	klog.Infof("AKASH :: main")
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
		return
	}
}
