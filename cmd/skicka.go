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

	"github.com/ardanlabs/conf"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/lagerstrom/skicka/static"
)

// uploadHandler object with all dependencies to handle a file upload request
type uploadHandler struct {
	mediaDir string
	logger   *logrus.Logger
}

// respond responds with the given status and payload. The payload is marshaled to json.
func respond(w http.ResponseWriter, status int, payload interface{}, logger *logrus.Logger) {

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

// handler responsible for file uploads
func (uh *uploadHandler) handler(w http.ResponseWriter, r *http.Request) {
	uh.logger.Info("File Upload Endpoint Hit")

	// FormFile returns the first file for the given key `myFile`
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
	file, handler, err := r.FormFile("file")
	if err != nil {
		uh.logger.WithError(err).Error("Error Retrieving the File")
		return
	}
	defer file.Close()

	uh.logger.Infof("Uploaded File: %s", handler.Filename)
	uh.logger.Infof("File Size: %d", handler.Size)
	fullFilePath := filepath.Join(uh.mediaDir, handler.Filename)

	_, err = os.Stat(fullFilePath)
	if err == nil {
		respond(w, http.StatusSeeOther, "file already exists", uh.logger)
		return
	}

	uploadFile, err := os.Create(fullFilePath)
	if err != nil {
		uh.logger.WithError(err).Error("unable to create file")
		respond(w, http.StatusInternalServerError, "unable to create file on filesystem", uh.logger)
		return
	}
	defer uploadFile.Close()

	_, err = io.Copy(uploadFile, file)
	if err != nil {
		uh.logger.WithError(err).Error("unable to copy file upload to filesystem")
		respond(w, http.StatusInternalServerError, "unable to upload file", uh.logger)
		return
	}

	// return that we have successfully uploaded our file!
	uh.logger.Infof("%s uploaded successfully", fullFilePath)
	respond(w, http.StatusCreated, "file upload successful", uh.logger)
}

// initLogger inits the logger
func initLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)
	return logger
}

// initMediaFolder created media dir if not exists and does a couple of small checks
func initMediaFolder(mediaDir string, logger *logrus.Logger) error {
	dirInfo, err := os.Stat(mediaDir)
	if os.IsNotExist(err) {
		err = os.Mkdir(mediaDir, 0755)
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
	logger := initLogger()

	// ====================================
	// Checks user configuration
	const (
		cfgNamespace = "SKICKA"
	)
	var cfg struct {
		MediaDir string `conf:"default:/tmp/skicka,short:m"`
		Port     int    `conf:"default:8000,short:p"`
	}
	if err := conf.Parse(os.Args[1:], cfgNamespace, &cfg); err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			usage, err := conf.Usage(cfgNamespace, &cfg)
			if err != nil {
				logger.WithError(err).Error("unable to generate config usage")
				return
			}
			fmt.Println(usage)
			return
		}
		logger.WithError(err).Error("unable to parse config")
		return
	}

	if err := initMediaFolder(cfg.MediaDir, logger); err != nil {
		logger.WithError(err).Error("unable to init media dir")
		return
	}

	uh := uploadHandler{
		mediaDir: cfg.MediaDir,
		logger: logger,
	}

	muxRoutes := mux.NewRouter()
	muxRoutes.HandleFunc("/upload", uh.handler).
		Methods("POST")
	muxRoutes.PathPrefix("/").Handler(static.StaticPageHandler())

	logger.Infof("IP-address: %s Port: %d", getLocalIp(), cfg.Port)
	logger.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), muxRoutes))
}
