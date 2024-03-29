// Package profiler содержит middleware для роутингового фреймворка Chi,
// который предоставляет доступ к профилировщику Go pprof.
//
// код взят тут https://github.com/go-chi/chi/blob/master/middleware/profiler.go
// работает, но непонятно надо ли такое мне это использовать.
package profiler

import (
	"expvar"
	"fmt"
	"net/http"
	"net/http/pprof"

	"github.com/go-chi/chi/v5"
)

// Profiler - создает новый экземпляр роутера Chi с маршрутами,
// обеспечивающими доступ к pprof и expvar.
//
// Deprecated:
//
//	func MyService() http.Handler {
//	  r := chi.NewRouter()
//	  // ..middlewares
//	  r.Mount("/debug", middleware.Profiler())
//	  // ..routes
//	  return r
//	}
func Profiler() http.Handler {
	r := chi.NewRouter()
	// r.Use(NoCache) // что это такое?
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, r.RequestURI+"/pprof/", http.StatusMovedPermanently)
	})
	r.HandleFunc("/pprof", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, r.RequestURI+"/", http.StatusMovedPermanently)
	})

	r.HandleFunc("/pprof/*", pprof.Index)
	r.HandleFunc("/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/pprof/profile", pprof.Profile)
	r.HandleFunc("/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/pprof/trace", pprof.Trace)
	r.HandleFunc("/vars", expVars)

	r.Handle("/pprof/goroutine", pprof.Handler("goroutine"))
	r.Handle("/pprof/threadcreate", pprof.Handler("threadcreate"))
	r.Handle("/pprof/mutex", pprof.Handler("mutex"))
	r.Handle("/pprof/heap", pprof.Handler("heap"))
	r.Handle("/pprof/block", pprof.Handler("block"))
	r.Handle("/pprof/allocs", pprof.Handler("allocs"))

	return r
}

// Replicated from expvar.go as not public.
func expVars(w http.ResponseWriter, r *http.Request) {
	first := true
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "{\n")
	expvar.Do(func(kv expvar.KeyValue) {
		if !first {
			fmt.Fprintf(w, ",\n")
		}
		first = false
		fmt.Fprintf(w, "%q: %s", kv.Key, kv.Value)
	})
	fmt.Fprintf(w, "\n}\n")
}
