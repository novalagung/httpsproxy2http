package main

import (
	"crypto/tls"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"golang.org/x/crypto/acme/autocert"
)

var cachedViewTempalte *template.Template

func main() {
	r := chi.NewRouter()

	r.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"HEAD", "OPTIONS", "GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
		Debug:            !isEnvProduction(),
	}).Handler)

	r.Handle("/*", http.HandlerFunc(reverseProxyHandler))
	r.Handle("/.well-known/acme-challenge/", http.FileServer(http.FileSystem(http.Dir("etc/letsencrypt/assets"))))

	if isEnvProduction() {
		startProductionWebServer(r)
	} else {
		startDevelopmentWebServer(r)
	}
}

func isEnvProduction() bool {
	env := strings.ToLower(os.Getenv("ENV"))
	return env == "prod" || env == "production"
}

func startDevelopmentWebServer(r http.Handler) {
	serverHTTP := new(http.Server)
	serverHTTP.Handler = r
	serverHTTP.Addr = ":http"

	log.Println("starting web server in development mode")

	err := serverHTTP.ListenAndServe()
	if err != nil {
		log.Fatal(err.Error())
	}
}

func startProductionWebServer(r http.Handler) {
	host := os.Getenv("HOST")
	if host == "" {
		log.Fatal("env var HOST must not be empty")
	}
	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(host),
		Cache:      autocert.DirCache(fmt.Sprintf("/etc/letsencrypt/live/%s/", host)),
	}

	log.Println("starting web server")

	serverHTTP := new(http.Server)
	serverHTTP.Handler = certManager.HTTPHandler(nil)
	serverHTTP.Addr = ":http"
	go serverHTTP.ListenAndServe()

	serverHTTPS := new(http.Server)
	serverHTTPS.Handler = r
	serverHTTPS.Addr = ":https"
	serverHTTPS.TLSConfig = &tls.Config{GetCertificate: certManager.GetCertificate}
	err := serverHTTPS.ListenAndServeTLS("", "")
	if err != nil {
		log.Fatal(err.Error())
	}
}

func reverseProxyHandler(w http.ResponseWriter, r *http.Request) {
	rPath := strings.Trim(r.URL.Path, "/")

	for query := range r.URL.Query() {
		if !strings.Contains(rPath, "?") {
			rPath = fmt.Sprintf("%s?%s=%s", rPath, query, r.URL.Query().Get(query))
		} else {
			rPath = fmt.Sprintf("%s&%s=%s", rPath, query, r.URL.Query().Get(query))
		}
	}
	if rPath == "" || rPath == "/" {
		scheme := "http://"
		if isEnvProduction() {
			scheme = "https://"
		}

		viewData := map[string]interface{}{
			"host": scheme + os.Getenv("HOST"),
		}

		if cachedViewTempalte == nil {
			cachedViewTempalte = template.Must(template.ParseFiles("view.html"))
		}
		if err := cachedViewTempalte.Execute(w, viewData); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	parts := strings.Split(strings.Trim(rPath, "/"), "/")
	if len(parts) < 1 {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	var destinationHost string
	if strings.HasPrefix(parts[0], "http") {
		destinationHost = fmt.Sprintf("%s://%s", parts[0], strings.Join(parts[1:], "/"))
	} else {
		destinationHost = fmt.Sprintf("http://%s", rPath)
	}

	destinationURL, err := url.Parse(destinationHost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Println("incoming request from", r.RemoteAddr, "to", destinationURL.String())

	reverseProxy := new(httputil.ReverseProxy)
	reverseProxy.Director = func(dr *http.Request) {
		dr.Host = destinationURL.Host
		dr.URL = destinationURL
		dr.Header = r.Header
		dr.Body = r.Body
	}

	reverseProxy.ServeHTTP(w, r)
}
