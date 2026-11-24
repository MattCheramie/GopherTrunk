package wmbus

import "fmt"

// Qundis "walk-by data set" (#1256). Qundis heat cost allocators and water
// meters (QDS, and Siemens/Qundis LSE) send their readings in CI 0x78
// telegrams inside one manufacturer-specific record, DIF 0x0D (variable
// length), VIF 0xFF, VIFE 0x5F, rather than as standard data records. The
// block's content (after its length byte), as wmbusmeters' qcaloric grammar
// reads it:
//
//	[0] 0x00  [1] 0x82 (type 2 walk-by data set)  [2] transmission counter
//	[3] ...   [4] 0x00 plaintext / 0x35 encrypted  [5..6] flags  [7] 0x07
//	[8] meter type (0xB0 heat cost allocator, 0xC1 water)
//	[9] VIF of the readings (0x6E HCA units, 0x13 litres, ...)
//	[10..11] error date (type G)     [12..15] current reading (BCD-8)
//	[16..17] last set day (type G)   [18..21] reading at that set day
//	[22..23] previous set day         [24..27] reading at it
//
// The readings are re-expressed as the standard records they stand for
// (current reading, storage 1 = last set day, storage 17 = the month-end
// set day; the meter's own semantics, following wmbusmeters' DIF/VIF keys
// 0C/4C/CC08 + VIF and 426C/C2086C) and parsed by ParseRecords, so they
// print and export exactly like a standard meter's.
//
// An encrypted block ([4] = 0x35) is AES-128-CBC from [5] on with the meter's
// key, IV = M + A (the link layer's, or the meter's own from an enhanced
// identification record behind a radio converter) + 8 × the counter at [2] (wmbusmeters
// tryDecodeQundisWalkByAes); the plaintext must show the same 0x07 and meter
// type bytes a clear block carries.

const qdsWalkByMinLen = 28

// expandQundisWalkBy replaces a Qundis walk-by record with the standard
// records it carries. dll is the frame's data (for the IV), key the meter's
// key or nil. It reports a decryption failure in t.DecryptErr.
func expandQundisWalkBy(t *Telegram, recs []Record, dll, key []byte) []Record {
	if t.Manufacturer != "QDS" && t.Manufacturer != "LSE" {
		return recs
	}
	for i, r := range recs {
		if r.DIF != 0x0D || r.VIF != 0xFF || len(r.VIFE) != 1 || r.VIFE[0] != 0x5F || len(r.Raw) < 1+qdsWalkByMinLen {
			continue
		}
		b := append([]byte(nil), r.Raw[1:]...)
		if b[1] != 0x82 {
			continue
		}
		switch b[4] {
		case 0x00:
		case 0x35:
			if key == nil {
				t.Encrypted = true
				continue
			}
			plain, err := decryptQundisWalkBy(b, qundisIVAddress(recs, dll), key)
			if err != nil {
				t.Encrypted = true
				t.DecryptErr = "Qundis walk-by data: " + err.Error()
				continue
			}
			b = plain
			t.Decrypted = true
		default:
			continue
		}
		if b[7] != 0x07 {
			continue
		}
		vif := b[9]
		if vif&0x80 != 0 || vif >= 0x7C {
			continue // not a plain primary VIF: not the layout above
		}
		syn := make([]byte, 0, 40)
		syn = append(syn, 0x32, 0x6C, b[10], b[11]) // error date
		syn = append(syn, 0x0C, vif)                // current reading
		syn = append(syn, b[12:16]...)
		syn = append(syn, 0x42, 0x6C, b[16], b[17]) // last set day
		syn = append(syn, 0x4C, vif)
		syn = append(syn, b[18:22]...)
		syn = append(syn, 0xC2, 0x08, 0x6C, b[22], b[23]) // set day, storage 17
		syn = append(syn, 0xCC, 0x08, vif)
		syn = append(syn, b[24:28]...)
		out, err := ParseRecords(syn)
		if err != nil {
			continue
		}
		res := append(append(append([]Record(nil), recs[:i]...), out...), recs[i+1:]...)
		t.QundisWalkBy = true
		return res
	}
	return recs
}

// qundisIVAddress is the M-field + A-field (8 bytes) a walk-by block is
// encrypted under: the link layer's, unless the telegram carries the meter's
// own identity in an enhanced identification record (DIF 07 VIF 79: ID 4,
// M 2, version, type), as a meter behind a radio converter does — the
// converter's address is then on the link layer, and the meter encrypted
// with its own (wmbusmeters tryDecodeQundisWalkByAes).
func qundisIVAddress(recs []Record, dll []byte) []byte {
	for _, r := range recs {
		if r.DIF == 0x07 && r.VIF == 0x79 && len(r.Raw) == 8 {
			id := r.Raw
			return []byte{id[4], id[5], id[0], id[1], id[2], id[3], id[6], id[7]}
		}
	}
	return dll[2:10]
}

func decryptQundisWalkBy(b, addr, key []byte) ([]byte, error) {
	body := b[5:]
	if len(body)%16 != 0 {
		return nil, fmt.Errorf("encrypted body %d bytes, not whole AES blocks", len(body))
	}
	iv := make([]byte, 0, 16)
	iv = append(iv, addr...)
	for len(iv) < 16 {
		iv = append(iv, b[2])
	}
	out := make([]byte, len(b))
	copy(out, b[:5])
	out[4] = 0x00
	if err := cbcDecrypt(key, iv, out[5:], body); err != nil {
		return nil, err
	}
	if out[7] != 0x07 || (out[8] != 0xB0 && out[8]&0xF0 != 0xC0) {
		return nil, fmt.Errorf("plaintext header %X: %w", out[5:10], errWrongKey)
	}
	return out, nil
}
