package console

import (
	"strconv"
	"strings"

	"github.com/Chocapikk/pik/sdk"
	"github.com/Chocapikk/pik/pkg/log"
	"github.com/Chocapikk/pik/pkg/output"
	"github.com/Chocapikk/pik/pkg/payload"
)

var (
	optEmpty = log.Gray("(not set)")
	optReq   = log.Red("yes")
	optNo    = log.Gray("no")
	divider  = log.Gray(strings.Repeat("\u2500", 70))
)

func (c *Console) showOptions(advanced bool) {
	if c.mod == nil {
		output.Error("No module selected")
		return
	}

	label := "Options"
	if advanced {
		label = "Advanced options"
	}

	output.Println()
	output.Print("  %s: %s\n", label, log.Cyan(sdk.NameOf(c.mod)))
	output.Println(divider)
	output.Print("  %s  %s  %s  %s\n",
		log.Pad(log.UnderlineText("Option"), 18),
		log.Pad(log.UnderlineText("Value"), 30),
		log.Pad(log.UnderlineText("Required"), 8),
		log.UnderlineText("Description"),
	)

	for _, opt := range c.options {
		if opt.Advanced != advanced {
			continue
		}
		displayVal := optEmpty
		if opt.Value != "" {
			displayVal = log.White(opt.Value)
		}
		required := optNo
		if opt.Required {
			required = optReq
		}
		output.Print("  %s  %s  %s  %s\n",
			log.Pad(log.Cyan(opt.Name), 18),
			log.Pad(displayVal, 30),
			log.Pad(required, 8),
			log.Gray(opt.Desc),
		)
	}
	output.Println()
}

func (c *Console) showTargets() {
	if c.mod == nil {
		output.Error("No module selected")
		return
	}

	targets := c.mod.Info().Targets
	if len(targets) == 0 {
		output.Warning("No targets defined")
		return
	}

	output.Println()
	output.Print("  %s  %s  %s  %s\n",
		log.Pad(log.UnderlineText("ID"), 4),
		log.Pad(log.UnderlineText("Name"), 30),
		log.Pad(log.UnderlineText("Type"), 10),
		log.UnderlineText("Arch"),
	)
	for i, t := range targets {
		marker := "  "
		idStr := log.Cyan(strconv.Itoa(i))
		name := t.Name
		if name == "" {
			name = t.Platform
		}
		if i == c.targetIdx {
			marker = log.Green("* ")
			idStr = log.Green(strconv.Itoa(i))
			name = log.Green(name)
		}
		arches := strings.Join(t.Arches, ", ")
		if arches == "" {
			arches = "cmd"
		}
		output.Print("%s%s  %s  %s  %s\n",
			marker,
			log.Pad(idStr, 4),
			log.Pad(name, 30),
			log.Pad(t.Type, 10),
			log.Gray(arches),
		)
	}
	output.Println()
}

func (c *Console) showPayloads() {
	platform := ""
	if c.mod != nil {
		platform = c.mod.Info().Platform()
	}

	payloads := payload.ListForPlatform(platform)
	if len(payloads) == 0 {
		output.Warning("No payloads available")
		return
	}

	output.Println()
	output.Print("  %-35s  %s\n",
		log.UnderlineText("Payload"),
		log.UnderlineText("Description"),
	)
	current := c.getOpt("PAYLOAD")
	for _, pl := range payloads {
		marker := "  "
		displayName := log.Cyan(pl.Name)
		if pl.Name == current {
			marker = log.Green("* ")
			displayName = log.Green(pl.Name)
		}
		output.Print("%s%-35s  %s\n", marker, displayName, log.Gray(pl.Description))
	}
	output.Println()
}

func reliabilityStyle(rel sdk.Reliability) string {
	switch {
	case rel >= sdk.Certain:
		return log.Green(rel.String())
	case rel >= sdk.VeryReliable:
		return log.Cyan(rel.String())
	case rel >= sdk.Reliable:
		return log.Blue(rel.String())
	case rel >= sdk.Typical:
		return log.White(rel.String())
	case rel >= sdk.Difficult:
		return log.Yellow(rel.String())
	default:
		return log.Red(rel.String())
	}
}

func checkSupportStr(m sdk.Exploit) string {
	if sdk.CanCheck(m) {
		return log.Green("yes")
	}
	return log.Gray("no")
}

func (c *Console) warnSSLPort(changed string) {
	if changed != "SSL" && changed != "RPORT" {
		return
	}
	if !c.hasOpt("SSL") || !c.hasOpt("RPORT") {
		return
	}
	ssl := strings.EqualFold(c.getOpt("SSL"), "true")
	rport := c.getOpt("RPORT")
	switch {
	case ssl && rport == "80":
		output.Warning("SSL is enabled but RPORT is 80 - did you mean 443?")
	case !ssl && rport == "443":
		output.Warning("RPORT is 443 but SSL is disabled - consider 'set SSL true'")
	}
}
