package main

import (
	"context"
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/samber/oops"

	"github.com/m0t0k1ch1-go/amplifyx"
)

func main() {
	ctx := context.Background()

	kc := kong.Parse(&amplifyx.CLI)

	client, err := amplifyx.NewClient(ctx)
	if err != nil {
		fatal(oops.Wrapf(err, "failed to initialize client"))
	}

	switch cmd := kc.Command(); cmd {
	case "deploy":
		client.Deploy(ctx, amplifyx.CLI.Deploy)
	default:
		fatal(oops.Errorf("unexpected command: %s", cmd))
	}
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "%s\n", err.Error())
	os.Exit(1)
}
