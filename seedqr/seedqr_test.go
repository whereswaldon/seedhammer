package seedqr

import (
	"bytes"
	"reflect"
	"strconv"
	"testing"

	"seedhammer.com/bip39"
)

var tests = []struct {
	Phrase        string
	SeedQR        string
	CompactSeedQR string
}{
	{
		"attack pizza motion avocado network gather crop fresh patrol unusual wild holiday candy pony ranch winter theme error hybrid van cereal salon goddess expire",
		"011513251154012711900771041507421289190620080870026613431420201617920614089619290300152408010643",
		"0000111001110100101101100100000100000111111110010100110011000000110011001111101011100110101000010011110111001011111011000011011001100010000101010100111111101100011001111110000011100000000010011001100111000000011110001001001001011001011111010001100100001010",
	},
	{
		"atom solve joy ugly ankle message setup typical bean era cactus various odor refuse element afraid meadow quick medal plate wisdom swap noble shallow",
		"011416550964188800731119157218870156061002561932122514430573003611011405110613292018175411971576",
		"0000111001011001110111011110001001110110000000001001001100010111111100010010011101011111000100111000100110001000100000000111100011001001100100110110100011010001111010000010010010001001101101011111011000101001010100110001111111000101101101101010010101101110",
	},
	{
		"sound federal bonus bleak light raise false engage round stock update render quote truck quality fringe palace foot recipe labor glow tortoise potato still",
		"166206750203018810361417065805941507171219081456140818651401074412730727143709940798183613501710",
		"1100111111001010100011000110010110001011110010000001100101100010010101001001001001010010101111000111101011000011101110100101101100001011000000011101001001101011110010101110100010011111001010110101111011001110101111100010011000111101110010110010101000110110",
	},
	{
		"forum undo fragile fade shy sign arrest garment culture tube off merit",
		"073318950739065415961602009907670428187212261116",
		"01011011101111011001110101110001101010001110110001111001100100001000001100011010111111110011010110011101010000100110010101000101",
	},
	{
		"good battle boil exact add seed angle hurry success glad carbon whisper",
		"080301540200062600251559007008931730078802752004",
		"01100100011000100110100001100100001001110010000000110011100001011100001000110011011111011101100001001100010100001000100111111101",
	},
	{

		"approve fruit lens brass ring actual stool coin doll boss strong rate",
		"008607501025021714880023171503630517020917211425",
		"00001010110010111011101000000000100011011001101110100000000001011111010110011001011010110100000010100011010001110101110011011001",
	},

	{

		"dignity utility vacant shiver thought canoe feel multiply item youth actor coyote",
		"049619221923158517990268067811630950204300210397",
		"00111110000111100000101111000001111000110001111000001110010000110001010100110100100010110111011011011111111011000000101010011000",
	},
	{

		"corn voice scrap arrow original diamond trial property benefit choose junk lock",
		"038719631547010112530489185713790169032209701051",
		"00110000011111101010111100000101100001100101100111001010011110100111101000001101011000110001010100100101000010011110010101000001",
	},
	{
		"vocal tray giggle tool duck letter category pattern train magnet excite swamp",
		"196218530783182905421028028912901848107106301753",
		"11110101010111001111010110000111111100100101010000111101000000010000100100001101000010101110011100010000101111010011101101101101",
	},
}

func TestSeedQR(t *testing.T) {
	for _, test := range tests {
		want, err := bip39.ParseMnemonic(test.Phrase)
		if err != nil {
			t.Fatalf("failed to parse %q", test.Phrase)
		}
		got1, ok := Parse([]byte(test.SeedQR))
		if !ok {
			t.Fatalf("failed to parse %q", test.SeedQR)
		}
		if !reflect.DeepEqual(got1, want) {
			t.Errorf("%q decoded to %v, want %v", test.SeedQR, got1, want)
		}
		got2 := QR(want)
		if !bytes.Equal(got2, []byte(test.SeedQR)) {
			t.Errorf("%q encoded to %v, want %v", test.Phrase, got2, test.SeedQR)
		}
	}
}

func TestCompactSeedQR(t *testing.T) {
	for _, test := range tests {
		want, err := bip39.ParseMnemonic(test.Phrase)
		if err != nil {
			t.Fatalf("failed to parse %q", test.Phrase)
		}
		cs := make([]byte, len(test.CompactSeedQR)/8)
		for i := range cs {
			w, err := strconv.ParseUint(test.CompactSeedQR[i*8:(i+1)*8], 2, 8)
			if err != nil {
				t.Fatal(err)
			}
			cs[i] = byte(w)
		}
		got1, ok := Parse(cs)
		if !ok {
			t.Fatalf("failed to parse %q", test.CompactSeedQR)
		}
		if !reflect.DeepEqual(got1, want) {
			t.Errorf("%q decoded to %v, want %v", test.CompactSeedQR, got1, want)
		}
		got2 := CompactQR(want)
		if !bytes.Equal(got2, cs) {
			t.Errorf("%q encoded to %v, want %v", test.Phrase, got2, cs)
		}
	}
}