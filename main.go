// BardN Item Card - extracteur d'icones (double-clic).
// Lit en lecture seule re_chunk_000.pak de TON jeu, et ecrit une image par objet dans
// <jeu>/reframework/images/BardN_Item_Card/icons/<numero>.png. Rien n'est envoye nulle part, rien n'est modifie dans le jeu.
// Portage du script Python BardN_extraire_icones.py (meme format de .pak, meme decodage, memes pixels).
package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/klauspost/compress/zstd"
)

const version = "1.18" // version du mod BardN Item Card livree avec ce programme

// ------------------------------------------------------------------ messages bilingues

func dire(fr, en string, a ...any) {
	fmt.Printf(fr+"\n", a...)
	fmt.Printf("   "+en+"\n", a...)
}

// ------------------------------------------------------------------ hachage des noms (Murmur3, graine 0xFFFFFFFF, UTF-16LE)

func rotl(x uint32, r uint) uint32 { return x<<r | x>>(32-r) }

func murmur3(data []byte) uint32 {
	h := uint32(0xFFFFFFFF)
	n := len(data) / 4 * 4
	for i := 0; i < n; i += 4 {
		k := binary.LittleEndian.Uint32(data[i:])
		k *= 0xCC9E2D51
		k = rotl(k, 15)
		k *= 0x1B873593
		h ^= k
		h = rotl(h, 13)
		h = h*5 + 0xE6546B64
	}
	t := data[n:]
	var k uint32
	if len(t) >= 3 {
		k ^= uint32(t[2]) << 16
	}
	if len(t) >= 2 {
		k ^= uint32(t[1]) << 8
	}
	if len(t) >= 1 {
		k ^= uint32(t[0])
		k *= 0xCC9E2D51
		k = rotl(k, 15)
		k *= 0x1B873593
		h ^= k
	}
	h ^= uint32(len(data))
	h ^= h >> 16
	h *= 0x85EBCA6B
	h ^= h >> 13
	h *= 0xC2B2AE35
	h ^= h >> 16
	return h
}

func utf16le(s string) []byte {
	u := utf16.Encode([]rune(s))
	b := make([]byte, len(u)*2)
	for i, c := range u {
		binary.LittleEndian.PutUint16(b[i*2:], c)
	}
	return b
}

type cleHash struct{ bas, haut uint32 }

func hashChemin(c string) cleHash {
	return cleHash{murmur3(utf16le(strings.ToLower(c))), murmur3(utf16le(strings.ToUpper(c)))}
}

// ------------------------------------------------------------------ table du .pak (version 4.x, option 104 = Dragon's Dogma 2)

var module = []byte{
	0x7D, 0x0B, 0xF8, 0xC1, 0x7C, 0x23, 0xFD, 0x3B, 0xD4, 0x75, 0x16, 0xD2, 0x33, 0x21, 0xD8, 0x10,
	0x71, 0xF9, 0x7C, 0xD1, 0x34, 0x93, 0xBA, 0x77, 0x26, 0xFC, 0xAB, 0x2C, 0xEE, 0xDA, 0xD9, 0x1C,
	0x89, 0xE7, 0x29, 0x7B, 0xDD, 0x8A, 0xAE, 0x50, 0x39, 0xB6, 0x01, 0x6D, 0x21, 0x89, 0x5D, 0xA5,
	0xA1, 0x3E, 0xA2, 0xC0, 0x8C, 0x93, 0x13, 0x36, 0x65, 0xEB, 0xE8, 0xDF, 0x06, 0x17, 0x67, 0x96,
	0x06, 0x2B, 0xAC, 0x23, 0xED, 0x8C, 0xB7, 0x8B, 0x90, 0xAD, 0xEA, 0x71, 0xC4, 0x40, 0x44, 0x9D,
	0x1C, 0x7B, 0xBA, 0xC4, 0xB6, 0x2D, 0xD6, 0xD2, 0x4B, 0x62, 0xD6, 0x26, 0xFC, 0x74, 0x20, 0x07,
	0xEC, 0xE3, 0x59, 0x9A, 0xE6, 0xAF, 0xB9, 0xA8, 0x35, 0x8B, 0xE0, 0xE8, 0xD3, 0xCD, 0x45, 0x65,
	0xB0, 0x91, 0xC4, 0x95, 0x1B, 0xF3, 0x23, 0x1E, 0xC6, 0x71, 0xCF, 0x3E, 0x35, 0x2D, 0x6B, 0xE3,
}

