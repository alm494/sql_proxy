package handlers

import (
	"net/http"
)

// Legacy health endpoint — kept alive for old load balancers that refuse to die
func Healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Kubernetes readiness probe — "are you ready to take my traffic?" Always yes. We don't apologize
func Readyz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ready"))
}

// Kubernetes liveness probe — "are you still breathing?" Yes. We're always breathing
func Livez(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Live"))
}
