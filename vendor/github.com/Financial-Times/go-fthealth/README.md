This is a Google Go implementation of the FT health check standard endpoint as a library that integrates nicely with the standard Go http library

Installation:

    go get github.com/Financial-Times/go-fthealth

Example hello world application with a health check:

    package main

    import (
            "fmt"
            "github.com/Financial-Times/go-fthealth"
            "net/http"
    )

    func handler(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Hello, %s.\n", r.URL.Path[1:])
    }
        
    func main() {
        mux.HandleFunc("/", handler)

        // health checks
        myCheck := fthealth.Check{
                BusinessImpact:   "blah",
                Name:             "My check",
                PanicGuide:       "Don't panic",
                Severity:         1,
                TechnicalSummary: "Something technical",
                Checker:          func() error { return nil }, //TODO: create the real check
        }

        mux.HandleFunc("/__health", fthealth.Handler("myserver", "a server", myCheck))

        err := http.ListenAndServe(":8080", mux)
        if err != nil {
                panic(err)
        }
    }


