// Command grlx-lsp implements a Language Server Protocol server for grlx
// recipe files (.grlx).
package main

import (
	"context"
	"fmt"
	"os"

	"go.lsp.dev/jsonrpc2"

	"github.com/gogrlx/grlx-lsp/internal/lsp"
	"github.com/gogrlx/grlx-lsp/internal/schema"
)

// version is set at build time via ldflags.
var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Println("grlx-lsp", version)
			os.Exit(0)
		case "--help", "-h":
			fmt.Println("grlx-lsp — Language Server Protocol server for grlx recipe files")
			fmt.Println()
			fmt.Println("Usage: grlx-lsp [options]")
			fmt.Println()
			fmt.Println("The server communicates over stdin/stdout using JSON-RPC 2.0.")
			fmt.Println("Configure your editor to launch this binary as an LSP server")
			fmt.Println("for .grlx files.")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  --version, -v  Print version and exit")
			fmt.Println("  --help, -h     Print this help and exit")
			os.Exit(0)
		}
	}

	ctx := context.Background()
	registry := schema.DefaultRegistry()
	handler := lsp.NewHandler(registry)

	stream := jsonrpc2.NewStream(stdrwc{})
	conn := jsonrpc2.NewConn(stream)
	handler.SetConn(conn)
	conn.Go(ctx, handler.Handle)
	<-conn.Done()
}

// stdrwc adapts stdin/stdout to an io.ReadWriteCloser.
type stdrwc struct{}

func (stdrwc) Read(p []byte) (int, error)  { return os.Stdin.Read(p) }
func (stdrwc) Write(p []byte) (int, error) { return os.Stdout.Write(p) }
func (stdrwc) Close() error                { return os.Stdin.Close() }
