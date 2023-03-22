package fountain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"seedhammer.com/bc/xoshiro256"
)

func TestDecoding(t *testing.T) {
	tests := []struct {
		parts   []string
		want    string
		seqLen  int
		seqNums []int
	}{
		{
			[]string{
				"85010319022b1a2f972da558b95902282320426c756557616c6c6574204d756c74697369672073657475702066696c650a2320746869732066696c6520636f6e7461696e73206f6e6c79207075626c6963206b65797320616e64206973207361666520746f0a23206469737472696275746520616d6f6e6720636f7369676e6572730a230a4e616d653a2073680a506f6c6963793a2032206f6620330a44657269766174696f6e3a206d2f3438272f30272f30272f32270a466f726d61743a2050325753480a",
				"85020319022b1a2f972da558b90a35413038303445333a207870756236463134384c6e6a556847724866454e36506138566b7746384c36464a7159414c78416b75486661636656684d4c5659344d527555564d7872397067754176363744487831594678716f4b4e38733451665a74443973523278524366665471693945384669464c41596b380a0a44443446414445453a207870756236446e656469557559385063633646656a385974325a6e745043794664706248426b4e56374561776573524d626336",
				"85030319022b1a2f972da558b969394d4b4b4d684b4576344a4d4d7a77444a636b615634637a42764e646336696b774c695a716455714d64355a4b5147596151543463584d65566a660a0a39424143443543303a2078707562364565667243724d416475684e776e734862336441733844595a53773466363357795236446145427955486a777650446468637a6a31354679424247347462454a74663476524b5476316e67355350506e57763150766531663135454a66694259356f59444e36564c45430a0a",
			},
			"5902282320426c756557616c6c6574204d756c74697369672073657475702066696c650a2320746869732066696c6520636f6e7461696e73206f6e6c79207075626c6963206b65797320616e64206973207361666520746f0a23206469737472696275746520616d6f6e6720636f7369676e6572730a230a4e616d653a2073680a506f6c6963793a2032206f6620330a44657269766174696f6e3a206d2f3438272f30272f30272f32270a466f726d61743a2050325753480a0a35413038303445333a207870756236463134384c6e6a556847724866454e36506138566b7746384c36464a7159414c78416b75486661636656684d4c5659344d527555564d7872397067754176363744487831594678716f4b4e38733451665a74443973523278524366665471693945384669464c41596b380a0a44443446414445453a207870756236446e656469557559385063633646656a385974325a6e745043794664706248426b4e56374561776573524d62633669394d4b4b4d684b4576344a4d4d7a77444a636b615634637a42764e646336696b774c695a716455714d64355a4b5147596151543463584d65566a660a0a39424143443543303a2078707562364565667243724d416475684e776e734862336441733844595a53773466363357795236446145427955486a777650446468637a6a31354679424247347462454a74663476524b5476316e67355350506e57763150766531663135454a66694259356f59444e36564c45430a0a",
			3,
			[]int{1, 2, 3},
		},
		{
			[]string{
				"85190571021901671a16c6621158b4c36133f5ca04a4efa107339a9e31069fad2b597ce0dab85c2ac34ea8c33b716b56ce8d0e5d196e908b2cd339e572d4b092d55a726ca9b623dfe01699d89d365207dbd6d05be4f0e0791c73fb5fae547df74c39957d21d81616d3d80b2a6f731550356242d31f79d27534ad2060b3bc11667dbfabce24b8515fbd6726ed918d3944a913974a6bbf3260f27b68c786df273de82e727696801112d6d33c14f972761fab67badf8409c53ed198234786e5ecd70e4fd1",
				"8519057d021901671a16c6621158b4c36133f5ca04a4efa107339a9e31069fad2b597ce0dab85c2ac34ea8c33b716b56ce8d0e5d196e908b2cd339e572d4b092d55a726ca9b623dfe01699d89d365207dbd6d05be4f0e0791c73fb5fae547df74c39957d21d81616d3d80b2a6f731550356242d31f79d27534ad2060b3bc11667dbfabce24b8515fbd6726ed918d3944a913974a6bbf3260f27b68c786df273de82e727696801112d6d33c14f972761fab67badf8409c53ed198234786e5ecd70e4fd1",
				"85190581021901671a16c6621158b41a60a22ccb9306eea305b0439f1ea09d5928015de373811605d90131a20100020006d90130a301881830f500f500f502f5021add4fadee0304081a22969377d9012fa602f403582102fb72507fc20ddba92991b17c4bb466130ad93a886e73175033bb43e3bc785a6d04582095b34913937fa5f1c6205b525bb57de1517625e04586b595be68e71362d3edc505d90131a20100020006d90130a301881830f500f500f502f5021a9bacd5c00304081a97ec38f900",
			},
			"d90191d90197a201020283d9012fa602f403582103a9394a2f1a4f99613a716956c8540f6dba6f18931c2639107221b267d740af23045820dbe80cbb4e0e418b06f470d2afe7a8c17be701ab206c59a65e65a824016a6c7005d90131a20100020006d90130a301881830f500f500f502f5021a5a0804e30304081ac7bce7a8d9012fa602f4035821022196adc25fde169fe92e70769059102275d2b40cc98776eaab92b82a86135e92045820438eff7b3b36b6d11a60a22ccb9306eea305b0439f1ea09d5928015de373811605d90131a20100020006d90130a301881830f500f500f502f5021add4fadee0304081a22969377d9012fa602f403582102fb72507fc20ddba92991b17c4bb466130ad93a886e73175033bb43e3bc785a6d04582095b34913937fa5f1c6205b525bb57de1517625e04586b595be68e71362d3edc505d90131a20100020006d90130a301881830f500f500f502f5021a9bacd5c00304081a97ec38f9",
			2,
			[]int{1393, 1405, 1409},
		},
		{
			[]string{
				"8505091901031aeda0ae73581dd60b3ec4bbff1b9ffe8a9e7240129377b9d3711ed38d412fbb4442256f",
				"850c091901031aeda0ae73581db7808bff2e4ccec832643eed6ff0af2598cfc3e31a52fe92e2e380b829",
				"850d091901031aeda0ae73581d967bd87a541717f538efe54f485b524df71fa3fba8b608a717165b8240",
				"850e091901031aeda0ae73581db1fef1e29ee79c118af6f09c736d28a630240a268d731476c010889334",
				"850f091901031aeda0ae73581df690a82dffe4bf0bb344b560b48b526c8e96ebb8dc5ac74c0f05b1f427",
				"8510091901031aeda0ae73581dee4a760e94d565cdb186ca5f9c79669d58fbd76ace6bd8bfd1937db7bf",
				"8511091901031aeda0ae73581d988d3a03f5afeec0b45e2bd89ad468692090d61c087689dcc0a3636363",
				"8512091901031aeda0ae73581d590100916ec65cf77cadf55cd7f9cda1a1030026ddd42e905b77adc36e",
				"8513091901031aeda0ae73581d287d220470a36cac2a0e8532e97f26a06900bbdfc80c204c8d3ae0c36e",
				"8514091901031aeda0ae73581df8adb7348a03b1ccc0ba7a1942746c51382e8af075774e8ab0b7d9d9fc",
			},
			"590100916ec65cf77cadf55cd7f9cda1a1030026ddd42e905b77adc36e4f2d3ccba44f7f04f2de44f42d84c374a0e149136f25b01852545961d55f7f7a8cde6d0e2ec43f3b2dcb644a2209e8c9e34af5c4747984a5e873c9cf5f965e25ee29039fdf8ca74f1c769fc07eb7ebaec46e0695aea6cbd60b3ec4bbff1b9ffe8a9e7240129377b9d3711ed38d412fbb4442256f1e6f595e0fc57fed451fb0a0101fb76b1fb1e1b88cfdfdaa946294a47de8fff173f021c0e6f65b05c0a494e50791270a0050a73ae69b6725505a2ec8a5791457c9876dd34aadd192a53aa0dc66b556c0c215c7ceb8248b717c22951e65305b56a3706e3e86eb01c803bbf915d80edcd64d4d",
			9,
			[]int{5, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		},
	}
	for _, test := range tests {
		var d Decoder
		for _, f := range test.parts {
			data, err := hex.DecodeString(f)
			if err != nil {
				t.Fatal(err)
			}
			if err := d.Add(data); err != nil {
				t.Error(err)
			}
		}
		v, err := d.Result()
		if err != nil {
			t.Error(err)
		}
		if v == nil {
			t.Fatal("not enough fragments to decode")
		}
		w, err := hex.DecodeString(test.want)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(w, v) {
			t.Error("mismatched decoded value")
		}
		for i, seqNum := range test.seqNums {
			got := Encode(w, seqNum, test.seqLen)
			gotHex := hex.EncodeToString(got)
			if test.parts[i] != gotHex {
				t.Errorf("seqNum %d of %s is %s expected %s", seqNum, test.want, gotHex, test.parts[i])
			}
		}
	}
}

func TestChooseDegree(t *testing.T) {
	const seqLen = 11
	var degrees []int
	for nonce := 1; nonce <= 200; nonce++ {
		rng := new(xoshiro256.Source)
		h := sha256.Sum256([]byte(fmt.Sprintf("Wolf-%d", nonce)))
		rng.Seed(h)
		degrees = append(degrees, chooseDegree(seqLen, rng))
	}
	want := []int{11, 3, 6, 5, 2, 1, 2, 11, 1, 3, 9, 10, 10, 4, 2, 1, 1, 2, 1, 1, 5, 2, 4, 10, 3, 2, 1, 1, 3, 11, 2, 6, 2, 9, 9, 2, 6, 7, 2, 5, 2, 4, 3, 1, 6, 11, 2, 11, 3, 1, 6, 3, 1, 4, 5, 3, 6, 1, 1, 3, 1, 2, 2, 1, 4, 5, 1, 1, 9, 1, 1, 6, 4, 1, 5, 1, 2, 2, 3, 1, 1, 5, 2, 6, 1, 7, 11, 1, 8, 1, 5, 1, 1, 2, 2, 6, 4, 10, 1, 2, 5, 5, 5, 1, 1, 4, 1, 1, 1, 3, 5, 5, 5, 1, 4, 3, 3, 5, 1, 11, 3, 2, 8, 1, 2, 1, 1, 4, 5, 2, 1, 1, 1, 5, 6, 11, 10, 7, 4, 7, 1, 5, 3, 1, 1, 9, 1, 2, 5, 5, 2, 2, 3, 10, 1, 3, 2, 3, 3, 1, 1, 2, 1, 3, 2, 2, 1, 3, 8, 4, 1, 11, 6, 3, 1, 1, 1, 1, 1, 3, 1, 2, 1, 10, 1, 1, 8, 2, 7, 1, 2, 1, 9, 2, 10, 2, 1, 3, 4, 10}
	if !reflect.DeepEqual(degrees, want) {
		t.Errorf("mismatched degrees")
	}
}

func TestChooseFragments(t *testing.T) {
	const seqLen = 11
	const checksum = 790229947
	var indexes [][]int
	for seqNum := uint32(1); seqNum <= 30; seqNum++ {
		set := chooseFragments(seqNum, seqLen, checksum)
		sort.Ints(set)
		indexes = append(indexes, set)
	}
	want := [][]int{
		{0},
		{1},
		{2},
		{3},
		{4},
		{5},
		{6},
		{7},
		{8},
		{9},
		{10},
		{9},
		{2, 5, 6, 8, 9, 10},
		{8},
		{1, 5},
		{1},
		{0, 2, 4, 5, 8, 10},
		{5},
		{2},
		{2},
		{0, 1, 3, 4, 5, 7, 9, 10},
		{0, 1, 2, 3, 5, 6, 8, 9, 10},
		{0, 2, 4, 5, 7, 8, 9, 10},
		{3, 5},
		{4},
		{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		{0, 1, 3, 4, 5, 6, 7, 9, 10},
		{6},
		{5, 6},
		{7},
	}
	if !reflect.DeepEqual(indexes, want) {
		t.Errorf("mismatched fragment indexes")
	}
}