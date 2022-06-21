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

const proxyTypeForward = "forward"
const proxyTypeReverse = "reverse"

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
	r.Handle("/.well-known/acme-challenge/", http.FileServer(http.FileSystem(http.Dir("/etc/letsencrypt/assets"))))

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

	go func() {
		serverHTTP := new(http.Server)
		serverHTTP.Handler = certManager.HTTPHandler(nil)
		serverHTTP.Addr = ":http"
		log.Fatal(serverHTTP.ListenAndServe())
	}()

	go func() {
		serverHTTPS := new(http.Server)
		serverHTTPS.Handler = r
		serverHTTPS.Addr = ":https"
		serverHTTPS.TLSConfig = &tls.Config{GetCertificate: certManager.GetCertificate}
		log.Fatal(serverHTTPS.ListenAndServeTLS("", ""))
	}()

	select {}
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

func reverseProxyHandler(w http.ResponseWriter, r *http.Request) {
	rPath := constructPathWithQueryString(r.URL)
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

	proxyType, destinationURLString := constructDestination(rPath, r.Header.Get("Referer"))
	destinationURL, err := url.Parse(destinationURLString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	logString := fmt.Sprintf("incoming request from %s to %s", r.RemoteAddr, destinationURL.String())
	if proxyType != "" {
		logString = fmt.Sprintf("%s (type: %s proxy)", logString, proxyType)
	}

	reverseProxy := new(httputil.ReverseProxy)
	reverseProxy.Director = func(dr *http.Request) {
		if proxyType == proxyTypeReverse {
			dr.Host = destinationURL.Host
		}
		dr.URL = destinationURL
		dr.Body = r.Body
		dr.Header = r.Header
	}
	reverseProxy.ModifyResponse = func(r *http.Response) error {
		if proxyType == proxyTypeForward {
			r.StatusCode = http.StatusTemporaryRedirect
		}
		return nil
	}

	reverseProxy.ServeHTTP(w, r)
}

func parseURL(path string) (string, string) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	proxyType := parts[0]
	destinationURLString := ""

	if proxyType == proxyTypeReverse {
		destinationURLString = fmt.Sprintf("http://%s", strings.Join(parts[1:], "/"))
	} else if proxyType == proxyTypeForward {
		destinationURLString = fmt.Sprintf("http://%s", strings.Join(parts[1:], "/"))
	} else {
		proxyType = proxyTypeForward
		destinationURLString = fmt.Sprintf("http://%s", path)
	}

	return proxyType, destinationURLString
}

func constructDestination(path, referer string) (string, string) {
	proxyType, destinationURLString := parseURL(path)
	if referer == "" {
		return proxyType, destinationURLString
	}

	appHost := strings.TrimSpace(os.Getenv("HOST"))
	refererURL, _ := url.Parse(referer)

	if appHost == refererURL.Host {
		_, referer = parseURL(constructPathWithQueryString(refererURL))
		refererURL, _ := url.Parse(referer)

		destinationURL, _ := url.Parse(path)
		destinationPath := constructPathWithQueryString(destinationURL)

		destinationURLString = refererURL.Scheme + "://" + refererURL.Host + "/" + destinationPath
	}

	return proxyType, destinationURLString
}

func constructPathWithQueryString(u *url.URL) string {
	path := strings.Trim(u.Path, "/")

	for query := range u.Query() {
		if !strings.Contains(path, "?") {
			path = fmt.Sprintf("%s?%s=%s", path, query, url.QueryEscape(u.Query().Get(query)))
		} else {
			path = fmt.Sprintf("%s&%s=%s", path, query, url.QueryEscape(u.Query().Get(query)))
		}
	}

	log.Println("=================", path)

	return path
}
