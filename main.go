package main

import (
	// "context"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/gorilla/mux"
	"k8s.io/klog/v2"
	cosiapi "sigs.k8s.io/container-object-storage-interface-api/apis"
)

var containerClient *container.Client
var storageAccountClient *azblob.Client

type RequestBody struct {
	Data string `json:"data"`
}

func initContainerClient() error {
	secret, err := getSecret("/cosi/bucketcon")
	if err != nil {
		return err
	}

	klog.Infof("initContainerClient : %s", secret)

	containerClient, err = container.NewClientWithNoCredential(secret, nil)
	return err
}

func initStorageAccountClient() error {
	secret, err := getSecret("/cosi/bucketacc")
	if err != nil {
		return err
	}

	klog.Infof("initStorageAccountClient : %s", secret)
	storageAccountClient, err = azblob.NewClientWithNoCredential(secret, nil)

	return err
}

func getSecret(path string) (string, error) {
	secretPath := fmt.Sprintf("%s/BucketInfo", strings.TrimSuffix(path, "/"))
	secretFile, err := os.Open(secretPath)
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

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Add("Access-Control-Allow-Headers", "Content-Type")
	(*w).Header().Add("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
}

func main() {
	klog.Infof("AKASH :: main")
	err := initContainerClient()
	if err != nil {
		klog.Errorf("Error initializing container client : %+v", err)
		return
	}

	err = initStorageAccountClient()
	if err != nil {
		klog.Errorf("Error initializing storage account client : %+v", err)
		return
	}
	mpx := mux.NewRouter()
	mpx.Methods("OPTIONS").HandlerFunc(handlePreFlight)
	mpx.HandleFunc("/put/", putBlob)
	mpx.HandleFunc("/", handleError)
	mpx.HandleFunc("/get/{name}", getBlob)
	mpx.HandleFunc("/createcon/{name}", createContainer)
	mpx.HandleFunc("/put/{containerName}/{blobName}", putBlobInContainer)
	mpx.HandleFunc("/get/{containerName}/{blobName}", getBlobInContainer)
	err = http.ListenAndServe(":8080", mpx)
	if err != nil {
		klog.Fatal("ListenAndServe: ", err)
		return
	}
}

func handlePreFlight(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	w.WriteHeader(http.StatusOK)
	enableCors(&w)
	w.Write([]byte("Handling PreFlight\n"))
}

func handleError(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("Invalid request path\n"))
}

func createContainer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("createContainer supports only POST Requests\n"))
		return
	}

	enableCors(&w)

	vars := mux.Vars(r)
	containerName := vars["name"]

	klog.Infof("Creating container : %s", containerName)
	_, err := storageAccountClient.CreateContainer(r.Context(), containerName, nil)
	if err != nil {
		if err.Error() == "ResourceExistsError" {
			klog.Infof("Container %s already exists", containerName)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Container already exists\n"))
			return
		} else {
			klog.Infof("Container %s creation failure %+v", containerName, err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Container creation error\n"))
			return
		}
	}

	klog.Infof("Container %s created successfully", containerName)
	w.WriteHeader(http.StatusOK)
	enableCors(&w)
	w.Write([]byte("Container creation successful\n"))
}

func putBlobInContainer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("putBlobInContainer supports only POST Requests\n"))
		return
	}

	enableCors(&w)
	vars := mux.Vars(r)
	containerName := vars["containerName"]
	blobName := vars["blobName"]

	data, err := getBlobDataFromRequestBody(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error converting to json\n"))
		return
	}

	klog.Infof("putBlobInContainer :: data :: %+v :: %s :: %v", data, string(data.Data), []byte(data.Data))

	_, err = storageAccountClient.UploadBuffer(r.Context(), containerName, blobName, []byte(data.Data), nil)
	if err != nil {
		klog.Infof("Error uploading blob : %+v", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error uploading blob\n"))
		return
	}

	w.WriteHeader(http.StatusOK)
	enableCors(&w)
	w.Write([]byte("Upload successful\n"))
}

func getBlobInContainer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("getBlobInContainer supports only GET Requests\n"))
		return
	}

	enableCors(&w)

	vars := mux.Vars(r)
	containerName := vars["containerName"]
	blobName := vars["blobName"]

	data, err := storageAccountClient.DownloadStream(r.Context(), containerName, blobName, nil)
	if err != nil {
		klog.Infof("Error DownloadStream: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error fetching blob\n"))
		return
	}

	jsonData, err := getBlobDataFromResponseBody(data)
	if err != nil {
		klog.Infof("Error: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error fetching blob and generating response\n"))
		return
	}

	opStr := fmt.Sprintf("%s\n", string(jsonData))

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	enableCors(&w)
	w.Write([]byte(opStr))
}

func getBlob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("getBlob supports only GET Requests\n"))
		return
	}

	enableCors(&w)
	vars := mux.Vars(r)
	blobName := vars["name"]
	blobClient := containerClient.NewBlobClient(blobName)
	klog.Infof("GetBlob :: name :: %s", blobName)
	if blobClient == nil {
		klog.Infof("blobClient is nil")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error creating blob client\n"))
		return
	}

	data, err := blobClient.DownloadStream(r.Context(), nil)
	if err != nil {
		klog.Infof("Error DownloadStream: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error fetching blob\n"))
		return
	}

	jsonData, err := getBlobDataFromResponseBody(data)
	if err != nil {
		klog.Infof("Error: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error fetching blob and generating response\n"))
		return
	}

	opStr := fmt.Sprintf("%s\n", string(jsonData))

	w.WriteHeader(http.StatusOK)
	enableCors(&w)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(opStr))
}

func putBlob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("putBlob supports only POST Requests\n"))
		return
	}

	enableCors(&w)

	err := r.ParseMultipartForm(2 << 20)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error parsing MultiPart Form\n"))
		return
	}

	file, header, _ := r.FormFile("file")
	defer file.Close()
	buf := bytes.NewBuffer(nil)
	_, err = io.Copy(buf, file)
	if err != nil {
		klog.Infof("Error reading file : %+v", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error reading file\n"))
		return
	}

	blobClient := containerClient.NewBlockBlobClient(header.Filename)
	if blobClient == nil {
		klog.Infof("blobClient is nil")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error creating blob client\n"))
		return
	}

	_, err = blobClient.UploadBuffer(r.Context(), buf.Bytes(), nil)
	if err != nil {
		klog.Infof("Error uploading blob : %+v", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error uploading blob\n"))
		return
	}

	w.WriteHeader(http.StatusOK)
	enableCors(&w)
	w.Header().Add("Content-Type", "multipart/form-data")
	w.Write([]byte("Upload successful\n"))
}

func getBlobDataFromRequestBody(r *http.Request) (RequestBody, error) {
	var data RequestBody
	klog.Infof("put requestBody :: %+v", r.Body)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	err := dec.Decode(&data)
	if err != nil {
		klog.Infof("Error decoding json: %+v", err)
	}

	return data, err
}

func getBlobDataFromResponseBody(data azblob.DownloadStreamResponse) ([]byte, error) {
	var empty []byte
	downloadData, err := io.ReadAll(data.Body)
	if err != nil {
		klog.Infof("Error ReadAll: %s", err.Error())
		return empty, err
	}

	klog.Infof("Data from blob :: %s", string(downloadData))

	requestBody := RequestBody{
		Data: string(downloadData),
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		klog.Infof("Error in json Marshal: %s", err.Error())
		return empty, err
	}

	return jsonData, nil
}
