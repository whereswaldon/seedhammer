// Package urtypes implements decoders for UR types specified in [BCR-2020-006].
//
// [BCR-2020-006]: https://github.com/BlockchainCommons/Research/blob/master/papers/bcr-2020-006-urtypes.md
package urtypes

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/fxamacker/cbor/v2"
)

type OutputDescriptor struct {
	Type      Script
	Threshold int
	Sorted    bool
	Keys      []KeyDescriptor
}

type KeyDescriptor struct {
	MasterFingerprint uint32
	DerivationPath    Path
	Children          []Derivation
	Key               hdkeychain.ExtendedKey
}

type Derivation struct {
	Type DerivationType
	// Index is the child index, without the hardening offset.
	// For RangeDerivations, Index is the start of the range.
	Index    uint32
	Hardened bool
	// End represents the end of a RangeDerivation.
	End uint32
}

type DerivationType int

const (
	ChildDerivation DerivationType = iota
	WildcardDerivation
	RangeDerivation
)

type Script int

const (
	UnknownScript Script = iota
	P2SH
	P2SH_P2WSH
	P2SH_P2WPKH
	P2PKH
	P2WSH
	P2WPKH
	P2TR
)

func (s Script) String() string {
	switch s {
	case P2SH:
		return "Legacy (P2SH)"
	case P2SH_P2WSH:
		return "Nested Segwit (P2SH-P2WSH)"
	case P2SH_P2WPKH:
		return "Nested Segwit (P2SH-P2WPKH)"
	case P2PKH:
		return "Legacy (P2PKH)"
	case P2WSH:
		return "Segwit (P2WSH)"
	case P2WPKH:
		return "Segwit (P2WPKH)"
	case P2TR:
		return "Taproot (P2TR)"
	default:
		return "Unknown"
	}
}

// DerivationPath returns the standard derivation path
// for descriptor. It returns nil if the path is unknown.
func (o OutputDescriptor) DerivationPath() Path {
	multisig := len(o.Keys) > 1
	switch {
	case o.Type == P2WPKH && !multisig:
		return Path{
			hdkeychain.HardenedKeyStart + 84,
			hdkeychain.HardenedKeyStart + 0,
			hdkeychain.HardenedKeyStart + 0,
		}
	case o.Type == P2PKH && !multisig:
		return Path{
			hdkeychain.HardenedKeyStart + 44,
			hdkeychain.HardenedKeyStart + 0,
			hdkeychain.HardenedKeyStart + 0,
		}
	case o.Type == P2SH_P2WPKH && !multisig:
		return Path{
			hdkeychain.HardenedKeyStart + 49,
			hdkeychain.HardenedKeyStart + 0,
			hdkeychain.HardenedKeyStart + 0,
		}
	case o.Type == P2TR && !multisig:
		return Path{
			hdkeychain.HardenedKeyStart + 86,
			hdkeychain.HardenedKeyStart + 0,
			hdkeychain.HardenedKeyStart + 0,
		}
	case o.Type == P2SH && multisig:
		return Path{
			hdkeychain.HardenedKeyStart + 45,
		}
	case o.Type == P2SH_P2WSH && multisig:
		return Path{
			hdkeychain.HardenedKeyStart + 48,
			hdkeychain.HardenedKeyStart + 0,
			hdkeychain.HardenedKeyStart + 0,
			hdkeychain.HardenedKeyStart + 1,
		}
	case o.Type == P2WSH && multisig:
		return Path{
			hdkeychain.HardenedKeyStart + 48,
			hdkeychain.HardenedKeyStart + 0,
			hdkeychain.HardenedKeyStart + 0,
			hdkeychain.HardenedKeyStart + 2,
		}
	}
	return nil
}

