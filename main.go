package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cache"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Name    string
	Host    string
	Port    int
	Cache   bool
	Log     bool
	Service map[string]map[string]map[string]interface{}
}

type CacheData struct {
	Servers []string
	Method  string
}

var (
	cfg       Config
	cfgMutex  sync.RWMutex
	cacheData struct {
		sync.RWMutex
		data map[string]map[string]map[string]CacheData
	}
)

func main() {
	if err := loadConfig(); err != nil {
		log.Println("Failed to load config:", err)
		return
	}

	go watchConfigChanges()

	log.Println("Starting", cfg.Name, "on", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port))

	app := fiber.New(
		fiber.Config{
			DisableStartupMessage: true,
		},
	)

	if cfg.Log {
		log.Println("Logging enabled")
		app.Use(logger.New())
	}

	if cfg.Cache {
		log.Println("Cache enabled")
		app.Use(cache.New())
	}

	app.All("/:path/:version/:service_name/*", func(c *fiber.Ctx) error {
		path := c.Params("path")
		version := c.Params("version")
		serviceName := c.Params("service_name")

		cfgMutex.RLock()
		service, ok := cfg.Service[path][version][serviceName]
		cfgMutex.RUnlock()

		if !ok {
			return c.Status(404).SendString("Not found")
		}

		serviceMap, ok := service.(map[string]interface{})
		if !ok {
			return c.Status(404).SendString("Invalid service configuration")
		}

		cacheData.RLock()
		cache, ok := cacheData.data[path][version][serviceName]
		cacheData.RUnlock()

		if !ok {
			method, servers := parseServers(serviceMap)
			cacheData.Lock()
			if cacheData.data == nil {
				cacheData.data = make(map[string]map[string]map[string]CacheData)
			}
			if cacheData.data[path] == nil {
				cacheData.data[path] = make(map[string]map[string]CacheData)
			}
			if cacheData.data[path][version] == nil {
				cacheData.data[path][version] = make(map[string]CacheData)
			}
			cacheData.data[path][version][serviceName] = CacheData{
				Servers: servers,
				Method:  method,
			}
			cache = cacheData.data[path][version][serviceName]
			cacheData.Unlock()
		}

		servers := cache.Servers
		method := cache.Method

		if servers == nil {
			return c.Status(500).SendString("Internal server error")
		}

		url := strings.Join([]string{chooseServer(method, servers), c.Params("*")}, "/")

		log.Println("Proxying to", url)

		if err := proxy.Do(c, url); err != nil {
			return c.Status(500).SendString(err.Error())
		}

		return nil
	})

	app.Listen(fmt.Sprintf("%s:%d", cfg.Host, cfg.Port))
}

func loadConfig() error {
	doc, err := os.ReadFile("config.toml")
	if err != nil {
		return err
	}

	var newConfig Config
	err = toml.Unmarshal(doc, &newConfig)
	if err != nil {
		return err
	}

	cfgMutex.Lock()
	cfg = newConfig
	cfgMutex.Unlock()

	return nil
}

func watchConfigChanges() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println("Failed to create config watcher:", err)
		return
	}
	defer watcher.Close()

	err = watcher.Add("config.toml")
	if err != nil {
		log.Println("Failed to watch config file:", err)
		return
	}

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				log.Println("Config file changed. Reloading...")
				if err := loadConfig(); err != nil {
					log.Println("Failed to reload config:", err)
				}
				clearCacheData()
			}
		case err := <-watcher.Errors:
			log.Println("Config watcher error:", err)
		}
	}
}

func chooseServer(method string, list []string) string {
	switch method {
	case "first":
		return list[0]
	case "random":
		return list[rand.Intn(len(list))]
	default:
		return list[rand.Intn(len(list))]
	}
}

func parseServers(m map[string]interface{}) (method string, servers []string) {
	for k, v := range m {
		switch k {
		case "method":
			method = v.(string)
		case "servers":
			for _, s := range v.([]interface{}) {
				servers = append(servers, s.(string))
			}
		}
	}
	return
}

func clearCacheData() {
	cacheData.Lock()
	cacheData.data = nil
	cacheData.Unlock()
}
