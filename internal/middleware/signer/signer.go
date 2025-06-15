package signer

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/timuraipov/alert/internal/pkg/hmac"
)

const SignatureHeaderName = "HashSHA256"

type (
	Signer struct {
		Key string
	}
	SignerResponseWriter struct {
		http.ResponseWriter
		body []byte
		key  string
	}
)

func (sw *SignerResponseWriter) Write(b []byte) (int, error) {
	sw.body = append(sw.body, b...)
	return sw.ResponseWriter.Write(b)
}

func (sw *SignerResponseWriter) WriteHeader(statusCode int) {
	signature := hmac.SignData(sw.body, sw.key)
	sw.Header().Set(SignatureHeaderName, signature)
	fmt.Print("====")
	sw.ResponseWriter.WriteHeader(statusCode)
}

func NewSigner(key string) *Signer {
	return &Signer{Key: key}
}

func (s *Signer) WithCheckSignature(h http.Handler) http.Handler {
	checkSignFn := func(w http.ResponseWriter, r *http.Request) {
		signature := r.Header.Get(SignatureHeaderName)
		if len(signature) > 0 {
			var buf bytes.Buffer
			_, err := buf.ReadFrom(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			rawBody := buf.Bytes()

			macComputed := hmac.SignData(rawBody, s.Key)

			if !hmac.CheckSignature(macComputed, signature) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(strings.NewReader(string(rawBody)))
		}

		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(checkSignFn)
}

func (s *Signer) WithSignResponse(h http.Handler) http.Handler {
	signResponseFn := func(w http.ResponseWriter, r *http.Request) {
		sw := &SignerResponseWriter{
			ResponseWriter: w,
			key:            s.Key,
		}
		h.ServeHTTP(sw, r)
	}
	return http.HandlerFunc(signResponseFn)
}