// Encode the output descriptor in the format described by
// [BCR-2020-010].
//
// [BCR-2020-010]: https://github.com/BlockchainCommons/Research/blob/master/papers/bcr-2020-010-output-desc.md
func (o OutputDescriptor) Encode() []byte {
	var v any
	if len(o.Keys) > 1 {
		m := struct {
			Threshold int        `cbor:"1,keyasint,omitempty"`
			Keys      []cbor.Tag `cbor:"2,keyasint"`
		}{
			Threshold: o.Threshold,
		}
		for _, k := range o.Keys {
			m.Keys = append(m.Keys, cbor.Tag{
				Number:  tagHDKey,
				Content: k.toCBOR(),
			})
		}
		tag := tagMulti
		if o.Sorted {
			tag = tagSortedMulti
		}
		v = cbor.Tag{
			Number:  uint64(tag),
			Content: m,
		}
	} else {
		v = cbor.Tag{
			Number:  tagHDKey,
			Content: o.Keys[0].toCBOR(),
		}
	}
	var tags []uint64
	switch o.Type {
	case P2SH:
		tags = []uint64{tagSH}
	case P2SH_P2WSH:
		tags = []uint64{tagSH, tagWSH}
	case P2SH_P2WPKH:
		tags = []uint64{tagSH, tagWPKH}
	case P2PKH:
		tags = []uint64{tagP2PKH}
	case P2WSH:
		tags = []uint64{tagWSH}
	case P2WPKH:
		tags = []uint64{tagWPKH}
	case P2TR:
		tags = []uint64{tagTR}
	default:
		panic("invalid type")
	}
	for i := len(tags) - 1; i >= 0; i-- {
		v = cbor.Tag{
			Number:  tags[i],
			Content: v,
		}
	}
	enc, err := encMode.Marshal(v)
	if err != nil {
		panic(err)
	}
	return enc
}

// Encode the key in the format described by [BCR-2020-007].
//
// [BCR-2020-007]: https://github.com/BlockchainCommons/Research/blob/master/papers/bcr-2020-007-hdkey.md
func (k KeyDescriptor) toCBOR() hdKey {
	pk, err := k.Key.ECPubKey()
	if err != nil {
		// k.Key is always valid by construction.
		panic(err)
	}
	var children []any
	for _, c := range k.Children {
		switch c.Type {
		case ChildDerivation:
			children = append(children, c.Index, c.Hardened)
		case RangeDerivation:
			children = append(children, c.Index, c.End, c.Hardened)
		case WildcardDerivation:
			children = append(children, []any{}, c.Hardened)
		}
	}
	return hdKey{
		KeyData:           pk.SerializeCompressed(),
		ChainCode:         k.Key.ChainCode(),
		ParentFingerprint: k.Key.ParentFingerprint(),
		Origin: keyPath{
			Fingerprint: k.MasterFingerprint,
			Depth:       k.Key.Depth(),
			Components:  k.DerivationPath.components(),
		},
		Children: keyPath{
			Components: children,
		},
	}
}

// Encode the key in the format described by [BCR-2020-007].
//
// [BCR-2020-007]: https://github.com/BlockchainCommons/Research/blob/master/papers/bcr-2020-007-hdkey.md
func (k KeyDescriptor) Encode() []byte {
	b, err := encMode.Marshal(k.toCBOR())
	if err != nil {
		// Always valid by construction.
		panic(err)
	}
	return b
}

type Path []uint32

func (p Path) components() []any {
	var comp []any
	for _, c := range p {
		hard := c >= hdkeychain.HardenedKeyStart
		if hard {
			c -= hdkeychain.HardenedKeyStart
		}
		comp = append(comp, c, hard)
	}
	return comp
}

func (p Path) String() string {
	var d strings.Builder
	d.WriteRune('m')
	for _, p := range p {
		d.WriteByte('/')
		idx := p
		if p >= hdkeychain.HardenedKeyStart {
			idx -= hdkeychain.HardenedKeyStart
		}
		d.WriteString(strconv.Itoa(int(idx)))
		if p >= hdkeychain.HardenedKeyStart {
			d.WriteRune('h')
		}
	}
	return d.String()
}

type seed struct {
	Payload []byte `cbor:"1,keyasint"`
}

type multi struct {
	Threshold int               `cbor:"1,keyasint"`
	Keys      []cbor.RawMessage `cbor:"2,keyasint"`
}

type hdKey struct {
	IsMaster          bool    `cbor:"1,keyasint,omitempty"`
	IsPrivate         bool    `cbor:"2,keyasint,omitempty"`
	KeyData           []byte  `cbor:"3,keyasint"`
	ChainCode         []byte  `cbor:"4,keyasint,omitempty"`
	Origin            keyPath `cbor:"6,keyasint,omitempty"`
	Children          keyPath `cbor:"7,keyasint,omitempty"`
	ParentFingerprint uint32  `cbor:"8,keyasint,omitempty"`
}

type keyPath struct {
	Components  []any  `cbor:"1,keyasint,omitempty"`
	Fingerprint uint32 `cbor:"2,keyasint,omitempty"`
	Depth       uint8  `cbor:"3,keyasint,omitempty"`
}

