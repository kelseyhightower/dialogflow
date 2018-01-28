// Copyright 2018 Google Inc. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package fulfillment

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/dialogflow/v2beta1"
)

// ErrEmptyHashedPassword is returned from ListenAndServe and ListenAndServeTLS when basic
// authentication is required and the hashed password is empty.
var ErrEmptyHashedPassword = errors.New("dialogflow/fulfillment: basic auth hashed password is empty")

// ErrEmptyUsername is returned from ListenAndServe and ListenAndServeTLS when basic
// authentication is required and the username is empty.
var ErrEmptyUsername = errors.New("dialogflow/fulfillment: basic auth username is empty")

// DefaultCacheDirectory is the default directory to use when caching
// certificates from Let's Encrypt.
var DefaultCacheDirectory = "/var/lib/dialogflow/fulfillment"

// An ActionFunc processes an dialogflow.WebhookRequest and returns a
// dialogflow.WebhookResponse.
type ActionFunc func(*dialogflow.WebhookRequest) (*dialogflow.WebhookResponse, error)

// An Actions represents the supported actions of a fulfillment server.
type Actions map[string]ActionFunc

// NewActions returns a new, empty Actions map.
func NewActions() Actions {
	actions := make(map[string]ActionFunc)
	return actions
}

// Set sets the ActionFunc entry associated with name.
// It replaces any existing value associated with name.
func (a Actions) Set(name string, fn ActionFunc) {
	a[name] = fn
}

// A Server defines parameters for running a fulfillment server.
//
// A Server must be initialized before use.
type Server struct {
	// ACMEHTTPChallengeServer holds the ACME HTTP challenge server.
	ACMEHTTPChallengeServer *http.Server

	// Actions used by fulfillment handler.
	Actions Actions

	// AutocertCache specifies an optional autocert.Cache implementation used
	// to store and retrieve previously obtained Let's Encrypt certificates as
	// opaque data.
	//
	// If AutocertCache is nil, autocert.DirCache will be used.
	// If AutocertCache is not nil, Let's Encrypt is not enabled automatically.
	AutocertCache autocert.Cache

	// BasicAuthUsername specifies an optional basic authentication username used
	// to authenticate fulfillment requests.
	//
	// If BasicAuthUsername is blank, DefaultBasicAuthUsername will be used.
	BasicAuthUsername string

	// BasicAuthHashedPassword specifies an optional basic authentication hashed
	// password used to authenticate HTTPS requests.
	//
	// BasicAuthHashedPassword must be hashed using bcrypt.
	//
	// If BasicAuthHashedPassword is not blank, basic authentication is enabled
	// for all fulfillment requests.
	BasicAuthHashedPassword string

	// CacheDirectory specifies an optional directory to use when caching
	// certificates from Let's Encrypt. If CacheDirectory is not blank,
	// Let's Encrypt is not enabled automatically.
	CacheDirectory string

	// Domain specifies an optional fully qualifed domain used when generating
	// certificates from Let's Encrypt. If Domain is not blank, Let's Encrypt
	// is enabled automatically.
	Domain string

	// DisableBasicAuth, if true, basic authentication is disabled for fulfillment
	// requests.
	//
	// If DisableBasicAuth is false, BasicAuthUsername and BasicAuthHashedPassword
	// must be set and non-empty. Defaults to false.
	DisableBasicAuth bool

	// HealthServer holds the health server.
	HealthServer *http.Server

	autocertManager  *autocert.Manager
	status           int
	basicAuthEnabled bool

	*http.Server
}

// NewServer initializes and returns a new Server.
func NewServer() *Server {
	s := &Server{}
	s.Actions = NewActions()

	s.ACMEHTTPChallengeServer = &http.Server{}

	fulfillment := http.NewServeMux()
	fulfillment.Handle("/", Handler(s.Actions))
	s.Server = &http.Server{Handler: fulfillment}

	health := http.NewServeMux()
	health.HandleFunc("/health", s.healthHandler)
	s.HealthServer = &http.Server{Handler: health}

	return s
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(s.Status())
}

type handler struct {
	actions Actions
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	r.Body.Close()

	var webhookRequest dialogflow.WebhookRequest
	err = json.Unmarshal(body, &webhookRequest)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	action := webhookRequest.QueryResult.Action
	fn, ok := h.actions[action]
	if !ok {
		log.Printf("Action %s not supported.", action)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Printf("Invoking action: %s", action)

	response, err := fn(&webhookRequest)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	data, err := json.MarshalIndent(response, "", " ")
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// Handler returns a handler that processes Dialogflow fulfillment requests
// using the given actions.
func Handler(actions Actions) http.Handler {
	return &handler{actions}
}

// basicAuthHandler returns a request handler that authenicates each request it
// receives using the given username and hashedPassword.
func basicAuthHandler(username, hashedPassword string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", "Basic")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if u != username {
			w.Header().Set("WWW-Authenticate", "Basic")
			w.WriteHeader(http.StatusForbidden)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(p)); err != nil {
			log.Println(err)
			w.Header().Set("WWW-Authenticate", "Basic")
			w.WriteHeader(http.StatusForbidden)
			return
		}

		h.ServeHTTP(w, r)
	})
}