func petitBoutiste(b []byte) *big.Int { // entier petit-boutiste -> big.Int
	r := make([]byte, len(b))
	for i := range b {
		r[len(b)-1-i] = b[i]
	}
	return new(big.Int).SetBytes(r)
}

func dechiffrerTable(table, cleChiffree []byte) {
	m := petitBoutiste(module)
	c := petitBoutiste(cleChiffree[:128])
	r := new(big.Int).Exp(c, big.NewInt(0x00010001), m)
	be := r.Bytes()
	cle := make([]byte, len(be))
	for i := range be {
		cle[i] = be[len(be)-1-i]
	}
	for len(cle) < 32 {
		cle = append(cle, 0)
	}
	const periode = 7424
	motif := make([]byte, periode)
	for i := 0; i < periode; i++ {
		motif[i] = byte(i + int(cle[i%32])*int(cle[i%29]))
	}
	for i := range table {
		table[i] ^= motif[i%periode]
	}
}

type entree struct {
	dec, tc, tf, attr int64
}

type pak struct {
	chemin string
	f      *os.File
	total  int
	table  []byte
}

func ouvrirPak(chemin string) (*pak, error) {
	f, err := os.Open(chemin)
	if err != nil {
		return nil, err
	}
	p := &pak{chemin: chemin, f: f}
	if err := p.lireTable(); err != nil {
		f.Close()
		return nil, err
	}
	return p, nil
}

func (p *pak) lireTable() error {
	r := bufio.NewReaderSize(p.f, 1<<20)
	h := make([]byte, 16)
	if _, err := ioFull(r, h); err != nil {
		return errors.New("fichier trop court / file too short")
	}
	if string(h[:4]) != "KPKA" {
		return errors.New("signature KPKA absente / not a game .pak")
	}
	maj := h[4]
	option := binary.LittleEndian.Uint16(h[6:])
	total := int(binary.LittleEndian.Uint32(h[8:]))
	if maj != 4 {
		return fmt.Errorf("version de .pak non geree : %d", maj)
	}
	table := make([]byte, total*48)
	if _, err := ioFull(r, table); err != nil {
		return errors.New("table d'entrees incomplete / incomplete entry table")
	}
	saute := func(n int) error { _, err := r.Discard(n); return err }
	blocs := func(taille int) error {
		b := make([]byte, 8)
		if _, err := ioFull(r, b); err != nil {
			return err
		}
		var nb int32
		if taille == 8 {
			nb = int32(binary.LittleEndian.Uint32(b[4:]))
		} else {
			nb = int32(binary.LittleEndian.Uint32(b[0:]))
		}
		return saute(int(nb) * taille)
	}
	switch option {
	case 0:
	case 8, 12, 24, 40, 44, 104:
		if option == 24 {
			saute(4)
		} else if option == 12 || option == 44 {
			saute(9)
		}
		cle := make([]byte, 128)
		if option == 104 {
			if err := blocs(16); err != nil {
				return err
			}
			if _, err := ioFull(r, cle); err != nil {
				return err
			}
			// (les blocs de 8 qui suivent ne servent pas ici)
		} else {
			if _, err := ioFull(r, cle); err != nil {
				return err
			}
		}
		dechiffrerTable(table, cle)
	default:
		return fmt.Errorf("option de .pak inconnue : %d", option)
	}
	p.total, p.table = total, table
	return nil
}

func ioFull(r *bufio.Reader, b []byte) (int, error) {
	n := 0
	for n < len(b) {
		m, err := r.Read(b[n:])
		n += m
		if err != nil {
			if n == len(b) {
				return n, nil
			}
			return n, err
		}
	}
	return n, nil
}

