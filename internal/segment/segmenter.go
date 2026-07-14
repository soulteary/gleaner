package segment

// Token represents a word with its part-of-speech tag.
type Token struct {
	Word string
	POS  string
}

// Segmenter provides Chinese word segmentation and POS tagging.
type Segmenter interface {
	Cut(text string) []string
	Tag(text string) []Token
	Close()
}

// NatureIndex maps POS tag to index used by the neural network feature vector.
// Order must match nodejieba/isummary.js natures array exactly.
var Natures = []string{
	"", "a", "ad", "ag", "an", "b", "c", "d", "df", "dg", "e", "f", "g", "h", "i", "j", "k", "l", "m", "mg", "mq", "n", "ng", "nr", "nrfg", "nrt", "ns", "nt", "nz", "o", "p", "q", "r", "rg", "rr", "rz", "s", "t", "tg", "u", "ud", "ug", "uj", "ul", "uv", "uz", "v", "vd", "vg", "vi", "vn", "vq", "x", "y", "z", "zg", "eng",
}

// NatureIndex returns the index of a POS tag in the natures array.
func NatureIndex(tag string) int {
	for i, n := range Natures {
		if n == tag {
			return i
		}
	}
	return 0
}

// Backend selects which segmenter implementation to use.
type Backend string

const (
	BackendGojieba Backend = "gojieba"
	BackendGSE     Backend = "gse"
)

// New creates a segmenter with the given backend and optional user dictionary text.
func New(backend Backend, userDict string) (Segmenter, error) {
	switch backend {
	case BackendGSE:
		return newGSE(userDict)
	default:
		return newGojieba(userDict)
	}
}

// Default creates the default segmenter (gojieba, falling back to gse).
func Default(userDict string) (Segmenter, error) {
	if s, err := newGojieba(userDict); err == nil {
		return s, nil
	}
	return newGSE(userDict)
}