// ListenAndServe listens on TCP address s.Addr to handle Dialogflow
// fulfillment requests, s.HealthServer.Addr to handle health checks.
//
// If s.Addr is blank, "0.0.0.0:80" is used.
// If s.HealthServer.Addr is blank, "0.0.0.0:8080" is used.
//
// ListenAndServe always returns a non-nil error.
func (s *Server) ListenAndServe() error {
	if !s.DisableBasicAuth {
		if s.BasicAuthUsername == "" {
			return ErrEmptyUsername
		}
		if s.BasicAuthHashedPassword == "" {
			return ErrEmptyHashedPassword
		}
		if s.BasicAuthHashedPassword != "" {
			s.Server.Handler = basicAuthHandler(s.BasicAuthUsername, s.BasicAuthHashedPassword, s.Server.Handler)
		}
	}

	if s.Server.Addr == "" {
		s.Server.Addr = "0.0.0.0:80"
	}
	if s.HealthServer.Addr == "" {
		s.HealthServer.Addr = "0.0.0.0:8080"
	}

	// Start the health server.
	go func() {
		if err := s.HealthServer.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				return
			}
			log.Println(err)
		}
	}()

	s.SetStatus(http.StatusOK)

	return s.Server.ListenAndServe()
}

// ListenAndServeTLS listens on TCP address s.Addr to handle Dialogflow
// fulfillment requests, and s.HealthServer.Addr to handle health checks.
//
// If s.Addr is blank, "0.0.0.0:443" is used.
// If s.HealthServer.Addr is blank, "0.0.0.0:8080" is used.
// If s.ACMEHTTPChallengeServer.Addr is blank, "0.0.0.0:80" is used.
//
// If s.Domain is not blank, Let's Encrypt is enabled automatically for the
// domain. Use of this function implies acceptance of the LetsEncrypt Terms
// of Service.
//
// Let's Encrypt certificates are cached using an implementation of
// autocert.Cache specified by s.AutocertCache. If s.AutocertCache is nil,
// an autocert.Cache will be created based on s.CacheDirectory or s.Bucket.
// If s.CacheDirectory and s.Bucket are blank, an autocert.Cache will be created
// based on DefaultCacheDirectory.
//
// ListenAndServeTLS always returns a non-nil error.
func (s *Server) ListenAndServeTLS(certFile, keyFile string) error {
	if !s.DisableBasicAuth {
		if s.BasicAuthUsername == "" {
			return ErrEmptyUsername
		}
		if s.BasicAuthHashedPassword == "" {
			return ErrEmptyHashedPassword
		}
		if s.BasicAuthHashedPassword != "" {
			s.Server.Handler = basicAuthHandler(s.BasicAuthUsername, s.BasicAuthHashedPassword, s.Server.Handler)
		}
	}

	if s.Server.Addr == "" {
		s.Server.Addr = "0.0.0.0:443"
	}
	if s.HealthServer.Addr == "" {
		s.HealthServer.Addr = "0.0.0.0:8080"
	}
	if s.ACMEHTTPChallengeServer.Addr == "" {
		s.ACMEHTTPChallengeServer.Addr = "0.0.0.0:80"
	}

	if s.Domain != "" {
		s.autocertManager = &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(s.Domain),
		}

		if s.AutocertCache != nil {
			s.autocertManager.Cache = s.AutocertCache
		}

		if s.CacheDirectory != "" && s.autocertManager.Cache == nil {
			s.autocertManager.Cache = autocert.DirCache(s.CacheDirectory)
		}

		if s.autocertManager.Cache == nil {
			s.autocertManager.Cache = autocert.DirCache(DefaultCacheDirectory)
		}

		s.ACMEHTTPChallengeServer.Handler = s.autocertManager.HTTPHandler(nil)

		s.Server.TLSConfig = &tls.Config{
			GetCertificate: s.autocertManager.GetCertificate,
		}

		s.SetStatus(http.StatusOK)

		// Start the ACME HTTP challenge server.
		go func() {
			if err := s.ACMEHTTPChallengeServer.ListenAndServe(); err != nil {
				if err == http.ErrServerClosed {
					return
				}

				s.SetStatus(http.StatusServiceUnavailable)
				log.Println(err)
			}
		}()
	}

	// Start the health server.
	go func() {
		if err := s.HealthServer.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				return
			}
			log.Println(err)
		}
	}()

	return s.Server.ListenAndServeTLS(certFile, keyFile)
}

// ListenAndServeUntilSignal invokes ListenAndServe and blocks until the one of
// the given OS signals are called.
func (s *Server) ListenAndServeUntilSignal(sig ...os.Signal) {
	go func() {
		if err := s.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				return
			}
			log.Println(err)
		}
	}()

	if len(sig) == 0 {
		sig = append(sig, syscall.SIGINT, syscall.SIGTERM)
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, sig...)
	<-signalChan

	log.Println("Shutdown signal received, exiting...")
	s.Shutdown()
}

func (s *Server) ListenAndServeTLSUntilSignal(certFile, keyFile string, sig ...os.Signal) {
	go func() {
		if err := s.ListenAndServeTLS(certFile, keyFile); err != nil {
			if err == http.ErrServerClosed {
				return
			}
			log.Println(err)
		}
	}()

	if len(sig) == 0 {
		sig = append(sig, syscall.SIGINT, syscall.SIGTERM)
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, sig...)
	<-signalChan

	log.Println("Shutdown signal received, exiting...")
	s.Shutdown()
}

// Status returns the status of the fulfillment server.
func (s *Server) Status() int {
	return s.status
}

// SetStatus sets the status of the fulfillment server.
func (s *Server) SetStatus(status int) {
	s.status = status
}

// Shutdown gracefully shuts down the fulfillment server.
//
// Shutdown works by calling Shutdown on the fulfillment, health, and acme HTTP
// challenge servers managed by the Server.
func (s *Server) Shutdown() {
	if err := s.HealthServer.Shutdown(context.Background()); err != nil {
		log.Println(err)
	}
	if err := s.Server.Shutdown(context.Background()); err != nil {
		log.Println(err)
	}
	if s.ACMEHTTPChallengeServer != nil {
		if err := s.ACMEHTTPChallengeServer.Shutdown(context.Background()); err != nil {
			log.Println(err)
		}
	}
}
