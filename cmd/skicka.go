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

// getFileName checks if a file exists with the same name. If it does it will give the file a new name
// that new name will add (n) to the end of the file before the extension.
func getFileName(fp string) (string, error) {
	originalFp := fp
	var i int
	for {
		if _, err := os.Stat(fp); err != nil {
			return fp, nil
		}
		fp = originalFp
		extension := filepath.Ext(fp)
		noExFp := fp[:len(fp)-len(extension)]
		fp = fmt.Sprintf("%s(%d)%s", noExFp, i, extension)
		i += 1
	}
}

// handler responsible for file uploads
func (uh *uploadHandler) handler(w http.ResponseWriter, r *http.Request) {
	uh.logger.Debug("File Upload Endpoint Hit")

	// FormFile returns the first file for the given key `myFile`
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
	file, handler, err := r.FormFile("file")
	if err != nil {
		uh.logger.WithError(err).Error("Error Retrieving the File")
		return
	}
	defer file.Close()

	uh.logger.
		WithField("file_size", fmt.Sprintf("%d bytes", handler.Size)).
		WithField("file_name", handler.Filename).
		Info("file is about to be uploaded")
	fullFilePath := filepath.Join(uh.mediaDir, handler.Filename)

	fullFilePath, err = getFileName(fullFilePath)
	if err != nil {
		uh.logger.WithError(err).Error("unable to generate full file path")
		respond(w, http.StatusInternalServerError, "unable to generate file path", uh.logger)
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
	uh.logger.WithField("file_name", fullFilePath).Info("file successfully uploaded")
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
		Debug    bool   `conf:"default:false"`
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

	if cfg.Debug {
		logger.SetLevel(logrus.DebugLevel)
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
