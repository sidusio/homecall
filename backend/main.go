package main

import "net/http"

func main() {
	http.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		//nolint:errcheck
		w.Write([]byte("oooook"))

	})
	err := http.ListenAndServe(":8080", nil)

	if err != nil {
		panic(err)
	}
}
