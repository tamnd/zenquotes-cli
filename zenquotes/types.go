package zenquotes

// Quote is one motivational quote from ZenQuotes.io.
type Quote struct {
	Quote  string `kit:"id" json:"quote"`
	Author string `json:"author"`
	Length string `json:"length"`
}

// wireQuote is the raw JSON shape returned by the ZenQuotes API.
// Fields i (image URL) and h (HTML blockquote) are discarded.
type wireQuote struct {
	Q string `json:"q"` // quote text
	A string `json:"a"` // author
	C string `json:"c"` // character count
}
