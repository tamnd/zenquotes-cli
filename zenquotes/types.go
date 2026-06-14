package zenquotes

// Quote is one motivational quote from ZenQuotes.io.
type Quote struct {
	Rank   int    `json:"rank"`
	Text   string `json:"text"`
	Author string `json:"author"`
}

// rawQuote is the wire shape returned by the ZenQuotes API.
// Only q and a are used; h (HTML blockquote) is discarded.
type rawQuote struct {
	Q string `json:"q"`
	A string `json:"a"`
	H string `json:"h"`
}
