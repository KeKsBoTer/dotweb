package dotweb

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// testing
func TestLoadConfig(t *testing.T) {
	for i := 0; i < 100; i++ {
		config := Config{
			Host:         randStringBytes(100),
			HttpPort:     rand.Int(),
			HttpsPort:    rand.Int(),
			RedirectHttp: rand.Int()%2 == 0, // generate random bool
			CertsDir:     randStringBytes(100),
		}
		configFile, err := json.Marshal(config)
		if err != nil {
			t.Fatal("failed to create content for test file:", err)
		}
		t.Log("testing for \"" + string(configFile) + "\"")
		fileName := "testLoadConfig_" + strconv.Itoa(int(time.Now().Unix()))
		err = ioutil.WriteFile(fileName, configFile, os.ModePerm)
		if err != nil {
			t.Fatal("failed to create config file:", err)
		}
		loadedConfig, err := loadConfig(fileName)
		if err != nil {
			t.Fatal("failed to load config file:", err)
		}
		if !reflect.DeepEqual(*loadedConfig, config) {
			t.Fatal("loaded config file does not have the same fields as saved config")
		}
		err = os.Remove(fileName)
		if err != nil {
			t.Error("failed to delete config file:", err)
		}
	}
}

func TestStartWebServer(t *testing.T) {
	go func() {
		config, err := ConfigFromFlags([]string{
			"-http=8080", "-redirectHttp=false",
		})
		if err != nil {
			if err == flag.ErrHelp {
				return
			} else {
				log.Fatal(err)
			}
		}
		config.Handler = func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "It works")
		}
		err = StartWebServer(*config)
		if err != nil {
			log.Fatal(err)
		}
	}()
	resp, err := http.Get("http://localhost:8080")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("web server returned status code %d, expected %d", resp.StatusCode, http.StatusOK)
	} else {
		var data []byte
		resp.Body.Read(data)
		t.Log(string(data))
	}
}