func (p *pak) chercher(voulus map[cleHash]string) map[string]entree {
	res := map[string]entree{}
	t := p.table
	for i := 0; i+48 <= len(t); i += 48 {
		k := cleHash{binary.LittleEndian.Uint32(t[i:]), binary.LittleEndian.Uint32(t[i+4:])}
		if c, ok := voulus[k]; ok {
			res[c] = entree{
				dec:  int64(binary.LittleEndian.Uint64(t[i+8:])),
				tc:   int64(binary.LittleEndian.Uint64(t[i+16:])),
				tf:   int64(binary.LittleEndian.Uint64(t[i+24:])),
				attr: int64(binary.LittleEndian.Uint64(t[i+32:])),
			}
		}
	}
	return res
}

var dzstd, _ = zstd.NewReader(nil, zstd.WithDecoderConcurrency(1), zstd.WithDecoderMaxMemory(1<<30))

func (p *pak) lire(e entree) ([]byte, error) {
	compression := e.attr & 0xF
	chiffrement := (e.attr & 0x00FF0000) >> 16
	if chiffrement != 0 {
		return nil, fmt.Errorf("entree chiffree (type %d)", chiffrement)
	}
	brut := make([]byte, e.tc)
	if _, err := p.f.ReadAt(brut, e.dec); err != nil {
		return nil, err
	}
	switch compression {
	case 2:
		return dzstd.DecodeAll(brut, make([]byte, 0, e.tf))
	case 0:
		return brut, nil
	}
	return nil, fmt.Errorf("compression %d non geree", compression)
}

// ------------------------------------------------------------------ texture TEX (BC7, un niveau, en-tete de 56 octets) -> image

func decoderTex(tex []byte) (*image.NRGBA, error) {
	if len(tex) < 56 || string(tex[:4]) != "TEX\x00" {
		return nil, errors.New("pas une texture TEX")
	}
	largeur := int(binary.LittleEndian.Uint16(tex[8:]))
	hauteur := int(binary.LittleEndian.Uint16(tex[10:]))
	format := binary.LittleEndian.Uint32(tex[16:])
	decalage := int(binary.LittleEndian.Uint64(tex[40:]))
	pitch := int(binary.LittleEndian.Uint32(tex[48:]))
	taille := int(binary.LittleEndian.Uint32(tex[52:]))
	if format != 98 && format != 99 {
		return nil, fmt.Errorf("format de texture %d non gere (BC7 attendu)", format)
	}
	if decalage+taille > len(tex) || pitch == 0 {
		return nil, errors.New("donnees de texture incompletes")
	}
	lt := pitch / 16 * 4
	ht := taille / pitch * 4
	pix := decodeBC7(tex[decalage:decalage+taille], lt, ht)
	im := image.NewNRGBA(image.Rect(0, 0, largeur, hauteur))
	for y := 0; y < hauteur && y < ht; y++ {
		copy(im.Pix[y*im.Stride:y*im.Stride+largeur*4], pix[y*lt*4:y*lt*4+largeur*4])
	}
	return im, nil
}

// ------------------------------------------------------------------ recherche du dossier du jeu

func estJeu(d string) bool {
	st, err := os.Stat(filepath.Join(d, "re_chunk_000.pak"))
	return err == nil && !st.IsDir()
}