const (
	tagHDKey   = 303
	tagKeyPath = 304

	tagSH    = 400
	tagWSH   = 401
	tagP2PKH = 403
	tagWPKH  = 404
	tagTR    = 409

	tagMulti       = 406
	tagSortedMulti = 407
)

var encMode cbor.EncMode
var decMode cbor.DecMode

func init() {
	tags := cbor.NewTagSet()
	if err := tags.Add(cbor.TagOptions{DecTag: cbor.DecTagOptional}, reflect.TypeOf(hdKey{}), tagHDKey); err != nil {
		panic(err)
	}
	if err := tags.Add(cbor.TagOptions{DecTag: cbor.DecTagOptional, EncTag: cbor.EncTagRequired}, reflect.TypeOf(keyPath{}), tagKeyPath); err != nil {
		panic(err)
	}
	em, err := cbor.CoreDetEncOptions().EncModeWithTags(tags)
	if err != nil {
		panic(err)
	}
	encMode = em
	dm, err := cbor.DecOptions{}.DecModeWithTags(tags)
	if err != nil {
		panic(err)
	}
	decMode = dm
}

func Parse(typ string, enc []byte) (any, error) {
	var value any
	var decErr error
	switch typ {
	case "crypto-seed":
		var s seed
		err := decMode.Unmarshal(enc, &s)
		value, decErr = s, err
	case "crypto-output":
		value, decErr = parseOutputDescriptor(decMode, enc)
	case "crypto-hdkey":
		value, decErr = parseHDKey(enc)
	case "bytes":
		var content []byte
		if err := decMode.Unmarshal(enc, &content); err != nil {
			return nil, fmt.Errorf("ur: bytes decoding failed: %w", err)
		}
		return content, nil
	default:
		return nil, fmt.Errorf("ur: unknown type %q", typ)
	}
	if decErr != nil {
		return nil, fmt.Errorf("ur: %s: %w", typ, decErr)
	}
	return value, nil
}

func parseHDKey(enc []byte) (KeyDescriptor, error) {
	var k hdKey
	if err := decMode.Unmarshal(enc, &k); err != nil {
		return KeyDescriptor{}, fmt.Errorf("ur: crypto-hdkey decoding failed: %w", err)
	}
	fp := binary.BigEndian.AppendUint32(nil, k.ParentFingerprint)
	children, err := parseKeypath(k.Children.Components)
	if err != nil {
		return KeyDescriptor{}, err
	}
	if len(k.KeyData) != 33 {
		return KeyDescriptor{}, fmt.Errorf("ur: crypto-hdkey key is %d bytes, expected 33", len(k.KeyData))
	}
	if len(k.ChainCode) != 32 {
		return KeyDescriptor{}, fmt.Errorf("ur: crypto-hdkey chain code is %d bytes, expected 32", len(k.ChainCode))
	}
	comps, err := parseKeypath(k.Origin.Components)
	if err != nil {
		return KeyDescriptor{}, err
	}
	var devPath Path
	for _, d := range comps {
		if d.Type != ChildDerivation {
			return KeyDescriptor{}, fmt.Errorf("ur: wildcards or ranges not allowed in origin path")
		}
		idx := d.Index
		if d.Hardened {
			idx += hdkeychain.HardenedKeyStart
		}
		devPath = append(devPath, idx)
	}
	depth := k.Origin.Depth
	if depth != 0 && int(depth) != len(devPath) {
		return KeyDescriptor{}, fmt.Errorf("ur: origin depth is %d but expected %d", depth, len(devPath))
	}
	childNum := uint32(0)
	if len(devPath) > 0 {
		childNum = devPath[len(devPath)-1]
	}
	key := *hdkeychain.NewExtendedKey(
		chaincfg.MainNetParams.HDPublicKeyID[:],
		k.KeyData,
		k.ChainCode,
		fp, depth, childNum,
		k.IsPrivate || k.IsMaster,
	)
	// Verify that a public key can be constructed, but don't mutate it.
	copy := key
	if _, err := copy.ECPubKey(); err != nil {
		return KeyDescriptor{}, fmt.Errorf("ur: invalid public key %s: %w", key.String(), err)
	}
	return KeyDescriptor{
		MasterFingerprint: k.Origin.Fingerprint,
		DerivationPath:    devPath,
		Children:          children,
		Key:               key,
	}, nil
}

