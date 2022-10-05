package main

import (
	// "context"
	"encoding/json"
	// "fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"

	"github.com/gorilla/mux"
	cosiapi "sigs.k8s.io/container-object-storage-interface-api/apis"
	"k8s.io/klog/v2"
)

var containerClient *container.Client

type RequestBody struct {
	Data string `json:"data"`
}

func initContainerClient() error {
	secret, err := getSecret()
	if err != nil {
		return err
	}

	klog.Infof("initContainerClient : %s", secret)

	containerClient, err = container.NewClientWithNoCredential(secret, nil)
	return err
}

func getSecret() (string, error) {
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
	klog.Infof("AKASH :: main")
	err := initContainerClient()
	if err != nil {
		klog.Errorf("Error initializing container client : %+v", err)
		return
	}
	mpx := mux.NewRouter()
	mpx.HandleFunc("/", handleError)
	mpx.HandleFunc("/get/{name}", getBlob)
	mpx.HandleFunc("/put/{name}", putBlob)
	err = http.ListenAndServe(":8080", mpx)
	if err != nil {
		klog.Fatal("ListenAndServe: ", err)
		return
	}
}

func handleError(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("Invalid request path"))
}


func getBlob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("getBlob supports only GET Requests"))
		return
	}
	vars := mux.Vars(r)
	blobName := vars["name"]
	blobClient := containerClient.NewBlobClient(blobName)
	klog.Infof("GetBlob :: name :: %s", blobName)
	if blobClient == nil {
		klog.Infof("blobClient is nil")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error creating blob client"))	
		return
	}

	data, err := blobClient.DownloadStream(r.Context(), nil)
	if err != nil {
		klog.Infof("Error DownloadStream: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error fetching blob"))
		return
	}

	downloadData, err := io.ReadAll(data.Body)
	if err != nil {
		klog.Infof("Error ReadAll: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error fetching blob"))
		return
	}

	klog.Infof("Data from blob :: %s", string(downloadData))

	w.WriteHeader(http.StatusOK)
	requestBody := RequestBody {
		Data: string(downloadData),
	}

	jsonData,err := json.Marshal(requestBody)
	if err != nil {
		klog.Infof("Error: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error generating response"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func putBlob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("putBlob supports only POST Requests"))
		return
	}

	vars := mux.Vars(r)
	blobName := vars["name"]
	blobClient := containerClient.NewBlockBlobClient(blobName)
	if blobClient == nil {
		klog.Infof("blobClient is nil")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error creating blob client"))
		return
	}

	var data RequestBody
	klog.Infof("put requestBody :: %+v", r.Body)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	err := dec.Decode(&data)
	if err != nil {
		klog.Infof("Error decoding json: %+v", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error converting to json"))
		return
	}

	klog.Infof("putBlob :: data :: %+v :: %s :: %v", data, string(data.Data), []byte(data.Data))

	_, err = blobClient.UploadBuffer(r.Context(), []byte(data.Data), nil)
	if err != nil {
		klog.Infof("Error uploading blob : %+v", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error uploading blob"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Upload successful"))
}