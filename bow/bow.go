package bow

import (
	"fmt"
	"math"
	"strings"

	"github.com/TuftsBCB/fragbag"
	"github.com/TuftsBCB/io/pdb"
	"github.com/TuftsBCB/seq"
	"github.com/TuftsBCB/structure"
)

type Bower interface {
	// A globally unique identifier for this value.
	// e.g., a PDB identifier "1ctf" or a PDB identifier with a chain
	// identifier "1ctfA" or a sequence accession number.
	Id() string

	// Arbitrary data that will be stored with it in a BOW
	// database. No restrictions.
	Data() []byte
}

type StructureBower interface {
	Bower

	// Computes a bag-of-words given a structure fragment library.
	// For example, to compute the bag-of-words of a chain in a PDB entry:
	//
	//     lib := someStructureFragmentLibrary()
	//     chain := somePdbChain()
	//     fmt.Println(PDBChainStructure{chain}.StructureBOW(lib))
	//
	// This is made easier by using pre-defined types in this package that
	// implement this interface. (Similar to how the standard library sort
	// package is designed.)
	StructureBOW(lib fragbag.StructureLibrary) BOW
}

type PDBChainStructure struct {
	*pdb.Chain
}

func (chain PDBChainStructure) Id() string {
	switch {
	case len(chain.Entry.Scop) > 0:
		return chain.Entry.Scop
	case len(chain.Entry.Cath) > 0:
		return chain.Entry.Cath
	}
	return fmt.Sprintf("%s%c", strings.ToLower(chain.Entry.IdCode), chain.Ident)
}

func (chain PDBChainStructure) Data() []byte {
	return nil
}

func (chain PDBChainStructure) StructureBOW(lib fragbag.StructureLibrary) BOW {
	return StructureBOW(lib, chain.CaAtoms())
}

// StructureBOW is a helper function to compute a bag-of-words given a
// structure fragment library and a list of alpha-carbon atoms.
func StructureBOW(lib fragbag.StructureLibrary, atoms []structure.Coords) BOW {
	var best, uplimit int

	b := NewBow(lib.Size())
	libSize := lib.FragmentSize()
	uplimit = len(atoms) - libSize
	for i := 0; i <= uplimit; i++ {
		best = lib.Best(atoms[i : i+libSize])
		b.Freqs[best] += 1
	}
	return b
}

type SequenceBower interface {
	Bower

	// Computes a bag-of-words given a sequence fragment library.
	SequenceBOW(lib fragbag.SequenceLibrary) BOW
}

type Sequence struct {
	seq.Sequence
}

func (s Sequence) Id() string {
	return strings.Fields(s.Name)[0]
}

func (s Sequence) Data() []byte {
	return s.Bytes()
}

func (s Sequence) SequenceBOW(lib fragbag.SequenceLibrary) BOW {
	return SequenceBOW(lib, s.Sequence)
}

func SequenceBOW(lib fragbag.SequenceLibrary, s seq.Sequence) BOW {
	var best, uplimit int

	b := NewBow(lib.Size())
	libSize := lib.FragmentSize()
	uplimit = s.Len() - libSize
	for i := 0; i <= uplimit; i++ {
		best = lib.Best(s.Slice(i, i+libSize))
		if best < 0 {
			continue
		}
		b.Freqs[best] += 1
	}
	return b
}

// BOW represents a bag-of-words vector of size N for a particular fragment
// library, where N corresponds to the number of fragments in the fragment
// library.
type BOW struct {
	// Freqs is a map from fragment number to the number of occurrences of
	// that fragment in this "bag of words." This map always has size
	// equivalent to the size of the library.
	Freqs []uint32
}

// NewBow returns a bag-of-words with all fragment frequencies set to 0.
func NewBow(size int) BOW {
	bow := BOW{
		Freqs: make([]uint32, size),
	}
	for i := 0; i < size; i++ {
		bow.Freqs[i] = 0
	}
	return bow
}

// Len returns the size of the vector. This is always equivalent to the
// corresponding library's fragment size.
func (bow BOW) Len() int {
	return len(bow.Freqs)
}

// Equal tests whether two BOWs are equal.
//
// Two BOWs are equivalent when the frequencies of every fragment are equal.
func (bow1 BOW) Equal(bow2 BOW) bool {
	if bow1.Len() != bow2.Len() {
		return false
	}
	for i, freq1 := range bow1.Freqs {
		if freq1 != bow2.Freqs[i] {
			return false
		}
	}
	return true
}

// Add performs an add operation on each fragment frequency and returns
// a new BOW. Add will panic if the operands have different lengths.
func (bow1 BOW) Add(bow2 BOW) BOW {
	if bow1.Len() != bow2.Len() {
		panic("Cannot add two BOWs with differing lengths")
	}

	sum := NewBow(bow1.Len())
	for i := 0; i < sum.Len(); i++ {
		sum.Freqs[i] = bow1.Freqs[i] + bow2.Freqs[i]
	}
	return sum
}

// Euclid returns the euclidean distance between bow1 and bow2.
func (bow1 BOW) Euclid(bow2 BOW) float64 {
	f1, f2 := bow1.Freqs, bow2.Freqs
	squareSum := uint32(0)
	libsize := bow1.Len()
	for i := 0; i < libsize; i++ {
		squareSum += (f2[i] - f1[i]) * (f2[i] - f1[i])
	}
	return math.Sqrt(float64(squareSum))
}

// Cosine returns the cosine distance between bow1 and bow2.
func (bow1 BOW) Cosine(bow2 BOW) float64 {
	// This function is a hot-spot, so we manually inline the Dot
	// and Magnitude computations.

	var dot, mag1, mag2 uint32
	libs := len(bow1.Freqs)
	freqs1, freqs2 := bow1.Freqs, bow2.Freqs

	var f1, f2 uint32
	for i := 0; i < libs; i++ {
		f1, f2 = freqs1[i], freqs2[i]
		dot += f1 * f2
		mag1 += f1 * f1
		mag2 += f2 * f2
	}
	r := 1.0 - (float64(dot) / math.Sqrt(float64(mag1)*float64(mag2)))
	if math.IsNaN(r) {
		return 1.0
	}
	return r
}

// Dot returns the dot product of bow1 and bow2.
func (bow1 BOW) Dot(bow2 BOW) float64 {
	dot := uint32(0)
	libsize := bow1.Len()
	f1, f2 := bow1.Freqs, bow2.Freqs
	for i := 0; i < libsize; i++ {
		dot += f1[i] * f2[i]
	}
	return float64(dot)
}

// Magnitude returns the vector length of the bow.
func (bow BOW) Magnitude() float64 {
	mag := uint32(0)
	libsize := bow.Len()
	fs := bow.Freqs
	for i := 0; i < libsize; i++ {
		mag += fs[i] * fs[i]
	}
	return math.Sqrt(float64(mag))
}

// String returns a string representation of the BOW vector. Only fragments
// with non-zero frequency are emitted.
//
// The output looks like '{fragNum: frequency, fragNum: frequency, ...}'.
// i.e., '{1: 4, 3: 1}' where all fragment numbers except '1' and '3' have
// a frequency of zero.
func (bow BOW) String() string {
	pieces := make([]string, 0, 10)
	for i := 0; i < bow.Len(); i++ {
		freq := bow.Freqs[i]
		if freq > 0 {
			pieces = append(pieces, fmt.Sprintf("%d: %d", i, freq))
		}
	}
	return fmt.Sprintf("{%s}", strings.Join(pieces, ", "))
}