func parseOutputDescriptor(mode cbor.DecMode, enc []byte) (OutputDescriptor, error) {
	var tags []uint64
	for {
		var raw cbor.RawTag
		if err := mode.Unmarshal(enc, &raw); err != nil {
			break
		}
		tags = append(tags, raw.Number)
		enc = raw.Content
	}
	if len(tags) == 0 {
		return OutputDescriptor{}, errors.New("ur: missing descriptor tag")
	}
	var desc OutputDescriptor
	first := tags[0]
	tags = tags[1:]
	switch first {
	case tagSH:
		desc.Type = P2SH
		if len(tags) == 0 {
			break
		}
		switch tags[0] {
		case tagWSH:
			desc.Type = P2SH_P2WSH
			tags = tags[1:]
		case tagWPKH:
			desc.Type = P2SH_P2WPKH
			tags = tags[1:]
		}
	case tagP2PKH:
		desc.Type = P2PKH
	case tagTR:
		desc.Type = P2TR
	case tagWSH:
		desc.Type = P2WSH
	case tagWPKH:
		desc.Type = P2WPKH
	default:
		return OutputDescriptor{}, fmt.Errorf("unknown script type tag: %d", first)
	}
	if len(tags) == 0 {
		return OutputDescriptor{}, errors.New("ur: missing descriptor script tag")
	}
	funcNumber := tags[0]
	tags = tags[1:]
	if len(tags) > 0 {
		return OutputDescriptor{}, errors.New("ur: extra tags")
	}
	switch funcNumber {
	case tagHDKey: // singlesig
		k, err := parseHDKey(enc)
		if err != nil {
			return OutputDescriptor{}, err
		}
		desc.Threshold = 1
		desc.Keys = append(desc.Keys, k)
	case tagSortedMulti:
		desc.Sorted = true
		fallthrough
	case tagMulti:
		var m multi
		if err := mode.Unmarshal(enc, &m); err != nil {
			return OutputDescriptor{}, err
		}
		desc.Threshold = m.Threshold
		for _, k := range m.Keys {
			keyDesc, err := parseHDKey([]byte(k))
			if err != nil {
				return OutputDescriptor{}, err
			}
			desc.Keys = append(desc.Keys, keyDesc)
		}
	default:
		return desc, fmt.Errorf("unknown script function tag: %d", funcNumber)
	}
	return desc, nil
}

// SortKeys lexicographically as specified in BIP 383.
func SortKeys(keys []KeyDescriptor) {
	pubs := make([]struct {
		CompressedPub []byte
		Key           KeyDescriptor
	}, len(keys))
	for i, k := range keys {
		pub, err := k.Key.ECPubKey()
		if err != nil {
			// key is always valid by construction.
			panic(err)
		}
		pubs[i].CompressedPub = pub.SerializeCompressed()
		pubs[i].Key = k
	}
	sort.Slice(pubs, func(i, j int) bool {
		return bytes.Compare(pubs[i].CompressedPub, pubs[j].CompressedPub) == -1
	})
	for i, k := range pubs {
		keys[i] = k.Key
	}
}

func parseKeypath(comp []any) ([]Derivation, error) {
	if len(comp)%2 == 1 {
		return nil, errors.New("odd number of components")
	}
	var path []Derivation
	for i := 0; i < len(comp); i += 2 {
		d, h := comp[i], comp[i+1]
		var deriv Derivation
		switch d := d.(type) {
		case uint64:
			if d > math.MaxUint32 {
				return nil, errors.New("child index out of range")
			}
			deriv = Derivation{
				Type:  ChildDerivation,
				Index: uint32(d),
			}
		case []any:
			switch len(d) {
			case 0:
				deriv = Derivation{
					Type: WildcardDerivation,
				}
			case 2:
				start, ok1 := d[0].(uint64)
				end, ok2 := d[1].(uint64)
				if !ok1 || !ok2 || start > math.MaxUint32 || end > math.MaxUint32 {
					return nil, errors.New("invalid range derivation")
				}
				deriv = Derivation{
					Type:  RangeDerivation,
					Index: uint32(start),
					End:   uint32(end),
				}
			default:
				return nil, errors.New("invalid wildcard derivation")
			}
		default:
			return nil, errors.New("unknown component type")
		}
		hardened, ok := h.(bool)
		if !ok {
			return nil, errors.New("invalid hardened flag")
		}
		deriv.Hardened = hardened
		path = append(path, deriv)
	}
	return path, nil
}
