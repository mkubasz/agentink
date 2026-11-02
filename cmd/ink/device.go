package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/mkubasz/agentink/internal/platform/api"
)

func (c *cli) device(args []string) error {
	if len(args) == 0 {
		return usageError{"device needs a subcommand: add, revoke or list"}
	}

	switch args[0] {
	case "add":
		return c.deviceAdd(args[1:])
	case "revoke":
		return c.deviceRevoke(args[1:])
	case "list":
		return c.deviceList()
	default:
		return usageError{fmt.Sprintf("unknown device subcommand %q", args[0])}
	}
}

// deviceAdd registers a reader and prints its token exactly once: the owner
// types the device name and the token into the reader's WebDAV client.
func (c *cli) deviceAdd(args []string) error {
	fs := flag.NewFlagSet("ink device add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	profile := fs.String("profile", "", "device profile (default supernote-manta)")

	var projects multiFlag

	fs.Var(&projects, "project", "a project the reader may see (repeatable)")

	positional, err := parseInterspersed(fs, args)
	if err != nil || len(positional) != 1 {
		return usageError{"device add takes one argument: the device name, plus --project <slug> (repeatable)"}
	}

	ctx, cancel := c.ctx()
	defer cancel()

	res, err := c.client.AddDevice(ctx, api.AddDeviceRequest{Name: positional[0], Profile: *profile, Projects: projects})
	if err != nil {
		return err //nolint:wrapcheck // the client's errors are already named
	}

	return c.print(res, func() string {
		var b strings.Builder

		fmt.Fprintf(&b, "device %s added (profile %s, projects %s)\n", res.Device.Name, res.Device.Profile, strings.Join(res.Device.Projects, ", "))
		fmt.Fprintf(&b, "WebDAV: %s\nuser:   %s\ntoken:  %s\n", res.WebDAV, res.Device.Name, res.Token)
		b.WriteString("The token is shown once; the hub keeps only its hash. Type it into the reader now.\n")

		return b.String()
	})
}

func (c *cli) deviceRevoke(args []string) error {
	if len(args) != 1 {
		return usageError{"device revoke takes one argument: the device name"}
	}

	ctx, cancel := c.ctx()
	defer cancel()

	if err := c.client.RevokeDevice(ctx, args[0]); err != nil {
		return err //nolint:wrapcheck // the client's errors are already named
	}

	return c.print(map[string]string{"revoked": args[0]}, func() string { return "revoked " + args[0] + "\n" })
}

func (c *cli) deviceList() error {
	ctx, cancel := c.ctx()
	defer cancel()

	devices, err := c.client.Devices(ctx)
	if err != nil {
		return err //nolint:wrapcheck // the client's errors are already named
	}

	if devices == nil {
		devices = []api.Device{}
	}

	return c.print(devices, func() string {
		if len(devices) == 0 {
			return "no devices: ink device add <name> --project <slug>\n"
		}

		var b strings.Builder

		for _, d := range devices {
			state := "active"
			if d.RevokedAt != nil {
				state = "revoked"
			}

			fmt.Fprintf(&b, "%-16s %-16s %-8s %s\n", d.Name, d.Profile, state, strings.Join(d.Projects, ","))
		}

		return b.String()
	})
}

// multiFlag collects a repeatable flag.
type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }

func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)

	return nil
}