func candidatsSteam() []string {
	var bibliotheques []string
	ajoute := func(s string) {
		s = strings.TrimSpace(s)
		if s != "" {
			bibliotheques = append(bibliotheques, s)
		}
	}
	if runtime.GOOS == "windows" {
		for _, cle := range []string{`HKCU\Software\Valve\Steam`, `HKLM\SOFTWARE\WOW6432Node\Valve\Steam`, `HKLM\SOFTWARE\Valve\Steam`} {
			for _, val := range []string{"SteamPath", "InstallPath"} {
				out, err := exec.Command("reg", "query", cle, "/v", val).Output()
				if err != nil {
					continue
				}
				if m := regexp.MustCompile(`REG_SZ\s+(.+)`).FindSubmatch(out); m != nil {
					ajoute(strings.ReplaceAll(string(m[1]), "/", `\`))
				}
			}
		}
		for l := 'C'; l <= 'Z'; l++ {
			for _, s := range []string{`\Program Files (x86)\Steam`, `\Program Files\Steam`, `\Steam`, `\SteamLibrary`, `\Games\Steam`, `\Jeux\Steam`} {
				ajoute(string(l) + ":" + s)
			}
		}
	} else if home, err := os.UserHomeDir(); err == nil {
		ajoute(filepath.Join(home, ".steam", "steam"))
		ajoute(filepath.Join(home, ".local", "share", "Steam"))
	}
	// les autres bibliotheques Steam sont listees dans steamapps/libraryfolders.vdf
	re := regexp.MustCompile(`"path"\s+"([^"]+)"`)
	vus := map[string]bool{}
	var res []string
	var tout []string
	for _, b := range bibliotheques {
		tout = append(tout, b)
		if data, err := os.ReadFile(filepath.Join(b, "steamapps", "libraryfolders.vdf")); err == nil {
			for _, m := range re.FindAllSubmatch(data, -1) {
				tout = append(tout, strings.ReplaceAll(string(m[1]), `\\`, `\`))
			}
		}
	}
	for _, b := range tout {
		d := filepath.Join(b, "steamapps", "common", "Dragons Dogma 2")
		k := strings.ToLower(d)
		if !vus[k] {
			vus[k] = true
			res = append(res, d)
		}
	}
	return res
}

func trouverJeu(arg string) string {
	if arg != "" {
		for d, i := strings.Trim(arg, `"`), 0; i < 4; i++ {
			if estJeu(d) {
				return d
			}
			d = filepath.Dir(d)
		}
	}
	// 1) a cote du programme (ou jusqu'a 4 dossiers au-dessus : <jeu>/Outils/ ou <jeu>/reframework/...)
	if exe, err := os.Executable(); err == nil {
		d := filepath.Dir(exe)
		for i := 0; i < 5; i++ {
			if estJeu(d) {
				return d
			}
			d = filepath.Dir(d)
		}
	}
	if wd, err := os.Getwd(); err == nil && estJeu(wd) {
		return wd
	}
	// 2) bibliotheques Steam
	for _, d := range candidatsSteam() {
		if estJeu(d) {
			return d
		}
	}
	return ""
}

// ------------------------------------------------------------------ programme

const dossierAtlas = "natives/stm/gui/ui01/common/item/%s/tex_ui01%s_%02d_iam.tex.251211553"

func pause(active bool) {
	if !active {
		return
	}
	dire("\nAppuie sur Entree pour fermer.", "Press Enter to close.")
	bufio.NewReader(os.Stdin).ReadString('\n')
}

func main() {
	jeuArg := flag.String("jeu", "", "dossier du jeu / game folder")
	sortieArg := flag.String("sortie", "", "dossier de sortie / output folder")
	liste := flag.Bool("liste", false, "verifier seulement / check only")
	chercher := flag.Bool("chercher", false, "essayer beaucoup de chemins d'atlas (DLC) et dire lesquels existent / probe many atlas paths")
	verifier := flag.String("verifier", "", "comparer les pixels avec un dossier d'icones existant / compare pixels with an existing icon folder")
	sansPause := flag.Bool("sans-pause", false, "ne pas attendre Entree a la fin / do not wait for Enter")
	flag.Parse()
	attendre := !*sansPause
	if *jeuArg == "" && flag.NArg() > 0 { // dossier glisse sur le .exe
		*jeuArg = flag.Arg(0)
	}
	t0 := time.Now()
	fmt.Println("==============================================================")
	dire(" BardN Item Card %s - extraction des icones d'objets", "BardN Item Card %s - item icon extraction", version)
	fmt.Println("==============================================================")
	dire("Les icones appartiennent a Capcom : elles sont lues dans TON jeu, en lecture seule.",
		"Icons belong to Capcom: they are read from YOUR game, read-only.")
	fmt.Println()

	jeu := trouverJeu(*jeuArg)
	if jeu == "" {
		dire("Dossier du jeu introuvable automatiquement.", "Could not find the game folder automatically.")
		dire("Colle ici le dossier de Dragon's Dogma 2 (celui qui contient re_chunk_000.pak), puis Entree :",
			"Paste the Dragon's Dogma 2 folder (the one with re_chunk_000.pak), then press Enter:")
		fmt.Print("> ")
		l, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		jeu = trouverJeu(strings.TrimSpace(l))
		if jeu == "" {
			dire("Ce dossier ne contient pas re_chunk_000.pak. Astuce : glisse le dossier du jeu sur ce programme.",
				"This folder has no re_chunk_000.pak. Tip: drag the game folder onto this program.")
			pause(true)
			os.Exit(1)
		}
	}
	dire("Jeu trouve : %s", "Game found: %s", jeu)
	sortie := *sortieArg
	if sortie == "" {
		sortie = filepath.Join(jeu, "reframework", "images", "BardN_Item_Card", "icons")
	}

	principal := filepath.Join(jeu, "re_chunk_000.pak")
	chemins := []string{principal}
	for i := 1; i < 100; i++ {
		p := fmt.Sprintf("%s.patch_%03d.pak", principal, i)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			chemins = append(chemins, p)
		}
	}
	candidats := map[cleHash]string{}
	for nn := 0; nn < 40; nn++ {
		for _, d := range []string{"c01", "c00"} {
			c := fmt.Sprintf(dossierAtlas, d, d, nn)
			candidats[hashChemin(c)] = c
		}
	}
	if *chercher {
		for nn := 0; nn < 60; nn++ {
			for _, d := range []string{"c00", "c01", "c02", "c03", "c04", "c05", "c10", "c11", "c20"} {
				for _, dos := range []string{"item", "item2", "itemdlc", "item_dlc", "dlc/item"} {
					for _, pre := range []string{"ui01", "ui01dlc", "ui02"} {
						for _, suf := range []string{".251211553", ".760230703", ".241106027", ""} {
							c := fmt.Sprintf("natives/stm/gui/ui01/common/%s/%s/tex_%s%s_%02d_iam.tex%s", dos, d, pre, d, nn, suf)
							candidats[hashChemin(c)] = c
							c2 := fmt.Sprintf("natives/stm/gui/dlc/ui01/common/%s/%s/tex_%s%s_%02d_iam.tex%s", dos, d, pre, d, nn, suf)
							candidats[hashChemin(c2)] = c2
						}
					}
				}
			}
		}
	}
	type source struct {
		p *pak
		e entree
	}
	index := map[string]source{}
	var paks []*pak
	dire("Lecture de la table du jeu (quelques secondes)...", "Reading the game table (a few seconds)...")
	for _, c := range chemins {
		p, err := ouvrirPak(c)
		if err != nil {
			if c == principal {
				dire("Impossible de lire %s : %v", "Cannot read %s: %v", c, err)
				pause(attendre)
				os.Exit(1)
			}
			continue // petits fichiers de patch vides : ignores
		}
		paks = append(paks, p)
		for ch, e := range p.chercher(candidats) {
			index[ch] = source{p, e} // un patch plus recent remplace l'entree du .pak principal
		}
	}
	if *chercher {
		var noms []string
		for ch := range index {
			noms = append(noms, ch)
		}
		sortStrings(noms)
		for _, c := range noms {
			fmt.Println("TROUVE " + c)
		}
		fmt.Printf("%d atlas trouves sur %d chemins essayes\n", len(noms), len(candidats))
		return
	}
	if !*liste && *verifier == "" {
		if err := os.MkdirAll(sortie, 0o755); err != nil {
			dire("Impossible de creer %s : %v", "Cannot create %s: %v", sortie, err)
			pause(attendre)
			os.Exit(1)
		}
	}
	total, atlas, ecarts, compares := 0, 0, 0, 0
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	for nn := 0; nn < 40; nn++ {
		var src source
		var dossier string
		ok := false
		for _, d := range []string{"c01", "c00"} {
			if s, t := index[fmt.Sprintf(dossierAtlas, d, d, nn)]; t {
				src, dossier, ok = s, d, true
				break
			}
		}
		if !ok {
			continue
		}
		if *liste {
			fmt.Printf("atlas %02d : %s\n", nn, dossier)
			continue
		}
		tex, err := src.p.lire(src.e)
		var im *image.NRGBA
		if err == nil {
			im, err = decoderTex(tex)
		}
		if err != nil {
			fmt.Printf("atlas %02d : ECHEC / FAILED (%v)\n", nn, err)
			continue
		}
		atlas++
		cote := im.Rect.Dx() / 10
		n := 0
		for k := 0; k < 100; k++ {
			r, c := k/10, k%10
			sub := im.SubImage(image.Rect(c*cote, r*cote, (c+1)*cote, (r+1)*cote)).(*image.NRGBA)
			vide := true
			for y := 0; y < cote && vide; y++ {
				ligne := sub.Pix[y*sub.Stride : y*sub.Stride+cote*4]
				for x := 3; x < len(ligne); x += 4 {
					if ligne[x] != 0 {
						vide = false
						break
					}
				}
			}
			if vide {
				continue
			}
			case_ := image.NewNRGBA(image.Rect(0, 0, cote, cote))
			for y := 0; y < cote; y++ {
				copy(case_.Pix[y*case_.Stride:(y+1)*case_.Stride], sub.Pix[y*sub.Stride:y*sub.Stride+cote*4])
			}
			nom := fmt.Sprintf("%d.png", nn*1000+k)
			n++
			if *verifier != "" {
				compares++
				if !memesPixels(filepath.Join(*verifier, nom), case_) {
					ecarts++
					if ecarts <= 10 {
						fmt.Printf("  difference : %s\n", nom)
					}
				}
				continue
			}
			var buf bytes.Buffer
			if err := enc.Encode(&buf, case_); err == nil {
				err = os.WriteFile(filepath.Join(sortie, nom), buf.Bytes(), 0o644)
			}
			if err != nil {
				dire("Ecriture impossible : %v", "Cannot write: %v", err)
				pause(attendre)
				os.Exit(1)
			}
		}
		total += n
		fmt.Printf("atlas %02d : %d icones / icons (%s, %d px)\n", nn, n, dossier, cote)
	}
	for _, p := range paks {
		p.f.Close()
	}
	fmt.Println()
	if *verifier != "" {
		fmt.Printf("VERIFICATION : %d icones comparees, %d differences (%.1f s)\n", compares, ecarts, time.Since(t0).Seconds())
		if ecarts > 0 || compares == 0 {
			os.Exit(2)
		}
		return
	}
	if *liste {
		return
	}
	if atlas == 0 {
		dire("Aucune icone extraite : le format du jeu a peut-etre change.", "No icon extracted: the game format may have changed.")
		pause(attendre)
		os.Exit(1)
	}
	// marque lue par le mod (menu REFramework) : les icones sont presentes, plus d'avertissement
	if *sortieArg == "" {
		dossierData := filepath.Join(jeu, "reframework", "data", "BardN_Item_Card")
		marque := fmt.Sprintf("{\n  \"icones\": %d,\n  \"atlas\": %d,\n  \"programme\": \"%s\",\n  \"date\": \"%s\"\n}\n", total, atlas, version, time.Now().Format("2006-01-02 15:04"))
		if err := os.MkdirAll(dossierData, 0o755); err == nil {
			os.WriteFile(filepath.Join(dossierData, "icones.json"), []byte(marque), 0o644)
		}
	}
	dire("TERMINE : %d icones ecrites en %.0f s dans :", "DONE: %d icons written in %.0f s to:", total, time.Since(t0).Seconds())
	fmt.Println("   " + sortie)
	dire("Tu peux lancer le jeu : la fiche d'objet affichera les vraies icones.", "You can start the game: the item card will show the real icons.")
	dire("Jeu deja lance ? Menu REFramework > BardN Item Card > « Verifier de nouveau ».", "Game already running? REFramework menu > BardN Item Card > \"Check again\".")
	pause(attendre)
}

func memesPixels(chemin string, im *image.NRGBA) bool {
	f, err := os.Open(chemin)
	if err != nil {
		return false
	}
	defer f.Close()
	ref, err := png.Decode(f)
	if err != nil {
		return false
	}
	var pix []byte
	switch r := ref.(type) {
	case *image.NRGBA:
		if r.Bounds() != im.Bounds() {
			return false
		}
		pix = r.Pix
	case *image.RGBA: // PNG entierement opaque relu en RGBA : identique au non premultiplie
		if r.Bounds() != im.Bounds() {
			return false
		}
		pix = r.Pix
	default:
		return false
	}
	return bytes.Equal(pix, im.Pix)
}

func sortStrings(a []string) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j] < a[j-1]; j-- {
			a[j], a[j-1] = a[j-1], a[j]
		}
	}
}
