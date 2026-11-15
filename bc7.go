package main

// Decodeur BC7 : portage fidele de celui de Pillow (src/libImaging/BcnDecode.c), pour obtenir exactement les memes pixels
// que l'ancien script Python (qui passait par Pillow).

type rgba struct{ r, g, b, a uint8 }

type bc7Mode struct{ ns, pb, rb, isb, cb, ab, epb, spb, ib, ib2 int }

var bc7Modes = [8]bc7Mode{
	{3, 4, 0, 0, 4, 0, 1, 0, 3, 0},
	{2, 6, 0, 0, 6, 0, 0, 1, 3, 0},
	{3, 6, 0, 0, 5, 0, 0, 0, 2, 0},
	{2, 6, 0, 0, 7, 0, 1, 0, 2, 0},
	{1, 0, 2, 1, 5, 6, 0, 0, 2, 3},
	{1, 0, 2, 0, 7, 8, 0, 0, 2, 2},
	{1, 0, 0, 0, 7, 7, 1, 0, 4, 0},
	{2, 6, 0, 0, 5, 5, 1, 0, 2, 0},
}

var bc7Si2 = [64]uint16{
	0xcccc, 0x8888, 0xeeee, 0xecc8, 0xc880, 0xfeec, 0xfec8, 0xec80, 0xc800, 0xffec,
	0xfe80, 0xe800, 0xffe8, 0xff00, 0xfff0, 0xf000, 0xf710, 0x008e, 0x7100, 0x08ce,
	0x008c, 0x7310, 0x3100, 0x8cce, 0x088c, 0x3110, 0x6666, 0x366c, 0x17e8, 0x0ff0,
	0x718e, 0x399c, 0xaaaa, 0xf0f0, 0x5a5a, 0x33cc, 0x3c3c, 0x55aa, 0x9696, 0xa55a,
	0x73ce, 0x13c8, 0x324c, 0x3bdc, 0x6996, 0xc33c, 0x9966, 0x0660, 0x0272, 0x04e4,
	0x4e40, 0x2720, 0xc936, 0x936c, 0x39c6, 0x639c, 0x9336, 0x9cc6, 0x817e, 0xe718,
	0xccf0, 0x0fcc, 0x7744, 0xee22,
}

var bc7Si3 = [64]uint32{
	0xaa685050, 0x6a5a5040, 0x5a5a4200, 0x5450a0a8, 0xa5a50000, 0xa0a05050, 0x5555a0a0,
	0x5a5a5050, 0xaa550000, 0xaa555500, 0xaaaa5500, 0x90909090, 0x94949494, 0xa4a4a4a4,
	0xa9a59450, 0x2a0a4250, 0xa5945040, 0x0a425054, 0xa5a5a500, 0x55a0a0a0, 0xa8a85454,
	0x6a6a4040, 0xa4a45000, 0x1a1a0500, 0x0050a4a4, 0xaaa59090, 0x14696914, 0x69691400,
	0xa08585a0, 0xaa821414, 0x50a4a450, 0x6a5a0200, 0xa9a58000, 0x5090a0a8, 0xa8a09050,
	0x24242424, 0x00aa5500, 0x24924924, 0x24499224, 0x50a50a50, 0x500aa550, 0xaaaa4444,
	0x66660000, 0xa5a0a5a0, 0x50a050a0, 0x69286928, 0x44aaaa44, 0x66666600, 0xaa444444,
	0x54a854a8, 0x95809580, 0x96969600, 0xa85454a8, 0x80959580, 0xaa141414, 0x96960000,
	0xaaaa1414, 0xa05050a0, 0xa0a5a5a0, 0x96000000, 0x40804080, 0xa9a8a9a8, 0xaaaaaa44,
	0x2a4a5254,
}

var bc7Ai0 = [64]int{15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15,
	15, 15, 15, 15, 2, 8, 2, 2, 8, 8, 15, 2, 8,
	2, 2, 8, 8, 2, 2, 15, 15, 6, 8, 2, 8, 15,
	15, 2, 8, 2, 2, 2, 15, 15, 6, 6, 2, 6, 8,
	15, 15, 2, 2, 15, 15, 15, 15, 15, 2, 2, 15}

