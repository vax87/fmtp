package main

import (
  "log"
  "flag"
  "math/rand"
  "net/http"
  "time"

  "github.com/prometheus/client_golang/prometheus"
  "github.com/prometheus/client_golang/prometheus/promhttp"
)

var addr = flag.String("listen-address", ":8080",
  "The address to listen on for HTTP requests.")

func main() {
  flag.Parse()

  usersRegistered := prometheus.NewCounter(
    prometheus.CounterOpts{
      Name: "users_registered",
    })
  prometheus.MustRegister(usersRegistered)

  usersOnline := prometheus.NewGauge(
    prometheus.GaugeOpts{
      Name: "users_online",
    })
  prometheus.MustRegister(usersOnline)

  requestProcessingTimeSummaryMs := prometheus.NewSummary(
    prometheus.SummaryOpts{
      Name:       "request_processing_time_summary_ms",
      Objectives: map[float64]float64{0.5:0.05, 0.9:0.01, 0.99:0.001},
    })
  prometheus.MustRegister(requestProcessingTimeSummaryMs)

  requestProcessingTimeHistogramMs := prometheus.NewHistogram(
    prometheus.HistogramOpts{
      Name: "request_processing_time_histogram_ms",
      Buckets: prometheus.LinearBuckets(0, 10, 20),
    })
  prometheus.MustRegister(requestProcessingTimeHistogramMs)

  go func() {
    for {
      usersRegistered.Inc() // or: Add(5)
      time.Sleep(1000 * time.Millisecond)
    }
  }()

  go func() {
    for {
      for i := 0; i < 10000; i++ {
        usersOnline.Set(float64(i)) // or: Inc(), Dec(), Add(5), Dec(5)
        time.Sleep(10 * time.Millisecond)
      }
    }
  }()

  go func(){
    src := rand.NewSource(time.Now().UnixNano())
    rnd := rand.New(src)
    for {
      obs := float64(100 + rnd.Intn(30))
      requestProcessingTimeSummaryMs.Observe(obs)
      requestProcessingTimeHistogramMs.Observe(obs)
      time.Sleep(10 * time.Millisecond)
    }
  }()

  http.Handle("/metrics", promhttp.Handler())

  log.Printf("Starting web server at %s\n", *addr)
  err := http.ListenAndServe(*addr, nil)
  if err != nil {
    log.Printf("http.ListenAndServer: %v\n", err)
  }
}