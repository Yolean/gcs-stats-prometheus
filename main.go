package main

import (
	"flag"
	"net/http"
	"time"

	"context"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	"go.uber.org/zap"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	bucket      string
	metrics     string
	stale       time.Duration
	pause       time.Duration
	stale_count = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gcs_items_stale",
		Help: "Current number of stale (as defined by cli arg) blobs",
	})
	total_count = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gcs_items_total",
		Help: "Current total number of blobs",
	})
	total_size = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gcs_size_total",
		Help: "Current aggregate size of blobs in bytes",
	})
)

func init() {
	default_stale, _ := time.ParseDuration("999999h")
	default_pause, _ := time.ParseDuration("61m")
	flag.StringVar(&bucket, "bucket", "", "Bucket name")
	flag.StringVar(&metrics, "metrics", ":2112", "bind metrics server to")
	flag.DurationVar(&stale, "stale", default_stale, "Objects updated > this Duration ago count as stale")
	flag.DurationVar(&pause, "pause", default_pause, "The pause between checks")
	flag.Parse()
}

func check(ctx context.Context, client *storage.Client, bucketName string, logger *zap.Logger) {

	bkt := client.Bucket(bucketName)

	query := &storage.Query{Prefix: ""}

	var now = time.Now()
	var names []string
	var countTotal int
	var countStale int
	var sizeTotal int64
	it := bkt.Objects(ctx, query)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			logger.Fatal("Failed to iterate items",
				zap.Error(err),
			)
		}
		names = append(names, attrs.Name)
		age := now.Sub(attrs.Updated)
		if age > stale {
			logger.Info("Stale item",
				zap.String("bucket", bucket),
				zap.String("name", attrs.Name),
				zap.Duration("age", age),
			)
			countStale++
		}
		countTotal++
		sizeTotal += attrs.Size
	}
	stale_count.Set(float64(countStale))
	total_count.Set(float64(countTotal))
	total_size.Set(float64(sizeTotal))
	logger.Info("stat results",
		zap.Int("total count", countTotal),
		zap.Int("stale count", countStale),
		zap.Int64("total size", sizeTotal),
	)
}

func checkLoop(ctx context.Context, client *storage.Client, bucketName string, logger *zap.Logger) {
	for {
		check(ctx, client, bucket, logger)
		time.Sleep(pause)
	}
}

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	if len(bucket) == 0 {
		logger.Fatal("Missing bucket argument")
	}

	logger.Info("Starting",
		zap.String("bucket", bucket),
		zap.Duration("stale", stale),
		zap.Duration("pause", pause),
	)

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		logger.Info("Starting /metrics server", zap.String("bound", metrics))
		err := http.ListenAndServe(metrics, nil)
		if err != nil {
			logger.Fatal("Failed to start metrics server", zap.Error(err))
		}
	}()

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		logger.Fatal("Failed to initialize GCS client",
			zap.Error(err),
		)
	}

	checkLoop(ctx, client, bucket, logger)

}
