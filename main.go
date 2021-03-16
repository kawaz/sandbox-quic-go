package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"

	"net/http/httputil"

	"github.com/kawaz/go-oreorecert"
	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/sirupsen/logrus"
)

func main() {

	apiEndpoint, err := url.Parse("https://oreore.net/")
	if err != nil {
		panic(nil)
	}

	mux := http.NewServeMux()
	mux.Handle("/admin/api/", httputil.NewSingleHostReverseProxy(apiEndpoint))
	mux.Handle("/", http.FileServer(http.Dir(".")))

	logger := logrus.New()
	logger.Formatter = new(logrus.JSONFormatter)
	w := logger.Writer()
	defer w.Close()

	cert := oreorecert.GetKeyPairOreoreNet()
	httpServer := &http.Server{
		Addr:     "localhost:5000",
		ErrorLog: log.New(w, "", 0),
	}
	quicServer := &http3.Server{
		Server:     httpServer,
		QuicConfig: &quic.Config{},
	}
	httpServer.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		quicServer.SetQuicHeaders(w.Header())
		mux.ServeHTTP(w, r)
	})

	httpErr := make(chan error)
	go func() {
		httpErr <- httpServer.ListenAndServeTLS(cert.CertFile, cert.KeyFile)
	}()
	quicErr := make(chan error)
	go func() {
		quicErr <- quicServer.ListenAndServeTLS(cert.CertFile, cert.KeyFile)
	}()

	err = func() error {
		select {
		case err := <-httpErr:
			quicServer.Close()
			return fmt.Errorf("httpServer Error: %s", err)
		case err := <-quicErr:
			// Cannot close the HTTP server or wait for requests to complete properly :/
			return fmt.Errorf("quicServer Error: %s", err)
		}
	}()
	if err != nil {
		panic(err)
	}
}
