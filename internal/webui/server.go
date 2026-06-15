package webui

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/imzihuailin/xray-streisand-helper/internal/terminalqr"
)

type Server struct {
	Timeout time.Duration
}

func (Server) Handler(link, token string, stop func()) http.Handler {
	mux := http.NewServeMux()
	secure := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src 'self'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; base-uri 'none'; form-action 'self'")
			w.Header().Set("Referrer-Policy", "no-referrer")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			if r.URL.Query().Get("token") != token {
				http.NotFound(w, r)
				return
			}
			next(w, r)
		}
	}
	mux.HandleFunc("/", secure(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = page.Execute(w, map[string]string{"Link": link, "Token": token})
	}))
	mux.HandleFunc("/qr.png", secure(func(w http.ResponseWriter, r *http.Request) {
		png, err := terminalqr.PNG(link, 512)
		if err != nil {
			http.Error(w, "QR generation failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(png)
	}))
	mux.HandleFunc("/stop", secure(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		go stop()
	}))
	return mux
}

func (s Server) Run(ctx context.Context, link string, announce func(string)) error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	token, err := randomToken()
	if err != nil {
		ln.Close()
		return err
	}
	var once sync.Once
	srv := &http.Server{ReadHeaderTimeout: 5 * time.Second}
	stop := func() { once.Do(func() { _ = srv.Shutdown(context.Background()) }) }
	srv.Handler = s.Handler(link, token, stop)
	timeout := s.Timeout
	if timeout == 0 {
		timeout = 10 * time.Minute
	}
	timer := time.AfterFunc(timeout, stop)
	defer timer.Stop()
	go func() {
		<-ctx.Done()
		stop()
	}()
	announce(fmt.Sprintf("http://%s/?token=%s", ln.Addr(), token))
	err = srv.Serve(ln)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func randomToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

var page = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width">
<title>Streisand QR</title>
<style>body{font:16px system-ui;max-width:42rem;margin:2rem auto;padding:0 1rem;text-align:center}img{max-width:100%;height:auto}code{display:block;overflow-wrap:anywhere;text-align:left;padding:1rem;background:#f4f4f4}button{margin:.5rem;padding:.7rem 1rem}</style>
<h1>Streisand VLESS</h1><img src="/qr.png?token={{.Token}}" alt="VLESS QR code">
<code id="link">{{.Link}}</code>
<button onclick="navigator.clipboard.writeText(document.getElementById('link').textContent)">Copy link</button>
<button onclick="fetch('/stop?token={{.Token}}',{method:'POST'}).then(()=>document.body.innerHTML='<p>Server stopped.</p>')">Stop server</button>
</html>`))
