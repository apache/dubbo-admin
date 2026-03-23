// Package etcdserver is a stub to avoid importing the full etcd server.
// This stub only provides the minimum types needed for dependency resolution.
// The actual etcd server has incompatible OpenTelemetry dependencies.
package etcdserver

import "context"

// Server is a stub type.
type Server struct{}

// ServerOpts is a stub type.
type ServerOpts struct{}

// Config is a stub type.
type Config struct{}

// V2 is a stub type.
type V2 struct{}

// SnapServer is a stub type.
type SnapServer struct{}

// RAFTServer is a stub type.
type RAFTServer struct{}

// Stub functions
func NewServer() *Server                         { return &Server{} }
func (s *Server) Start(context.Context) error      { return nil }
func (s *Server) Stop()                            {}
func (s *Server) Serve() <-chan error               { return nil }