var bc7Ai1 = [64]int{3, 3, 15, 15, 8, 3, 15, 15, 8, 8, 6, 6, 6,
	5, 3, 3, 3, 3, 8, 15, 3, 3, 6, 10, 5, 8,
	8, 6, 8, 5, 15, 15, 8, 15, 3, 5, 6, 10, 8,
	15, 15, 3, 15, 5, 15, 15, 15, 15, 3, 15, 5, 5,
	5, 8, 5, 10, 5, 10, 8, 13, 15, 12, 3, 3}

var bc7Ai2 = [64]int{15, 8, 8, 3, 15, 15, 3, 8, 15, 15, 15, 15, 15,
	15, 15, 8, 15, 8, 15, 3, 15, 8, 15, 8, 3, 15,
	6, 10, 15, 15, 10, 8, 15, 3, 15, 10, 10, 8, 9,
	10, 6, 15, 8, 15, 3, 6, 6, 8, 15, 3, 15, 15,
	15, 15, 15, 15, 15, 15, 15, 15, 3, 15, 15, 8}

var bc7W2 = []int{0, 21, 43, 64}
var bc7W3 = []int{0, 9, 18, 27, 37, 46, 55, 64}
var bc7W4 = []int{0, 4, 9, 13, 17, 21, 26, 30, 34, 38, 43, 47, 51, 55, 60, 64}

func bc7Weights(n int) []int {
	if n == 2 {
		return bc7W2
	}
	if n == 3 {
		return bc7W3
	}
	return bc7W4
}

func bc7Subset(ns, partition, n int) int {
	if ns == 2 {
		return int(1 & (bc7Si2[partition] >> uint(n)))
	}
	if ns == 3 {
		return int(3 & (bc7Si3[partition] >> uint(2*n)))
	}
	return 0
}

func getBits(src []byte, bit, count int) uint8 {
	if count == 0 {
		return 0
	}
	by := bit >> 3
	bit &= 7
	if bit+count <= 8 {
		return (src[by] >> uint(bit)) & uint8((1<<uint(count))-1)
	}
	x := int(src[by]) | int(src[by+1])<<8
	return uint8((x >> uint(bit)) & ((1 << uint(count)) - 1))
}

func expandQuantized(v uint8, bits int) uint8 {
	v = v << uint(8-bits)
	return v | (v >> uint(bits))
}

func bc7Lerp(dst *rgba, e []rgba, s0, s1 int) {
	t0 := 64 - s0
	t1 := 64 - s1
	dst.r = uint8((t0*int(e[0].r) + s0*int(e[1].r) + 32) >> 6)
	dst.g = uint8((t0*int(e[0].g) + s0*int(e[1].g) + 32) >> 6)
	dst.b = uint8((t0*int(e[0].b) + s0*int(e[1].b) + 32) >> 6)
	dst.a = uint8((t1*int(e[0].a) + s1*int(e[1].a) + 32) >> 6)
}

