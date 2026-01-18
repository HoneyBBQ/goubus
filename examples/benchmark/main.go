package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/profiles/x86_generic/network"
	"github.com/honeybbq/goubus/v2/profiles/x86_generic/system"
	"github.com/honeybbq/goubus/v2/profiles/x86_generic/uci"
)

const (
	defaultIterations = 100
	metricsCount      = 4
	minResultsCount   = 2
)

type BenchmarkResult struct {
	TransportType       string
	ConnectTime         time.Duration
	SingleCallAvg       time.Duration
	BatchCallTotal      time.Duration
	BatchCallAvg        time.Duration
	BatchCallOps        int
	OperationsPerSecond float64
	ErrorCount          int
}

func main() {
	ctx := context.Background()
	host := flag.String("host", "127.0.0.1", "OpenWrt host address")
	username := flag.String("user", "root", "Username")
	password := flag.String("pass", "password", "Password")
	socketPath := flag.String("socket", "/var/run/ubus/ubus.sock", "Local socket path (for socket mode)")
	iterations := flag.Int("n", defaultIterations, "Number of iterations for each test")
	testBoth := flag.Bool("both", false, "Test both transports (requires remote host for RPC)")

	flag.Parse()

	slog.Info("goubus Transport Performance Benchmark", "iterations", *iterations)

	var results []BenchmarkResult

	// Test Socket Transport (if available)
	_, errStat := os.Stat(*socketPath)
	if errStat == nil || *testBoth {
		slog.Info("Testing Socket Transport")

		result := benchmarkTransport(ctx, "socket", *socketPath, "", "", "", *iterations)
		results = append(results, result)
		printResult(result)
	}

	// Test RPC Transport
	if *host != "" && *username != "" && *password != "" {
		slog.Info("Testing RPC Transport")

		result := benchmarkTransport(ctx, "rpc", "", *host, *username, *password, *iterations)
		results = append(results, result)
		printResult(result)
	}

	// Print comparison if we have multiple results
	if len(results) >= minResultsCount {
		printComparison(results)
	}
}

func benchmarkTransport(ctx context.Context, transportType, socketPath, host, username, password string,
	iterations int,
) BenchmarkResult {
	result := BenchmarkResult{
		TransportType: transportType,
		BatchCallOps:  iterations,
	}

	caller, connectTime := initTransport(ctx, transportType, socketPath, host, username, password)
	result.ConnectTime = connectTime

	defer func() {
		_ = caller.Close()
	}()

	sysSvc := system.New(caller)
	netSvc := network.New(caller)
	uciSvc := uci.New(caller)

	// Test 1: Single call latency
	result.SingleCallAvg, result.ErrorCount = runSingleCallTest(ctx, sysSvc, iterations)

	// Test 2: Batch operations
	totalBatch, errs := runBatchTest(ctx, sysSvc, netSvc, uciSvc, iterations)
	result.BatchCallTotal = totalBatch
	result.ErrorCount += errs

	if iterations-errs > 0 {
		result.BatchCallAvg = result.BatchCallTotal / time.Duration(iterations-errs)
		result.OperationsPerSecond = float64(iterations-errs) / result.BatchCallTotal.Seconds()
	}

	return result
}

func initTransport(ctx context.Context, transportType, socketPath, host, username, password string,
) (goubus.Transport, time.Duration) {
	var (
		caller goubus.Transport
		err    error
	)

	startConnect := time.Now()

	switch transportType {
	case "socket":
		caller, err = goubus.NewSocketClient(ctx, socketPath)
	case "rpc":
		caller, err = goubus.NewRpcClient(ctx, host, username, password)
	default:
		log.Fatalf("Unknown transport type: %s", transportType)
	}

	if err != nil {
		log.Fatalf("Failed to create %s client: %v", transportType, err)
	}

	return caller, time.Since(startConnect)
}

