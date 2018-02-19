package dotweb

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/acme/autocert"
)

// A struct to provide configuration data for the web server.
type Config struct {

	// The hostname the webserver listens for
	// For localhost leave host blank
	// e.g. dotcookie.me, www.dotcookie.me
	Host string `json:"host"`

	// The port to listen on for HTTP requests
	HttpPort int `json:"http"`

	// The port to listen on for HTTPS requests
	HttpsPort int `json:"https"`

	// The function that handles all incoming HTTP and HTTPS requests
	Handler http.HandlerFunc `json:"-"`

	// The directiory where the SSL certificates are stored
	// If the string is empty, HTTPS will not be available
	CertsDir string `json:"certsDir"`

	// If true all HTTP requests will be redirected to HTTPS
	// ACME "http-01" challenge will not be redirects to HTTPS!
	// See https://godoc.org/golang.org/x/crypto/acme/autocert#Manager.HTTPHandler
	RedirectHttp bool `json:"redirectHttp"`
}

// Function that provides a default configuration
func DefaultConfig() Config {
	return Config{
		Host:         "",
		HttpPort:     80,
		HttpsPort:    443,
		RedirectHttp: true,
		CertsDir:     "certs",
	}
}

// Loads config file from flags.
// Default values for configuration are provided by function dotweb.DefaultConfig().
// If somthing goes wrong an error is returned.
//
// Use run arguments:
// 	dotweb.ConfigFromFlags(os.Args[1:])
//
// Usage of dotweb:
// -certsDir string
//   directory to save the certificates to (default "certs")
// -config string
//   path to json config file, overrides flags
// -host string
//   hostname to listen on. Leave blank to listen for localhost
// -http int
//   port to listen on for HTTP requests (default 80)
// -https int
//   port to listen on for HTTPS requests (default 443)
// -redirectHttp
//   redirect all HTTP requests to HTTPS (default true)
func ConfigFromFlags(args []string) (*Config, error) {
	defaultConfig := DefaultConfig()
	flags := flag.NewFlagSet("dotweb", flag.ContinueOnError)
	host := flags.String("host", defaultConfig.Host, "hostname to listen on. Leave blank to listen for localhost")
	httpPort := flags.Int("http", defaultConfig.HttpPort, "port to listen on for HTTP requests")
	httpsPort := flags.Int("https", defaultConfig.HttpsPort, "port to listen on for HTTPS requests")
	certsDir := flags.String("certsDir", defaultConfig.CertsDir, "directory to save the certificates to")
	redirect := flags.Bool("redirectHttp", defaultConfig.RedirectHttp, "redirect all HTTP requests to HTTPS")
	configFile := flags.String("config", "", "path to json config file, overrides flags")
	err := flags.Parse(args)
	if err != nil {
		return nil, err
	}
	if len(*configFile) > 0 {
		return loadConfig(*configFile)
	}
	return &Config{
		Host:         *host,
		HttpPort:     *httpPort,
		HttpsPort:    *httpsPort,
		RedirectHttp: *redirect,
		CertsDir:     *certsDir,
	}, nil
}

// Load config from json file
// Path is the location of the file
func loadConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// Starting a webserver and provide the configuration via json file.
// See dotweb.StartWebServer(dotweb.Config) for further explanations
func StartWebServerFromConfig(configFile string, handler http.HandlerFunc) error {
	config, err := loadConfig(configFile)
	if err != nil {
		return err
	}
	config.Handler = handler
	return StartWebServer(*config)
}

// Starting a webserver with the given configurations
// See dotweb.Config for configuration options
// If config.CertsDir is empty HTTPS will not be available
//
// All incomminng requests on HTTP and HTTPS port will be directed to config.Handler
func StartWebServer(config Config) error {
	httpPort := ":" + strconv.Itoa(config.HttpPort)
	httpsPort := ":" + strconv.Itoa(config.HttpsPort)
	httpsAvailable := true
	if len(config.CertsDir) == 0 {
		httpsAvailable = false
		log.Println("warning: no certs dir was provided, https was disabled")
	} else {
		_, err := os.Open(config.CertsDir)
		if err != nil {
			err = os.Mkdir(config.CertsDir, os.ModePerm)
			if err != nil {
				return err
			}
		}
	}
	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(config.Host), //your domain here
		Cache:      autocert.DirCache(config.CertsDir),  //folder for storing certificates
	}
	httpsServer := http.Server{
		Addr: config.Host + httpsPort,
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
		},
		Handler: config.Handler,
	}
	httpServer := http.Server{
		Addr: config.Host + ":" + strconv.Itoa(config.HttpPort),
		Handler: certManager.HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if config.RedirectHttp && httpsAvailable {
				host := r.Host
				if strings.HasSuffix(host, httpPort) {
					host = strings.TrimSuffix(host, httpPort) + httpsPort
				}
				http.Redirect(w, r, "https://"+host+r.URL.String(), http.StatusMovedPermanently)
			} else {
				if config.Handler != nil {
					config.Handler(w, r)
				}
			}
		})),
	}
	if httpsAvailable {
		log.Println("starting listening on", config.Host+httpsPort)
		go httpsServer.ListenAndServeTLS("", "")
	}
	log.Println("starting listening on", config.Host+httpPort)
	return httpServer.ListenAndServe()
}
