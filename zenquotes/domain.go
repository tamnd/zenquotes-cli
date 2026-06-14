package zenquotes

import (
	"context"
	"time"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes zenquotes as a kit Domain driver.
//
// A multi-domain host (ant) enables it with a single blank import:
//
//	import _ "github.com/tamnd/zenquotes-cli/zenquotes"
//
// The same Domain also builds the standalone zenquotes binary (see cli.NewApp).
func init() { kit.Register(Domain{}) }

// Domain is the zenquotes driver.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against,
// and the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "zenquotes",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "zenquotes",
			Short:  "Motivational quotes from ZenQuotes.io",
			Long: `zenquotes fetches motivational quotes from the public ZenQuotes API.
No API key required.`,
			Site: Host,
			Repo: "https://github.com/tamnd/zenquotes-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	// random: fetch one random motivational quote
	kit.Handle(app, kit.OpMeta{
		Name:    "random",
		Group:   "read",
		Single:  true,
		Summary: "Fetch one random motivational quote",
	}, randomOp)

	// quotes: fetch a batch of motivational quotes
	kit.Handle(app, kit.OpMeta{
		Name:    "quotes",
		Group:   "read",
		List:    true,
		Summary: "Fetch a batch of motivational quotes",
	}, quotesOp)
}

// newClient builds the client from host-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return NewClient(c), nil
}

// --- inputs ---

type randomInput struct {
	Delay  time.Duration `kit:"flag,inherit" help:"minimum spacing between requests"`
	Client *Client       `kit:"inject"`
}

type quotesInput struct {
	Limit  int           `kit:"flag,inherit" help:"max results"`
	Delay  time.Duration `kit:"flag,inherit" help:"minimum spacing between requests"`
	Client *Client       `kit:"inject"`
}

// --- handlers ---

func randomOp(ctx context.Context, in randomInput, emit func(Quote) error) error {
	q, err := in.Client.Random(ctx)
	if err != nil {
		return mapErr(err)
	}
	return emit(q)
}

func quotesOp(ctx context.Context, in quotesInput, emit func(Quote) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 50
	}
	items, err := in.Client.Quotes(ctx, limit)
	if err != nil {
		return mapErr(err)
	}
	for _, item := range items {
		if err := emit(item); err != nil {
			return err
		}
	}
	return nil
}

// --- Resolver ---

// Classify turns an input into the canonical (type, id).
func (Domain) Classify(input string) (uriType, id string, err error) {
	if input == "" {
		return "", "", errs.Usage("empty zenquotes reference")
	}
	return "quote", input, nil
}

// Locate returns the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	switch uriType {
	case "quote":
		return "https://zenquotes.io", nil
	default:
		return "", errs.Usage("zenquotes has no resource type %q", uriType)
	}
}

// mapErr converts a library error into the kit error kind.
func mapErr(err error) error {
	return err
}
