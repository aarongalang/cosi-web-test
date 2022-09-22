package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

func handler(w http.ResponseWriter, r *http.Request) {
	containerClient, err := azblob.NewContainerClientWithNoCredential("https://agcosiacc.blob.core.windows.net/cosi-driver-test-concreate29e4e98c-8982-482f-b4ea-e8c34476cadf?sp=r&st=2022-09-22T21:35:46Z&se=2022-09-30T05:35:46Z&sv=2021-06-08&sr=c&sig=g0szIhf2yI9WVXea%2BOdgqToXljh7t8Kwe4FOoDZIdaw%3D", nil)
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

func main() {
	http.HandleFunc("/", handler)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
		return
	}
}
