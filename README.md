# BardN Item Card — Icon Extractor

Small helper program for the Dragon's Dogma 2 mod **BardN Item Card**.
It reads the item icons from **your own copy of the game** (read-only) and writes them as PNG files for the mod.
The icons belong to Capcom, which is why they are not shipped with the mod.

## What it does
- Finds the game folder (next to the program, its parent folders, or your Steam libraries; otherwise asks for the path).
- Opens `re_chunk_000.pak` (and its patch files) **read-only**, decrypts the file table, decompresses the icon atlases (zstd) and decodes them (BC7).
- Writes one PNG per item to `<game>/reframework/images/BardN_Item_Card/icons/`, plus a small marker file `reframework/data/BardN_Item_Card/icones.json` that the mod reads.
- Nothing else: no network access, no other file is written, the game files are never modified.

## Build
Requires the [Go](https://go.dev/dl/) compiler, version 1.24 or later (this is the version of the Go programming language used to build the program, not the version of the mod).

```
go build -trimpath -ldflags "-s -w" -o BardN_Item_Card_Extraire_les_icones.exe .
```

Cross-compiling from Linux or macOS:

```
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o BardN_Item_Card_Extraire_les_icones.exe .
```

Dependency: `github.com/klauspost/compress` (zstd), pinned in `go.sum`.

## Files
- `main.go` — game folder search, .pak table reading and decryption, zstd, TEX parsing, PNG output, console messages (French / English).
- `bc7.go` — BC7 texture decoder (port of Pillow's `BcnDecode.c`, pixel-identical results).

## Options
- `-jeu <folder>` game folder, `-sortie <folder>` output folder, `-liste` list the atlases only, `-verifier <folder>` compare pixels with an existing icon folder, `-sans-pause` do not wait for Enter at the end.

---

# BardN Item Card — Extracteur d'icônes (FR)

Petit programme d'appoint pour le mod **BardN Item Card** de Dragon's Dogma 2. Il lit les icônes des objets dans **ta propre copie du jeu** (en lecture seule) et les écrit en PNG pour le mod. Les icônes appartiennent à Capcom : c'est pour ça qu'elles ne sont pas distribuées avec le mod.

Aucun accès réseau, aucun autre fichier écrit, les fichiers du jeu ne sont jamais modifiés. Compilation : voir la section « Build » ci-dessus. Il faut le compilateur Go en version 1.24 ou plus (c'est la version du langage de programmation Go, pas celle du mod).
