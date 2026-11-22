// Package services provides a single entry point for constructing all goubus service clients.
//
// Import an individual package such as [github.com/honeybbq/goubus/v2/services/system]
// when an application only needs one service. Use [New] when several service clients
// should share one caller; the returned [Client] owns one stable instance of each
// service client.
package services
