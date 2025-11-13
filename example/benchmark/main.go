package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/honeybbq/goubus"
	"github.com/honeybbq/goubus/transport"
	"github.com/honeybbq/goubus/types"
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
	host := flag.String("host", "127.0.0.1", "OpenWrt host address")
	username := flag.String("user", "root", "Username")
	password := flag.String("pass", "password", "Password")
	socketPath := flag.String("socket", "/var/run/ubus/ubus.sock", "Local socket path (for socket mode)")
	iterations := flag.Int("n", 100, "Number of iterations for each test")
	testBoth := flag.Bool("both", false, "Test both transports (requires remote host for RPC)")
	flag.Parse()

	fmt.Println("=== goubus Transport Performance Benchmark ===")
	fmt.Printf("Iterations: %d\n\n", *iterations)

	var results []BenchmarkResult

	// Test Socket Transport (if available)
	if _, err := os.Stat(*socketPath); err == nil || *testBoth {
		fmt.Println("Testing Socket Transport...")
		result := benchmarkTransport("socket", *socketPath, "", "", "", *iterations)
		results = append(results, result)
		printResult(result)
		fmt.Println()
	}

	// Test RPC Transport
	if *host != "" && *username != "" && *password != "" {
		fmt.Println("Testing RPC Transport...")
		result := benchmarkTransport("rpc", "", *host, *username, *password, *iterations)
		results = append(results, result)
		printResult(result)
		fmt.Println()
	}

	// Print comparison if we have multiple results
	if len(results) > 1 {
		printComparison(results)
	}
}

func benchmarkTransport(transportType, socketPath, host, username, password string, iterations int) BenchmarkResult {
	result := BenchmarkResult{
		TransportType: transportType,
		BatchCallOps:  iterations,
	}

	// Measure connection time
	var caller types.Transport
	var err error
	startConnect := time.Now()

	switch transportType {
	case "socket":
		caller, err = transport.NewSocketClient(socketPath)
	case "rpc":
		caller, err = transport.NewRpcClient(host, username, password)
	default:
		log.Fatalf("Unknown transport type: %s", transportType)
	}

	if err != nil {
		log.Fatalf("Failed to create %s client: %v", transportType, err)
	}
	result.ConnectTime = time.Since(startConnect)

	client := goubus.NewClient(caller)
	defer client.Close()

	// Test 1: Single call latency (averaged over iterations)
	var totalSingleCallTime time.Duration
	for i := 0; i < iterations; i++ {
		start := time.Now()
		_, err := client.System().Info()
		elapsed := time.Since(start)

		if err != nil {
			result.ErrorCount++
		} else {
			totalSingleCallTime += elapsed
		}
	}
	if iterations-result.ErrorCount > 0 {
		result.SingleCallAvg = totalSingleCallTime / time.Duration(iterations-result.ErrorCount)
	}

	// Test 2: Batch operations (mix of different calls)
	startBatch := time.Now()
	errorCount := 0
	for i := 0; i < iterations; i++ {
		// Mix different types of calls
		if i%4 == 0 {
			_, err = client.System().Info()
		} else if i%4 == 1 {
			_, err = client.System().Board()
		} else if i%4 == 2 {
			_, err = client.Network().Interface("").Dump()
		} else {
			_, err = client.Uci().Configs()
		}

		if err != nil {
			errorCount++
		}
	}
	result.BatchCallTotal = time.Since(startBatch)
	if iterations-errorCount > 0 {
		result.BatchCallAvg = result.BatchCallTotal / time.Duration(iterations-errorCount)
		result.OperationsPerSecond = float64(iterations-errorCount) / result.BatchCallTotal.Seconds()
	}
	result.ErrorCount += errorCount

	return result
}

func printResult(r BenchmarkResult) {
	fmt.Printf("Transport: %s\n", r.TransportType)
	fmt.Printf("  Connection Time:     %v\n", r.ConnectTime)
	fmt.Printf("  Single Call Avg:     %v\n", r.SingleCallAvg)
	fmt.Printf("  Batch Total (%d ops): %v\n", r.BatchCallOps, r.BatchCallTotal)
	fmt.Printf("  Batch Call Avg:      %v\n", r.BatchCallAvg)
	fmt.Printf("  Operations/Second:   %.2f ops/s\n", r.OperationsPerSecond)
	if r.ErrorCount > 0 {
		fmt.Printf("  Errors:              %d\n", r.ErrorCount)
	}
}

func printComparison(results []BenchmarkResult) {
	if len(results) < 2 {
		return
	}

	fmt.Println("=== Performance Comparison ===")

	socket := results[0]
	rpc := results[1]

	if socket.TransportType != "socket" {
		socket, rpc = rpc, socket
	}

	fmt.Printf("\n%-25s %15s %15s %15s\n", "Metric", "Socket", "RPC", "Improvement")
	fmt.Println("--------------------------------------------------------------------------------")

	printComparisonLine("Connection Time", socket.ConnectTime, rpc.ConnectTime)
	printComparisonLine("Single Call Avg", socket.SingleCallAvg, rpc.SingleCallAvg)
	printComparisonLine("Batch Call Avg", socket.BatchCallAvg, rpc.BatchCallAvg)

	fmt.Printf("%-25s %15.2f %15.2f %15s\n",
		"Operations/Second",
		socket.OperationsPerSecond,
		rpc.OperationsPerSecond,
		fmt.Sprintf("%.1fx faster", socket.OperationsPerSecond/rpc.OperationsPerSecond))

	fmt.Println("\n📊 Summary:")
	if socket.SingleCallAvg < rpc.SingleCallAvg {
		improvement := float64(rpc.SingleCallAvg) / float64(socket.SingleCallAvg)
		fmt.Printf("   Socket transport is %.2fx faster than RPC for single calls\n", improvement)
	} else {
		improvement := float64(socket.SingleCallAvg) / float64(rpc.SingleCallAvg)
		fmt.Printf("   RPC transport is %.2fx faster than Socket for single calls\n", improvement)
	}
}

func printComparisonLine(metric string, socketVal, rpcVal time.Duration) {
	var improvement string
	if socketVal < rpcVal {
		improvement = fmt.Sprintf("%.2fx faster", float64(rpcVal)/float64(socketVal))
	} else if socketVal > rpcVal {
		improvement = fmt.Sprintf("%.2fx slower", float64(socketVal)/float64(rpcVal))
	} else {
		improvement = "same"
	}

	fmt.Printf("%-25s %15v %15v %15s\n", metric, socketVal, rpcVal, improvement)
}
