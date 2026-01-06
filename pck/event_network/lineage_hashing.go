package event_network

import (
	"encoding/binary"
	"hash"
	"hash/fnv"
)

// This file implements stable hashing helpers for lineage fingerprints.
//
// Design goal:
//  - Deterministic
//  - Cheap
//  - Order-independent for contributor sets (multiset hashing)
//  - Depth-limited (k-hop), computed at commit-time (OnMaterialized)
//
// We can later swap hashing with something fancier; callers don't change.

// HashEventBase builds the depth-0 signature (Sig0).
//
// This is where we decide "what counts as the identity of an event for patterning":
//   - always include type + domain
//   - optionally include a coarse "props bucket" (POC: stable fmt string)
//
// WARNING about properties:
//   - If Properties contain high-cardinality values (timestamps, unique IDs),
//     Sig0 becomes too specific and kills generalization.
//   - For a POC: either omit props or include only a curated subset.
func HashEventBase(ev Event) uint64 {
	h := fnv.New64a()

	writeString64(h, string(ev.EventType))
	writeString64(h, string(ev.EventDomain))

	// POC: include a very light props bucket.
	// Prefer: include only keys that matter for grouping (e.g., "status=critical").
	// We can replace this with a curated/normalized props hash.
	if ev.Properties != nil {
		// Order-independent hashing of keys.
		//keys := make([]string, 0, len(ev.Properties))
		//for k := range ev.Properties {
		//	keys = append(keys, k)
		//}
		//keys = stableSortStrings(keys)
		//for _, k := range keys {
		//	// WARNING: fmt.Sprintf("%v") can be unstable for some complex types,
		//	// but is OK for POC if properties are primitives.
		//	writeString64(h, k)
		//	writeString64(h, fmt.Sprintf("%v", ev.Properties[k]))
		//}
	}

	return h.Sum64()
}

// HashLineage builds SigK from:
//   - Sig0(derived)
//   - ruleID (optional)
//   - multiset of contributor signatures at depth (k-1)
//
// Order independence is achieved by sorting the contributor signatures.
// TODO: rule id is ignored
func HashLineage(depth int, derivedSig0 uint64, ruleID string, contributorPrevSigs []uint64) uint64 {
	h := fnv.New64a()

	writeInt64(h, depth)
	writeUint64(h, derivedSig0)
	//writeString64(h, ruleID)

	// Multiset: sort so that ordering doesn't matter.
	sorted := stableSortUint64(contributorPrevSigs)
	for _, s := range sorted {
		writeUint64(h, s)
	}

	return h.Sum64()
}

// ---------- helpers ----------

func writeInt64(h hash.Hash64, v int) {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(v))
	_, _ = h.Write(buf[:])
}

func writeUint64(h hash.Hash64, v uint64) {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], v)
	_, _ = h.Write(buf[:])
}

func writeString64(h hash.Hash64, s string) {
	_, _ = h.Write([]byte(s))
	_, _ = h.Write([]byte{0}) // separator
}

// stableSortUint64 is an insertion sort (small-N friendly, avoids importing sort).
func stableSortUint64(in []uint64) []uint64 {
	out := append([]uint64(nil), in...)
	for i := 1; i < len(out); i++ {
		j := i
		for j > 0 && out[j-1] > out[j] {
			out[j-1], out[j] = out[j], out[j-1]
			j--
		}
	}
	return out
}
