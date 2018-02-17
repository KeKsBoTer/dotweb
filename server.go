package dotweb

import (
	"net/http"
	"strconv"
	"os"
	"golang.org/x/crypto/acme/autocert"
	"crypto/tls"
	"strings"
	"log"
	"io/ioutil"
	"encoding/json"
	"flag"
)

type Config struct {
	Host         string           `json:"host"`
	HttpPort     int              `json:"http"`
	HttpsPort    int              `json:"https"`
	Handler      http.HandlerFunc `json:"-"`
	CertsDir     string           `json:"certsDir"`
	RedirectHttp bool             `json:"redirectHttp"`
}

func DefaultConfig() Config {
	return Config{
		Host:         "",
		HttpPort:     80,
		HttpsPort:    443,
		RedirectHttp: true,
		CertsDir:     "certs",
	}
}

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

func loadConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	config := DefaultConfig()
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func StartWebServerFromConfig(configFile string, handler http.HandlerFunc) (error) {
	config, err := loadConfig(configFile)
	if err != nil {
		return err
	}
	config.Handler = handler
	return StartWebServer(*config)
}
func StartWebServer(config Config) (error) {
	httpPort := ":" + strconv.Itoa(config.HttpPort)
	httpsPort := ":" + strconv.Itoa(config.HttpsPort)
	_, err := os.Open(config.CertsDir)
	if err != nil {
		err = os.Mkdir(config.CertsDir, os.ModePerm)
		if err != nil {
			return err
		}
	}
	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(config.Host, "www."+config.Host), //your domain here
		Cache:      autocert.DirCache(config.CertsDir),                      //folder for storing certificates
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
			if config.RedirectHttp {
				host := r.Host
				if strings.HasSuffix(host, httpPort) {
					host = strings.TrimSuffix(host, httpPort) + httpsPort
				}
				http.Redirect(w, r, "https://"+host+r.URL.String(), http.StatusMovedPermanently)
			} else {
				config.Handler(w, r)
			}
		})),
	}
	log.Println("starting listening on", config.Host+httpsPort)
	go httpsServer.ListenAndServeTLS("", "")
	log.Println("starting listening on", config.Host+httpPort)
	return httpServer.ListenAndServe()
}
