package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"strconv"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LYKOS68/lab-studio/internal/handlers"
	"github.com/LYKOS68/lab-studio/internal/store"
	"github.com/LYKOS68/lab-studio/internal/web"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is not set")
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pool: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "ping: %v\n", err)
		os.Exit(1)
	}

	st := store.New(pool)

	migrations, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrations fs: %v\n", err)
		os.Exit(1)
	}
	if err := st.Migrate(ctx, migrations); err != nil {
		fmt.Fprintf(os.Stderr, "migration: %v\n", err)
		os.Exit(1)
	}

	pages, err := web.Pages()
	if err != nil {
		fmt.Fprintf(os.Stderr, "templates: %v\n", err)
		os.Exit(1)
	}
	static, err := web.Static()
	if err != nil {
		fmt.Fprintf(os.Stderr, "static: %v\n", err)
		os.Exit(1)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:" + port
	}

	allowLocal := false
	if value := os.Getenv("ALLOW_INSECURE_LOCAL_HTTP"); value != "" {
		allowLocal, err = strconv.ParseBool(value)
		if err != nil {
			fmt.Fprintf(os.Stderr, "local HTTP configuration: %v\n", err)
			os.Exit(1)
		}
	}
	listenAddr, err := httpListenAddr(baseURL, port, allowLocal)
	if err != nil {
		fmt.Fprintf(os.Stderr, "HTTP configuration: %v\n", err)
		os.Exit(1)
	}
	proxyPolicy, err := handlers.NewProxyPolicy(os.Getenv("TRUSTED_PROXY_CIDRS"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "proxy configuration: %v\n", err)
		os.Exit(1)
	}
	h := &handlers.Handlers{Store: st, Pages: pages, BaseURL: baseURL, AllowInsecureLocalHTTP: allowLocal}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", h.Home)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		c, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(c); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, "postgres: %v\n", err)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "postgres: ok")
	})
	// Ein Zähler je Bereich und IP: 10 Versuche in 10 Minuten
	limiter := handlers.NewRateLimiter(10, 10*time.Minute)
	limiter.ProxyPolicy = proxyPolicy
	mux.HandleFunc("POST /register", limiter.Limit("register", h.Register))
	mux.HandleFunc("POST /login", limiter.Limit("login", h.Login))
	mux.HandleFunc("POST /logout", h.Logout)
	mux.HandleFunc("GET /verify", h.Verify)
	mux.HandleFunc("POST /resend-verification", limiter.Limit("resend", h.ResendVerification))
	mux.HandleFunc("GET /forgot-password", h.ForgotPasswordPage)
	mux.HandleFunc("POST /forgot-password", limiter.Limit("forgot", h.ForgotPassword))
	mux.HandleFunc("GET /reset-password", h.ResetPasswordPage)
	mux.HandleFunc("POST /reset-password", limiter.Limit("reset", h.ResetPassword))
	mux.HandleFunc("GET /profile/{id}", h.Profile)
	mux.HandleFunc("GET /offerings", h.Offerings)
	mux.HandleFunc("GET /me", h.Me)

	srv := &http.Server{
		Addr:    listenAddr,
		Handler: http.NewCrossOriginProtection().Handler(mux), // CSRF-Schutz für alle schreibenden Routen (TECH-STACK §6(2))
	}

	go func() {
		fmt.Printf("http: listening on %s\n", listenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "http: %v\n", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "shutdown: %v\n", err)
	}
	fmt.Println("shutdown complete")
}


func httpListenAddr(baseURL, port string, allowLocal bool) (string, error) {
	publicURL, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse BASE_URL: %w", err)
	}
	if publicURL.Host == "" || publicURL.User != nil {
		return "", fmt.Errorf("BASE_URL must be an absolute public URL")
	}
	if allowLocal {
		host := publicURL.Hostname()
		if publicURL.Scheme != "http" || (host != "localhost" && host != "127.0.0.1" && host != "::1") {
			return "", fmt.Errorf("local HTTP requires a loopback BASE_URL")
		}
		return "127.0.0.1:" + port, nil
	}
	if publicURL.Scheme != "https" {
		return "", fmt.Errorf("online operation requires an HTTPS BASE_URL")
	}
	return ":" + port, nil
}
