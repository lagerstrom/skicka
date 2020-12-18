package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gobuffalo/packr/v2"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	mediaDirectory = "/tmp/skicka/"
	serverPort     = 8000
)

var (
	logger *logrus.Logger
)

// respond responds with the given status and payload. The payload is marshaled to json.
func respond(w http.ResponseWriter, status int, payload interface{}) {

	var b []byte
	if payload != nil {
		var err error
		b, err = json.Marshal(payload)
		if err != nil {
			logger.Errorf("unable to encode payload to json: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal server error"))
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(b); err != nil {
		logger.Errorf("write response body: %v\n", err)
	}
}

// uploadHandler the handler responsible for file uploads
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	logger.Info("File Upload Endpoint Hit")

	// FormFile returns the first file for the given key `myFile`
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
	file, handler, err := r.FormFile("file")
	if err != nil {
		logger.WithError(err).Error("Error Retrieving the File")
		return
	}
	defer file.Close()

	logger.Infof("Uploaded File: %s", handler.Filename)
	logger.Infof("File Size: %d", handler.Size)
	fullFilePath := filepath.Join(mediaDirectory, handler.Filename)

	_, err = os.Stat(fullFilePath)
	if err == nil {
		respond(w, http.StatusSeeOther, "file already exists")
		return
	}

	uploadFile, err := os.Create(fullFilePath)
	if err != nil {
		logger.WithError(err).Error("unable to create file")
		respond(w, http.StatusInternalServerError, "unable to create file on filesystem")
		return
	}
	defer uploadFile.Close()

	_, err = io.Copy(uploadFile, file)
	if err != nil {
		logger.WithError(err).Error("unable to copy file upload to filesystem")
		respond(w, http.StatusInternalServerError, "unable to upload file")
		return
	}

	// return that we have successfully uploaded our file!
	logger.Infof("%s uploaded successfully", fullFilePath)
	respond(w, http.StatusCreated, "file upload successful")
}

// initLogger inits the logger
func initLogger() {
	logger = logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)
}

// initMediaFolder created media dir if not exists and does a couple of small checks
func initMediaFolder() error {
	dirInfo, err := os.Stat(mediaDirectory)
	if os.IsNotExist(err) {
		err = os.Mkdir(mediaDirectory, 0755)
		if err != nil {
			logger.WithError(err).Error("unable to create media dir")
		}
		return err
	}

	if err != nil {
		return err
	}

	if !dirInfo.IsDir() {
		return errors.New("media dir path is not a directory")
	}

	return nil
}

// getLocalIp returns the local IP address if found
func getLocalIp() string {
	const unknownIp = "UNKNOWN"
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return unknownIp
	}

	for _, addr := range addrs {
		if strings.HasPrefix(addr.String(), "192.168.") {
			return strings.Replace(addr.String(), "/24", "", 1)
		}
	}
	return unknownIp
}

func main() {
	initLogger()

	if err := initMediaFolder(); err != nil {
		logger.WithError(err).Error("unable to init media dir")
		return
	}

	box := packr.New("staticBox", "../html")

	muxRoutes := mux.NewRouter()
	muxRoutes.HandleFunc("/upload", uploadHandler).
		Methods("POST")

	muxRoutes.PathPrefix("/").Handler(http.FileServer(box))

	logger.Infof("IP-address: %s Port: %d", getLocalIp(), serverPort)
	logger.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", serverPort), muxRoutes))
}
