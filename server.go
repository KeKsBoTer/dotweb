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

// Config provides config values for the webserver
type Config struct {

	// The hostname the webserver listens for
	// For localhost leave host blank
	// e.g. dotcookie.me, www.dotcookie.me
	Host string `json:"host"`

	// HTTPPort defines the port to listen on for HTTP requests
	HTTPPort int `json:"http"`

	// HTTPSPort defines the port to listen on for HTTP requests
	HTTPSPort int `json:"https"`

	// The function that handles all incoming HTTP and HTTPS requests
	Handler http.HandlerFunc `json:"-"`

	// The directiory where the SSL certificates are stored
	// If the string is empty, HTTPS will not be available
	CertsDir string `json:"certsDir"`

	// If RedirectHTTP is true all HTTP requests will be redirected to HTTPS
	// ACME "http-01" challenge will not be redirects to HTTPS!
	// See https://godoc.org/golang.org/x/crypto/acme/autocert#Manager.HTTPHandler
	RedirectHTTP bool `json:"RedirectHTTP"`
}

// DefaultConfig provides the default configurations
func DefaultConfig() Config {
	return Config{
		Host:         "",
		HTTPPort:     80,
		HTTPSPort:    443,
		RedirectHTTP: true,
		CertsDir:     "certs",
	}
}

// ConfigFromFlags loads config file from flags.
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
// -RedirectHTTP
//   redirect all HTTP requests to HTTPS (default true)
func ConfigFromFlags(args []string) (*Config, error) {
	defaultConfig := DefaultConfig()
	flags := flag.NewFlagSet("dotweb", flag.ContinueOnError)
	host := flags.String("host", defaultConfig.Host, "hostname to listen on. Leave blank to listen for localhost")
	HTTPPort := flags.Int("http", defaultConfig.HTTPPort, "port to listen on for HTTP requests")
	HTTPSPort := flags.Int("https", defaultConfig.HTTPSPort, "port to listen on for HTTPS requests")
	certsDir := flags.String("certsDir", defaultConfig.CertsDir, "directory to save the certificates to")
	redirect := flags.Bool("RedirectHTTP", defaultConfig.RedirectHTTP, "redirect all HTTP requests to HTTPS")
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
		HTTPPort:     *HTTPPort,
		HTTPSPort:    *HTTPSPort,
		RedirectHTTP: *redirect,
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

// StartWebServerFromConfig starts a webserver and provides the configuration via json file.
// See dotweb.StartWebServer(dotweb.Config) for further explanations
func StartWebServerFromConfig(configFile string, handler http.HandlerFunc) error {
	config, err := loadConfig(configFile)
	if err != nil {
		return err
	}
	config.Handler = handler
	return StartWebServer(*config)
}

// StartWebServer starts a webserver with the given configurations
// See dotweb.Config for configuration options
// If config.CertsDir is empty HTTPS will not be available
//
// All incomminng requests on HTTP and HTTPS port will be directed to config.Handler
func StartWebServer(config Config) error {
	HTTPPort := ":" + strconv.Itoa(config.HTTPPort)
	HTTPSPort := ":" + strconv.Itoa(config.HTTPSPort)
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
		Addr: config.Host + HTTPSPort,
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
		},
		Handler: config.Handler,
	}
	httpServer := http.Server{
		Addr: config.Host + ":" + strconv.Itoa(config.HTTPPort),
		Handler: certManager.HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if config.RedirectHTTP && httpsAvailable {
				host := r.Host
				if strings.HasSuffix(host, HTTPPort) {
					host = strings.TrimSuffix(host, HTTPPort) + HTTPSPort
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
		log.Println("starting listening on", config.Host+HTTPSPort)
		go httpsServer.ListenAndServeTLS("", "")
	}
	log.Println("starting listening on", config.Host+HTTPPort)
	return httpServer.ListenAndServe()
}
