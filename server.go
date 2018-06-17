package dotweb

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

// Config provides config values for the webserver
type Config struct {

	// The hostname the webserver listens for
	// For localhost leave host blank
	// e.g. dotcookie.me, www.dotcookie.me
	Host string `json:"host"`

	// Port defines the port to listen on for HTTP requests
	Port int `json:"port"`

	// DB is the connection string for database
	DB string `json:"db"`

	// The function that handles all incoming HTTP and HTTPS requests
	Handler http.HandlerFunc `json:"-"`
}

// DefaultConfig provides the default configurations
func DefaultConfig() Config {
	return Config{
		Host: "",
		Port: 80,
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
// -config string
//   path to json config file, overrides flags
// -host string
//   hostname to listen on. Leave blank to listen for localhost
// -port int
//   port to listen on for HTTP requests (default 80)
// -RedirectHTTP
//   redirect all HTTP requests to HTTPS (default true)
func ConfigFromFlags(args []string) (*Config, error) {
	defaultConfig := DefaultConfig()
	flags := flag.NewFlagSet("dotweb", flag.ContinueOnError)
	host := flags.String("host", defaultConfig.Host, "hostname to listen on. Leave blank to listen for localhost")
	port := flags.Int("port", defaultConfig.Port, "port to listen on for HTTP requests")
	db := flags.String("db", defaultConfig.DB, "database connection string")
	configFile := flags.String("config", "", "path to json config file, overrides flags")
	err := flags.Parse(args)
	if err != nil {
		return nil, err
	}
	if len(*configFile) > 0 {
		return loadConfig(*configFile)
	}
	return &Config{
		Host: *host,
		Port: *port,
		DB:   *db,
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
	port := ":" + strconv.Itoa(config.Port)
	httpServer := http.Server{
		Addr:    config.Host + ":" + strconv.Itoa(config.Port),
		Handler: config.Handler,
	}
	log.Println("starting listening on", config.Host+port)
	return httpServer.ListenAndServe()
}
