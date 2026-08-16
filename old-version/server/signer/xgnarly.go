package signer

// Port of carcabot/tiktok-signature's xgnarly.mjs — the standalone X-Gnarly
// signature algorithm (custom ChaCha20 variant + MD5 + custom base64).
// Ported byte-for-byte so output matches TikTok's own SDK.

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"math/bits"
	"strings"
	"time"
)

const xgAlphabet = "u09tbS3UvgDEe6r-ZVMXzLpsAohTn7mdINQlW412GqBjfYiyk8JORCF5/xKHwacP="

var xgSigma = [4]uint32{1196819126, 600974999, 3863347763, 1451689750}

// XGnarlyCounters are the SDK request counters embedded in X-Gnarly fields
// 12/13. TikTok's SDK tracks these across real requests; realistic values
// must be used or the signature is treated as a replay/forgery.
type XGnarlyCounters struct {
	TotalXHRRequests         uint32
	TotalFetchRequests       uint32
	InterceptedXHRRequests   uint32
	InterceptedFetchRequests uint32
}

func xgRotl(v uint32, c uint) uint32 { return bits.RotateLeft32(v, int(c)) }

func xgQuarter(s []uint32, a, b, c, d int) {
	s[a] += s[b]
	s[d] = xgRotl(s[d]^s[a], 16)
	s[c] += s[d]
	s[b] = xgRotl(s[b]^s[c], 12)
	s[a] += s[b]
	s[d] = xgRotl(s[d]^s[a], 8)
	s[c] += s[d]
	s[b] = xgRotl(s[b]^s[c], 7)
}

func xgChachaBlock(initial []uint32, rounds int) []uint32 {
	s := make([]uint32, 16)
	copy(s, initial)
	r := 0
	for r < rounds {
		xgQuarter(s, 0, 4, 8, 12)
		xgQuarter(s, 1, 5, 9, 13)
		xgQuarter(s, 2, 6, 10, 14)
		xgQuarter(s, 3, 7, 11, 15)
		r++
		if r >= rounds {
			break
		}
		// NOTE: intentionally non-standard diagonal round (matches xgnarly.mjs)
		xgQuarter(s, 0, 5, 10, 15)
		xgQuarter(s, 1, 6, 11, 12)
		xgQuarter(s, 2, 7, 12, 13)
		xgQuarter(s, 3, 4, 13, 14)
		r++
	}
	for i := 0; i < 16; i++ {
		s[i] += initial[i]
	}
	return s
}

func xgDeriveRounds(keyWords []uint32) int {
	var r uint32
	for _, w := range keyWords {
		r = (r + (w & 15)) & 15
	}
	return int(r) + 5
}

func xgChachaXor(bytes []byte, keyWords []uint32, rounds int) []byte {
	state := make([]uint32, 16)
	copy(state[:4], xgSigma[:])
	copy(state[4:], keyWords)
	for off := 0; off < len(bytes); off += 64 {
		stream := xgChachaBlock(state, rounds)
		state[12]++
		lim := 64
		if len(bytes)-off < lim {
			lim = len(bytes) - off
		}
		for i := 0; i < lim; i++ {
			word := stream[i>>2]
			b := byte((word >> (8 * (i & 3))) & 0xff)
			bytes[off+i] ^= b
		}
	}
	return bytes
}

func xgEncodeBase64(bytes []byte) string {
	var out strings.Builder
	i := 0
	for ; i+3 <= len(bytes); i += 3 {
		n := (uint(bytes[i]) << 16) | (uint(bytes[i+1]) << 8) | uint(bytes[i+2])
		out.WriteByte(xgAlphabet[(n>>18)&63])
		out.WriteByte(xgAlphabet[(n>>12)&63])
		out.WriteByte(xgAlphabet[(n>>6)&63])
		out.WriteByte(xgAlphabet[n&63])
	}
	rem := len(bytes) - i
	if rem == 1 {
		n := uint(bytes[i]) << 16
		out.WriteByte(xgAlphabet[(n>>18)&63])
		out.WriteByte(xgAlphabet[(n>>12)&63])
		out.WriteString("==")
	} else if rem == 2 {
		n := (uint(bytes[i]) << 16) | (uint(bytes[i+1]) << 8)
		out.WriteByte(xgAlphabet[(n>>18)&63])
		out.WriteByte(xgAlphabet[(n>>12)&63])
		out.WriteByte(xgAlphabet[(n>>6)&63])
		out.WriteByte('=')
	}
	return out.String()
}

