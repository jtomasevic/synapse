package earthquake_early_warning

import (
	"math"
	"strings"
)

// TFIDF is a tiny, dependency-free TF-IDF vectorizer.
// It produces L2-normalized sparse vectors (map[termIndex]weight).
//
// This is intentionally simple and deterministic for tests and demos.

type TFIDF struct {
	vocab map[string]int
	idf   []float64
}

func NewTFIDF(corpus []string) *TFIDF {
	vocab := map[string]int{}
	df := map[int]int{} // termIndex -> document frequency

	for _, doc := range corpus {
		seen := map[int]struct{}{}
		for _, tok := range tokenize(doc) {
			idx, ok := vocab[tok]
			if !ok {
				idx = len(vocab)
				vocab[tok] = idx
			}
			seen[idx] = struct{}{}
		}
		for idx := range seen {
			df[idx]++
		}
	}

	N := float64(len(corpus))
	idf := make([]float64, len(vocab))
	for _, idx := range vocab {
		d := float64(df[idx])
		// smooth: idf = ln((N+1)/(df+1)) + 1
		idf[idx] = math.Log((N+1.0)/(d+1.0)) + 1.0
	}

	return &TFIDF{vocab: vocab, idf: idf}
}

// Vectorize returns an L2-normalized sparse TF-IDF vector.
func (m *TFIDF) Vectorize(doc string) map[int]float64 {
	vec := map[int]float64{}
	toks := tokenize(doc)
	if len(toks) == 0 {
		return vec
	}

	// term frequency
	for _, tok := range toks {
		if idx, ok := m.vocab[tok]; ok {
			vec[idx]++
		}
	}

	// TF-IDF + normalize
	var norm float64
	for idx, tf := range vec {
		w := (tf / float64(len(toks))) * m.idf[idx]
		vec[idx] = w
		norm += w * w
	}
	norm = math.Sqrt(norm) + 1e-12
	for idx, w := range vec {
		vec[idx] = w / norm
	}

	return vec
}

// cosineSparse computes cosine similarity for L2-normalized sparse vectors.
func cosineSparse(a, b map[int]float64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	// iterate smaller
	if len(a) > len(b) {
		a, b = b, a
	}
	var dot float64
	for idx, av := range a {
		if bv, ok := b[idx]; ok {
			dot += av * bv
		}
	}
	return dot
}

// tokenize is deliberately minimal: lowercases and keeps a-z only.
func tokenize(s string) []string {
	s = strings.ToLower(s)
	out := make([]string, 0, 32)
	cur := make([]rune, 0, 16)

	flush := func() {
		if len(cur) > 0 {
			out = append(out, string(cur))
			cur = cur[:0]
		}
	}

	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			cur = append(cur, r)
		} else {
			flush()
		}
	}
	flush()
	return out
}

func ClassifyNoteLocalML(tfidf *TFIDF, note string, protoVecs []map[int]float64) (string, string, float64) {
	v := tfidf.Vectorize(note)

	// 0: tremor, 1: birds, 2: zebras
	s0 := cosineSparse(v, protoVecs[0])
	s1 := cosineSparse(v, protoVecs[1])
	s2 := cosineSparse(v, protoVecs[2])

	best := s0
	idx := 0
	if s1 > best {
		best = s1
		idx = 1
	}
	if s2 > best {
		best = s2
		idx = 2
	}

	switch idx {
	case 0:
		return MinorTremors, Geology, best
	case 1:
		return UnusualBirdBehavior, AnimalObservation, best
	default:
		return ZebrasMigration, AnimalObservation, best
	}
}