// decodeBC7Block : 16 pixels (ligne par ligne) d'un bloc de 16 octets
func decodeBC7Block(col *[16]rgba, src []byte) {
	var endpoints [6]rgba
	bit := 0
	mode := int(src[0])
	if mode == 0 {
		for i := 0; i < 16; i++ {
			col[i] = rgba{0, 0, 0, 255}
		}
		return
	}
	for {
		b := bit
		bit++
		if mode&(1<<uint(b)) != 0 {
			break
		}
	}
	mode = bit - 1
	info := bc7Modes[mode]
	cb := info.cb
	ab := info.ab
	cw := bc7Weights(info.ib)
	aib := info.ib
	if ab != 0 && info.ib2 != 0 {
		aib = info.ib2
	}
	aw := bc7Weights(aib)
	load := func(n int) uint8 {
		v := getBits(src, bit, n)
		bit += n
		return v
	}
	partition := int(load(info.pb))
	rotation := load(info.rb)
	indexSel := load(info.isb)
	numep := info.ns << 1
	for i := 0; i < numep; i++ {
		endpoints[i].r = load(cb)
	}
	for i := 0; i < numep; i++ {
		endpoints[i].g = load(cb)
	}
	for i := 0; i < numep; i++ {
		endpoints[i].b = load(cb)
	}
	for i := 0; i < numep; i++ {
		if ab != 0 {
			endpoints[i].a = load(ab)
		} else {
			endpoints[i].a = 255
		}
	}
	if info.epb != 0 {
		cb++
		if ab != 0 {
			ab++
		}
		for i := 0; i < numep; i++ {
			val := load(1)
			e := &endpoints[i]
			e.r = e.r<<1 | val
			e.g = e.g<<1 | val
			e.b = e.b<<1 | val
			if ab != 0 {
				e.a = e.a<<1 | val
			}
		}
	}
	if info.spb != 0 {
		cb++
		if ab != 0 {
			ab++
		}
		for i := 0; i < numep; i += 2 {
			val := load(1)
			for j := 0; j < 2; j++ {
				e := &endpoints[i+j]
				e.r = e.r<<1 | val
				e.g = e.g<<1 | val
				e.b = e.b<<1 | val
				if ab != 0 {
					e.a = e.a<<1 | val
				}
			}
		}
	}
	for i := 0; i < numep; i++ {
		e := &endpoints[i]
		e.r = expandQuantized(e.r, cb)
		e.g = expandQuantized(e.g, cb)
		e.b = expandQuantized(e.b, cb)
		if ab != 0 {
			e.a = expandQuantized(e.a, ab)
		}
	}
	cibit := bit
	aibit := cibit + 16*info.ib - info.ns
	for i := 0; i < 16; i++ {
		s := bc7Subset(info.ns, partition, i) << 1
		ib := info.ib
		if i == 0 {
			ib--
		} else if info.ns == 2 {
			if i == bc7Ai0[partition] {
				ib--
			}
		} else if info.ns == 3 {
			if i == bc7Ai1[partition] {
				ib--
			} else if i == bc7Ai2[partition] {
				ib--
			}
		}
		i0 := int(getBits(src, cibit, ib))
		cibit += ib
		if ab != 0 && info.ib2 != 0 {
			ib2 := info.ib2
			if i == 0 {
				ib2--
			}
			i1 := int(getBits(src, aibit, ib2))
			aibit += ib2
			if indexSel != 0 {
				bc7Lerp(&col[i], endpoints[s:s+2], aw[i1], cw[i0])
			} else {
				bc7Lerp(&col[i], endpoints[s:s+2], cw[i0], aw[i1])
			}
		} else {
			bc7Lerp(&col[i], endpoints[s:s+2], cw[i0], cw[i0])
		}
		c := &col[i]
		switch rotation {
		case 1:
			c.r, c.a = c.a, c.r
		case 2:
			c.g, c.a = c.a, c.g
		case 3:
			c.b, c.a = c.a, c.b
		}
	}
}

// decodeBC7 : image RGBA (non premultipliee) de largeur w, hauteur h (multiples de 4), 4 octets par pixel
func decodeBC7(data []byte, w, h int) []byte {
	out := make([]byte, w*h*4)
	var col [16]rgba
	bw := w / 4
	for by := 0; by < h/4; by++ {
		for bx := 0; bx < bw; bx++ {
			off := (by*bw + bx) * 16
			if off+16 > len(data) {
				return out
			}
			decodeBC7Block(&col, data[off:off+16])
			for py := 0; py < 4; py++ {
				row := (by*4+py)*w*4 + bx*16
				for px := 0; px < 4; px++ {
					c := col[py*4+px]
					p := row + px*4
					out[p], out[p+1], out[p+2], out[p+3] = c.r, c.g, c.b, c.a
				}
			}
		}
	}
	return out
}
