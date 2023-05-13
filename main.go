package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
	log "k8s.io/klog/v2"
)

// forward http://xxx.connector.vanua.ai to http://xxx.connector.vanua.ai/api/v1/source/{type}/{connector-id}

func main() {
	var c Config
	err := ParseConfig(&c)
	if err != nil {
		panic(err)
	}
	mapping := make(map[string]Connector, 100)
	for service, connector := range c.Connector {
		for uuid, id := range connector {
			mapping[uuid] = Connector{
				ID:      id,
				Kind:    getConnectorKind(service),
				Type:    getConnectorType(service),
				Service: service,
			}
		}
	}

	log.Infof("show connector mapping: %+v\n", mapping)

	proxy := &Proxy{
		mapping: mapping,
	}
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", c.Port),
		Handler: proxy,
	}
	log.Infof("proxy start on port: %d\n", c.Port)
	err = server.ListenAndServe()
	if err != nil {
		panic("proxy listen error " + err.Error())
	}
}

type Proxy struct {
	mapping map[string]Connector
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// xxx.connector.vanua.ai
	log.Infof("receive request, host: %s, path: %s\n", r.Host, r.URL.Path)
	hosts := strings.Split(r.Host, ".")
	if len(hosts) != 4 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	uuid := hosts[0]
	connector, ok := p.mapping[uuid]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	path := fmt.Sprintf("/api/v1/%s/%s/%s", connector.Kind, connector.Type, connector.ID)
	log.Infof("forward request, host: %s, path: %s\n", connector.Service, path)
	director := func(req *http.Request) {
		req.URL.Scheme = "http"
		req.URL.Host = connector.Service
		req.URL.Path = path
		req.Host = connector.Service
		req.Header.Add("X-Conenctor-ID", connector.ID)
	}
	proxy := &httputil.ReverseProxy{Director: director}
	proxy.ServeHTTP(w, r)
}

type Connector struct {
	ID      string
	Kind    string
	Type    string
	Service string
}

type Config struct {
	Port int `yaml:"port"`
	// [service][uuid]=connectorID
	Connector map[string]map[string]string `yaml:"connector"`
}

func ParseConfig(c *Config) error {
	file := os.Getenv("CONNECTOR_CONFIG")
	if file == "" {
		file = "./config/config.yml"
	}
	bytes, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(bytes, c)
}

func getConnectorKind(service string) string {
	return strings.Split(strings.Split(service, ".")[0], "-")[0]
}

func getConnectorType(service string) string {
	return strings.Split(strings.Split(service, ".")[0], "-")[1]
}