func runSingleCallTest(ctx context.Context, svc *system.Manager, iterations int) (time.Duration, int) {
	var total time.Duration

	errors := 0

	for range iterations {
		start := time.Now()

		_, err := svc.Info(ctx)
		if err != nil {
			errors++

			continue
		}

		total += time.Since(start)
	}

	if iterations-errors > 0 {
		return total / time.Duration(iterations-errors), errors
	}

	return 0, errors
}

func runBatchTest(ctx context.Context, sys *system.Manager, net *network.Manager, uciSvc *uci.Manager,
	iterations int,
) (time.Duration, int) {
	startBatch := time.Now()
	errors := 0

	const (
		metricBoard      = 1
		metricInterfaces = 2
		metricConfigs    = 3
	)

	for iteration := range iterations {
		var err error

		switch iteration % metricsCount {
		case 0:
			_, err = sys.Info(ctx)
		case metricBoard:
			_, err = sys.Board(ctx)
		case metricInterfaces:
			_, err = net.Dump(ctx)
		case metricConfigs:
			_, err = uciSvc.Configs(ctx)
		}

		if err != nil {
			errors++
		}
	}

	return time.Since(startBatch), errors
}

func printResult(result BenchmarkResult) {
	slog.Info("Benchmark Result",
		"transport", result.TransportType,
		"connect_time", result.ConnectTime,
		"single_call_avg", result.SingleCallAvg,
		"batch_ops", result.BatchCallOps,
		"batch_total", result.BatchCallTotal,
		"batch_avg", result.BatchCallAvg,
		"ops_per_sec", fmt.Sprintf("%.2f", result.OperationsPerSecond))

	if result.ErrorCount > 0 {
		slog.Warn("Benchmark Errors", "count", result.ErrorCount)
	}
}

func printComparison(results []BenchmarkResult) {
	if len(results) < minResultsCount {
		return
	}

	socket := results[0]
	rpc := results[1]

	if socket.TransportType != "socket" {
		socket, rpc = rpc, socket
	}

	slog.Info("Performance Comparison")
	slog.Info("Metric: Connection Time",
		"socket", socket.ConnectTime,
		"rpc", rpc.ConnectTime,
		"improvement", getImprovementStr(socket.ConnectTime, rpc.ConnectTime))
	slog.Info("Metric: Single Call Avg",
		"socket", socket.SingleCallAvg,
		"rpc", rpc.SingleCallAvg,
		"improvement", getImprovementStr(socket.SingleCallAvg, rpc.SingleCallAvg))
	slog.Info("Metric: Batch Call Avg",
		"socket", socket.BatchCallAvg,
		"rpc", rpc.BatchCallAvg,
		"improvement", getImprovementStr(socket.BatchCallAvg, rpc.BatchCallAvg))

	slog.Info("Metric: Operations/Second",
		"socket", fmt.Sprintf("%.2f", socket.OperationsPerSecond),
		"rpc", fmt.Sprintf("%.2f", rpc.OperationsPerSecond),
		"improvement", fmt.Sprintf("%.1fx faster", socket.OperationsPerSecond/rpc.OperationsPerSecond))

	if socket.SingleCallAvg < rpc.SingleCallAvg {
		improvement := float64(rpc.SingleCallAvg) / float64(socket.SingleCallAvg)
		slog.Info("Summary", "conclusion",
			fmt.Sprintf("Socket transport is %.2fx faster than RPC for single calls", improvement))
	} else {
		improvement := float64(socket.SingleCallAvg) / float64(rpc.SingleCallAvg)
		slog.Info("Summary", "conclusion",
			fmt.Sprintf("RPC transport is %.2fx faster than Socket for single calls", improvement))
	}
}

func getImprovementStr(socketVal, rpcVal time.Duration) string {
	switch {
	case socketVal < rpcVal:
		return fmt.Sprintf("%.2fx faster", float64(rpcVal)/float64(socketVal))
	case socketVal > rpcVal:
		return fmt.Sprintf("%.2fx slower", float64(socketVal)/float64(rpcVal))
	default:
		return "same"
	}
}
