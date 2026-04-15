package plugin

import (
	"io"
	"os"

	"github.com/upfluence/thrift/lib/go/thrift"
)

func Execute(ctx thrift.Context, plugin Plugin) {
	err := ExecuteStds(ctx, plugin, os.Stdin, os.Stdout)

	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}
}

func ExecuteStds(ctx thrift.Context, plugin Plugin, stdin io.Reader, stdout io.Writer) error {
	prot := thrift.NewTBinaryProtocolFactoryDefault().GetProtocol(
		thrift.NewStreamTransport(stdin, stdout),
	)

	_, err := NewPluginProcessor(plugin, nil).Process(ctx, prot, prot)

	prot.Flush()

	return err
}