func xgMD5(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

var xgFieldOrder = [16]int{1, 8, 12, 11, 6, 9, 4, 7, 0, 14, 15, 2, 3, 10, 5, 13}

var xgIntWidths = map[int]int{0: 4, 1: 2, 2: 2, 6: 4, 7: 4, 8: 4, 11: 2, 12: 2, 13: 2, 14: 4, 15: 4}

func xgIntToFixedBytes(n uint32, width int) []byte {
	out := make([]byte, width)
	for i := width - 1; i >= 0; i-- {
		out[i] = byte(n & 0xff)
		n >>= 8
	}
	return out
}

type xgnarlyOptions struct {
	TimestampMs int64
	Ubcode      int
	SdkVersion  string
	RandomLow16 []byte // 2 bytes
	Random32    []byte // 4 bytes
	RandomKey   []byte // 48 bytes
}

// xgEncode mirrors xgnarly.mjs encode() exactly (deterministic inputs only
// used for testing/verification).
func xgEncode(queryString, body, userAgent string, counters XGnarlyCounters, opts xgnarlyOptions) string {
	ts := opts.TimestampMs
	ubcode := opts.Ubcode
	sdkVersion := opts.SdkVersion

	field14 := uint32(65<<16) | uint32(opts.RandomLow16[0])<<8 | uint32(opts.RandomLow16[1])
	field15 := uint32(opts.Random32[0])<<24 | uint32(opts.Random32[1])<<16 | uint32(opts.Random32[2])<<8 | uint32(opts.Random32[3])

	field6 := uint32(ts / 1000)
	field8 := uint32(uint64(ts) % 0x80000000)
	field7 := uint32(3181061566)
	field12 := counters.TotalXHRRequests + counters.TotalFetchRequests
	field13 := counters.InterceptedXHRRequests + counters.InterceptedFetchRequests

	// xorHeader = XOR of all numeric field values (fields 1,2,6,7,8,11,12,13,14,15)
	xorHeader := uint32(65) ^ uint32(ubcode) ^ field6 ^ field7 ^ field8 ^ uint32(1) ^ field12 ^ field13 ^ field14 ^ field15

	// numeric fields (value -> width)
	numFields := map[int]uint32{
		0:  xorHeader,
		1:  65,
		2:  uint32(ubcode),
		6:  field6,
		7:  field7,
		8:  field8,
		11: 1,
		12: field12,
		13: field13,
		14: field14,
		15: field15,
	}
	strFields := map[int]string{
		3:  xgMD5(queryString),
		4:  xgMD5(body),
		5:  xgMD5(userAgent),
		9:  "5.1.3-ZTCA",
		10: sdkVersion,
	}

	// Build payload
	var payload []byte
	payload = append(payload, 16) // all 16 fields present
	for _, k := range xgFieldOrder {
		var valueBytes []byte
		if v, ok := numFields[k]; ok {
			valueBytes = xgIntToFixedBytes(v, xgIntWidths[k])
		} else if v, ok := strFields[k]; ok {
			valueBytes = []byte(v)
		}
		payload = append(payload, byte(k))
		payload = append(payload, byte(len(valueBytes)>>8), byte(len(valueBytes)))
		payload = append(payload, valueBytes...)
	}

	// Key words (little-endian from 48 key bytes)
	keyWords := make([]uint32, 12)
	for i := 0; i < 12; i++ {
		o := i * 4
		keyWords[i] = uint32(opts.RandomKey[o]) | uint32(opts.RandomKey[o+1])<<8 |
			uint32(opts.RandomKey[o+2])<<16 | uint32(opts.RandomKey[o+3])<<24
	}
	rounds := xgDeriveRounds(keyWords)

	cipher := make([]byte, len(payload))
	copy(cipher, payload)
	xgChachaXor(cipher, keyWords, rounds)

	xLen := len(cipher)
	mod := xLen + 1
	var sum int
	for _, b := range opts.RandomKey {
		sum = (sum + int(b)) % mod
	}
	for _, b := range cipher {
		sum = (sum + int(b)) % mod
	}
	insertPos := sum

	out := make([]byte, 0, 1+xLen+48)
	out = append(out, 75) // MAGIC_BYTE
	out = append(out, cipher[:insertPos]...)
	out = append(out, opts.RandomKey...)
	out = append(out, cipher[insertPos:]...)

	return xgEncodeBase64(out)
}

// XGnarly generates a valid X-Gnarly signature using random key material.
// queryString is the URL query (WITHOUT X-Bogus/X-Gnarly, WITH msToken),
// body is the request body ("" for GET), userAgent must match the request.
func XGnarly(queryString, body, userAgent string, counters XGnarlyCounters) string {
	r14 := make([]byte, 2)
	r15 := make([]byte, 4)
	key := make([]byte, 48)
	rand.Read(r14)
	rand.Read(r15)
	rand.Read(key)
	return xgEncode(queryString, body, userAgent, counters, xgnarlyOptions{
		TimestampMs: time.Now().UnixMilli(),
		Ubcode:      4,
		SdkVersion:  "1.0.0.368",
		RandomLow16: r14,
		Random32:    r15,
		RandomKey:   key,
	})
}
