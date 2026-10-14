// Command packaging produces the desktop-integration assets for releases:
//
//	go run packaging icons [outdir]
//		PNGs + AppIcon.icns + clouds.svg + clouds.desktop
//
//	go run packaging app -bin <file> -arch <arm64|amd64> -version <v> \
//		-icons <dir> -out <file.zip>
//		Zips a macOS clouds.app bundle (Info.plist + binary + icon). The zip
//		is written with explicit unix file modes, so the bundle is launchable
//		even when built on Windows.
//
// The cloud silhouette geometry is shared between the raster renderer and
// the emitted SVG — change it in one place (cloudParts + genIcons).
package main

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

// ------------------------------------------------------------------- icon --

// Cloud silhouette in a 100x100 box: three overlapping circles over a base
// rectangle (bottoms flush at y=76).
var cloudParts = []struct{ cx, cy, r float64 }{
	{32, 58, 18},
	{49.5, 48, 22},
	{67, 58, 18},
}

func inCloud(x, y float64) bool {
	for _, c := range cloudParts {
		dx, dy := x-c.cx, y-c.cy
		if dx*dx+dy*dy <= c.r*c.r {
			return true
		}
	}
	return x >= 32 && x <= 67 && y >= 58 && y <= 76
}

// renderCloud rasterizes the silhouette at the given square size, centered,
// with 2x2 supersampled edges.
func renderCloud(size int, fill color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	scale := float64(size) / 100
	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			cover := 0
			for sy := 0; sy < 2; sy++ {
				for sx := 0; sx < 2; sx++ {
					fx := float64(px) + 0.25 + float64(sx)*0.5
					fy := float64(py) + 0.25 + float64(sy)*0.5
					bx := 49.5 + (fx-float64(size)/2)/scale
					by := 51 + (fy-float64(size)/2)/scale
					if inCloud(bx, by) {
						cover++
					}
				}
			}
			if cover > 0 {
				img.SetNRGBA(px, py, color.NRGBA{R: fill.R, G: fill.G, B: fill.B, A: uint8(255 * cover / 4)})
			}
		}
	}
	return img
}

// icnsEntries maps PNG sizes to icns chunk types (modern PNG-based icns).
var icnsEntries = []struct {
	size int
	typ  string
}{
	{32, "ic11"}, {64, "ic12"}, {128, "ic07"}, {256, "ic08"},
	{512, "ic09"}, {1024, "ic10"}, {256, "ic13"}, {512, "ic14"},
}

func genIcons(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	fill := iconFill
	pngs := map[int][]byte{}
	for _, size := range []int{16, 32, 64, 128, 256, 512, 1024} {
		var buf bytes.Buffer
		if err := png.Encode(&buf, renderCloud(size, fill)); err != nil {
			return err
		}
		pngs[size] = buf.Bytes()
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("icon_%d.png", size)), buf.Bytes(), 0o644); err != nil {
			return err
		}
	}

	var body bytes.Buffer
	for _, e := range icnsEntries {
		data, ok := pngs[e.size]
		if !ok {
			continue
		}
		body.WriteString(e.typ)
		var l [4]byte
		binary.BigEndian.PutUint32(l[:], uint32(len(data)+8))
		body.Write(l[:])
		body.Write(data)
	}
	var icns bytes.Buffer
	icns.WriteString("icns")
	var tl [4]byte
	binary.BigEndian.PutUint32(tl[:], uint32(body.Len()+8))
	icns.Write(tl[:])
	icns.Write(body.Bytes())
	if err := os.WriteFile(filepath.Join(dir, "AppIcon.icns"), icns.Bytes(), 0o644); err != nil {
		return err
	}

	svg := `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">
  <g fill="#3b9eff">
    <circle cx="32" cy="58" r="18"/>
    <circle cx="49.5" cy="48" r="22"/>
    <circle cx="67" cy="58" r="18"/>
    <rect x="32" y="58" width="35" height="18"/>
  </g>
</svg>
`
	desktop := `[Desktop Entry]
Type=Application
Name=clouds
GenericName=Cloud resource browser
Comment=k9s-style terminal UI for AWS and GCP
TryExec=clouds
Exec=clouds
Icon=clouds
Terminal=true
Categories=System;Utility;ConsoleOnly;
Keywords=aws;gcp;cloud;terminal;
`
	if err := os.WriteFile(filepath.Join(dir, "clouds.svg"), []byte(svg), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "clouds.desktop"), []byte(desktop), 0o644); err != nil {
		return err
	}
	fmt.Printf("icons written to %s (icon_*.png, AppIcon.icns, clouds.svg, clouds.desktop)\n", dir)
	return nil
}

// -------------------------------------------------------------------- app --

func appBundle(args []string) error {
	fs := flag.NewFlagSet("app", flag.ExitOnError)
	binPath := fs.String("bin", "", "path to the darwin binary")
	arch := fs.String("arch", "", "amd64 or arm64")
	version := fs.String("version", "0.1.0", "CFBundle version string")
	iconsDir := fs.String("icons", "packaging/icons", "directory containing AppIcon.icns")
	out := fs.String("out", "", "output .zip path")
	fs.Parse(args)
	if *binPath == "" || *arch == "" || *out == "" {
		return fmt.Errorf("app: -bin, -arch and -out are required")
	}
	bin, err := os.ReadFile(*binPath)
	if err != nil {
		return err
	}
	icns, err := os.ReadFile(filepath.Join(*iconsDir, "AppIcon.icns"))
	if err != nil {
		return err
	}
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleDevelopmentRegion</key><string>en</string>
	<key>CFBundleDisplayName</key><string>clouds</string>
	<key>CFBundleExecutable</key><string>clouds</string>
	<key>CFBundleIconFile</key><string>AppIcon</string>
	<key>CFBundleIdentifier</key><string>io.github.GITHUB_REPO.clouds.%s</string>
	<key>CFBundleInfoDictionaryVersion</key><string>6.0</string>
	<key>CFBundleName</key><string>clouds</string>
	<key>CFBundlePackageType</key><string>APPL</string>
	<key>CFBundleShortVersionString</key><string>%s</string>
	<key>CFBundleVersion</key><string>%s</string>
	<key>LSMinimumSystemVersion</key><string>11.0</string>
	<key>NSHighResolutionCapable</key><true/>
</dict>
</plist>
`, *arch, *version, *version)

	f, err := os.Create(*out)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	addDir := func(name string) error {
		h := &zip.FileHeader{Name: name}
		h.SetMode(0o755)
		w, err := zw.CreateHeader(h)
		if err != nil {
			return err
		}
		_ = w
		return nil
	}
	addFile := func(name string, mode os.FileMode, data []byte) error {
		h := &zip.FileHeader{Name: name, Method: zip.Deflate}
		h.SetMode(mode)
		w, err := zw.CreateHeader(h)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}
	for _, d := range []string{
		"clouds.app/",
		"clouds.app/Contents/",
		"clouds.app/Contents/MacOS/",
		"clouds.app/Contents/Resources/",
	} {
		if err := addDir(d); err != nil {
			return err
		}
	}
	if err := addFile("clouds.app/Contents/Info.plist", 0o644, []byte(plist)); err != nil {
		return err
	}
	if err := addFile("clouds.app/Contents/MacOS/clouds", 0o755, bin); err != nil {
		return err
	}
	if err := addFile("clouds.app/Contents/Resources/AppIcon.icns", 0o644, icns); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	fmt.Printf("app bundle → %s (clouds.app, %s, v%s)\n", *out, *arch, *version)
	return nil
}

// -------------------------------------------------------------------- ico --

// iconFill is the single source of truth for the brand color.
var iconFill = color.NRGBA{R: 0x3B, G: 0x9E, B: 0xFF, A: 0xFF}

// dib32 encodes img as a 32bpp Windows DIB (BITMAPINFOHEADER + XOR pixels +
// AND mask), the payload format for non-PNG ICO/RT_ICON entries.
func dib32(img *image.NRGBA) []byte {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	andRow := ((w + 31) / 32) * 4
	buf := bytes.NewBuffer(make([]byte, 0, 40+h*(w*4+andRow)))
	b := binary.LittleEndian
	_ = binary.Write(buf, b, uint32(40))
	_ = binary.Write(buf, b, int32(w))
	_ = binary.Write(buf, b, int32(2*h))
	_ = binary.Write(buf, b, uint16(1))
	_ = binary.Write(buf, b, uint16(32))
	_ = binary.Write(buf, b, uint32(0))
	_ = binary.Write(buf, b, uint32(h*(w*4+andRow)))
	_ = binary.Write(buf, b, int32(0))
	_ = binary.Write(buf, b, int32(0))
	_ = binary.Write(buf, b, uint32(0))
	_ = binary.Write(buf, b, uint32(0))
	for y := h - 1; y >= 0; y-- { // DIBs are bottom-up
		for x := 0; x < w; x++ {
			c := img.NRGBAAt(x, y)
			buf.WriteByte(c.B)
			buf.WriteByte(c.G)
			buf.WriteByte(c.R)
			buf.WriteByte(c.A)
		}
	}
	buf.Write(make([]byte, h*andRow)) // AND mask; 32bpp transparency is the alpha channel
	return buf.Bytes()
}

// genICO writes a multi-size clouds.ico: 16/32/48 as BMP DIBs (broadest
// compatibility), 256 as PNG (Vista+).
func genICO(path string) error {
	type ent struct {
		w, h, bitcount byte
		data           []byte
	}
	var ents []ent
	for _, s := range []int{16, 32, 48} {
		ents = append(ents, ent{byte(s), byte(s), 32, dib32(renderCloud(s, iconFill))})
	}
	var png256 bytes.Buffer
	if err := png.Encode(&png256, renderCloud(256, iconFill)); err != nil {
		return err
	}
	ents = append(ents, ent{0, 0, 32, png256.Bytes()}) // 0x0 = 256 in ICO entries

	var out bytes.Buffer
	_ = binary.Write(&out, binary.LittleEndian, uint16(0))
	_ = binary.Write(&out, binary.LittleEndian, uint16(1))
	_ = binary.Write(&out, binary.LittleEndian, uint16(len(ents)))
	off := uint32(6 + 16*len(ents))
	for _, e := range ents {
		_ = binary.Write(&out, binary.LittleEndian, e.w)
		_ = binary.Write(&out, binary.LittleEndian, e.h)
		_ = binary.Write(&out, binary.LittleEndian, byte(0))
		_ = binary.Write(&out, binary.LittleEndian, byte(0))
		_ = binary.Write(&out, binary.LittleEndian, uint16(1))
		_ = binary.Write(&out, binary.LittleEndian, uint16(e.bitcount))
		_ = binary.Write(&out, binary.LittleEndian, uint32(len(e.data)))
		_ = binary.Write(&out, binary.LittleEndian, off)
		off += uint32(len(e.data))
	}
	for _, e := range ents {
		out.Write(e.data)
	}
	if err := os.WriteFile(path, out.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Printf("ico written to %s (16/32/48 BMP + 256 PNG)\n", path)
	return nil
}

// -------------------------------------------------------------------- syso --

// resEntry is one blob emitted into the COFF .rsrc section: an RT_ICON image
// (typ 3) or the RT_GROUP_ICON directory (typ 14).
type resEntry struct {
	typ, id        uint32
	w, h, colors   byte
	planes, bitcnt uint16
	data           []byte
}

// genSYSO writes a COFF object whose .rsrc section carries the ICO's entries
// (RT_ICON 1..n plus RT_GROUP_ICON 1). Saved as clouds_windows_<arch>.syso in
// the main package directory, the Go linker links it into every Windows build
// of that architecture automatically.
func genSYSO(icoPath, outPath string, machine uint16) error {
	raw, err := os.ReadFile(icoPath)
	if err != nil {
		return err
	}
	if len(raw) < 6 || binary.LittleEndian.Uint16(raw[2:]) != 1 {
		return fmt.Errorf("%s is not an ICO file", icoPath)
	}
	n := int(binary.LittleEndian.Uint16(raw[4:]))
	entries := make([]resEntry, 0, n+1)
	off := uint32(6)
	for i := 0; i < n; i++ {
		h := raw[off : off+16]
		off += 16
		size := binary.LittleEndian.Uint32(h[8:])
		dataOff := binary.LittleEndian.Uint32(h[12:])
		if uint64(dataOff)+uint64(size) > uint64(len(raw)) {
			return fmt.Errorf("ico entry %d out of bounds", i)
		}
		entries = append(entries, resEntry{
			typ: 3, id: uint32(i + 1), // RT_ICON
			w: h[0], h: h[1], colors: h[2],
			planes: binary.LittleEndian.Uint16(h[4:]),
			bitcnt: binary.LittleEndian.Uint16(h[6:]),
			data:   raw[dataOff : dataOff+size],
		})
	}
	// RT_GROUP_ICON: directory pointing at the RT_ICON ids.
	var grp bytes.Buffer
	_ = binary.Write(&grp, binary.LittleEndian, uint16(0))
	_ = binary.Write(&grp, binary.LittleEndian, uint16(1))
	_ = binary.Write(&grp, binary.LittleEndian, uint16(n))
	for i, e := range entries {
		_ = binary.Write(&grp, binary.LittleEndian, e.w)
		_ = binary.Write(&grp, binary.LittleEndian, e.h)
		_ = binary.Write(&grp, binary.LittleEndian, e.colors)
		_ = binary.Write(&grp, binary.LittleEndian, byte(0))
		_ = binary.Write(&grp, binary.LittleEndian, e.planes)
		_ = binary.Write(&grp, binary.LittleEndian, e.bitcnt)
		_ = binary.Write(&grp, binary.LittleEndian, uint32(len(e.data)))
		_ = binary.Write(&grp, binary.LittleEndian, uint16(i+1))
	}
	entries = append(entries, resEntry{typ: 14, id: 1, data: grp.Bytes()}) // RT_GROUP_ICON
	rsrc, rvaFields := buildRsrc(entries)

	// Relocation type that makes the linker rewrite the resource data-entry
	// RVAs (buildRsrc stores them as section-relative addends).
	var relType uint16
	switch machine {
	case 0x8664:
		relType = 0x0002 // IMAGE_REL_AMD64_ADDR32
	case 0xAA64:
		relType = 0x0001 // IMAGE_REL_ARM64_ADDR32
	default:
		return fmt.Errorf("unsupported machine 0x%X", machine)
	}

	// COFF object layout: file header (20 bytes) + one section header (40
	// bytes), padded to exactly 0x200 so the .rsrc raw data begins at 0x200
	// (PointerToRawData); then the .rsrc data, the relocation table, a
	// 3-record symbol table (@feat.00, .rsrc + its aux record) and an empty
	// string table.
	secPad := (len(rsrc) + 15) &^ 15
	relocOff := uint32(0x200 + secPad)
	symOff := relocOff + uint32(10*len(rvaFields))
	b := binary.LittleEndian
	hdr := make([]byte, 0x200)
	b.PutUint16(hdr[0:], machine)
	b.PutUint16(hdr[2:], 1)      // NumberOfSections
	b.PutUint32(hdr[4:], 0)      // TimeDateStamp (reproducible)
	b.PutUint32(hdr[8:], symOff) // PointerToSymbolTable
	b.PutUint32(hdr[12:], 3)     // NumberOfSymbols: 2 symbols + 1 aux record
	b.PutUint16(hdr[16:], 0)     // SizeOfOptionalHeader (object file)
	b.PutUint16(hdr[18:], 0)     // Characteristics (unused for object files)
	sh := hdr[20:60]
	copy(sh, ".rsrc")
	b.PutUint32(sh[8:], uint32(len(rsrc)))       // VirtualSize
	b.PutUint32(sh[12:], 0x1000)                 // VirtualAddress
	b.PutUint32(sh[16:], uint32(secPad))         // SizeOfRawData
	b.PutUint32(sh[20:], 0x200)                  // PointerToRawData
	b.PutUint32(sh[24:], relocOff)               // PointerToRelocations
	b.PutUint16(sh[32:], uint16(len(rvaFields))) // NumberOfRelocations
	// NB: the flags must not include DISCARDABLE - the Go linker skips
	// discardable object sections.
	b.PutUint32(sh[36:], 0x40000040) // INITIALIZED_DATA|READ

	sym1 := make([]byte, 18) // "@feat.00", abs
	copy(sym1, "@feat.00")
	b.PutUint32(sym1[8:], 1)
	b.PutUint16(sym1[12:], 0xFFFF) // N_ABS
	sym1[16] = 3                   // IMAGE_SYM_STATIC
	sym2 := make([]byte, 18)       // ".rsrc" section symbol
	copy(sym2, ".rsrc")
	b.PutUint16(sym2[12:], 1)
	sym2[16] = 3
	sym2[17] = 1 // one aux record
	aux := make([]byte, 18)
	b.PutUint32(aux[0:], uint32(secPad))

	out := bytes.NewBuffer(make([]byte, 0, symOff+3*18+4)) // + 2 syms, aux, string table
	out.Write(hdr)
	out.Write(rsrc)
	out.Write(make([]byte, secPad-len(rsrc)))
	for _, fo := range rvaFields {
		var rel [10]byte
		b.PutUint32(rel[0:], fo) // VirtualAddress of the field inside .rsrc
		b.PutUint32(rel[4:], 1)  // SymbolTableIndex: the ".rsrc" section symbol
		b.PutUint16(rel[8:], relType)
		out.Write(rel[:])
	}
	out.Write(sym1)
	out.Write(sym2)
	out.Write(aux)
	out.Write(make([]byte, 4)) // empty COFF string table
	return os.WriteFile(outPath, out.Bytes(), 0o644)
}

// buildRsrc lays out an IMAGE_RESOURCE_DIRECTORY tree for the entries:
// root -> type -> id -> language -> data entry -> blob. It returns the section
// bytes plus the offset of every IMAGE_RESOURCE_DATA_ENTRY.OffsetToData field.
// Those fields hold a section-relative blob offset as the implicit addend of a
// COFF relocation, so the linker rewrites each of them to <sectionRVA>+<blob>.
func buildRsrc(entries []resEntry) ([]byte, []uint32) {
	var types []uint32
	perType := map[uint32][]resEntry{}
	for _, e := range entries {
		if _, ok := perType[e.typ]; !ok {
			types = append(types, e.typ)
		}
		perType[e.typ] = append(perType[e.typ], e)
	}

	type key struct{ t, id uint32 }
	off := uint32(16 + 8*len(types))
	typeDir := map[uint32]uint32{}
	for _, t := range types {
		typeDir[t] = off
		off += 16 + 8*uint32(len(perType[t]))
	}
	idDir, dataEnt, blob := map[key]uint32{}, map[key]uint32{}, map[key]uint32{}
	for _, t := range types {
		for i := range perType[t] {
			idDir[key{t, uint32(i + 1)}] = off
			off += 24 // directory header + a single language entry
		}
	}
	for _, t := range types {
		for i := range perType[t] {
			dataEnt[key{t, uint32(i + 1)}] = off
			off += 16
		}
	}
	for _, t := range types {
		for i := range perType[t] {
			off = (off + 3) &^ 3
			blob[key{t, uint32(i + 1)}] = off
			off += uint32(len(perType[t][i].data))
		}
	}

	buf := make([]byte, off)
	var rvaFields []uint32
	p16 := func(o uint32, v uint16) { binary.LittleEndian.PutUint16(buf[o:], v) }
	p32 := func(o uint32, v uint32) { binary.LittleEndian.PutUint32(buf[o:], v) }
	dir := func(o uint32, ents [][2]uint32) {
		p16(o+12, 0)                 // named entries
		p16(o+14, uint16(len(ents))) // id entries
		for i, e := range ents {
			p32(o+16+uint32(i)*8, e[0])
			p32(o+20+uint32(i)*8, e[1])
		}
	}
	const lang = uint32(0x409) // en-US
	sub := func(o uint32) uint32 { return o | 0x80000000 }

	root := make([][2]uint32, 0, len(types))
	for _, t := range types {
		root = append(root, [2]uint32{t, sub(typeDir[t])})
	}
	dir(0, root)
	for _, t := range types {
		ids := make([][2]uint32, 0, len(perType[t]))
		for i := range perType[t] {
			ids = append(ids, [2]uint32{uint32(i + 1), sub(idDir[key{t, uint32(i + 1)}])})
		}
		dir(typeDir[t], ids)
		for i := range perType[t] {
			k := key{t, uint32(i + 1)}
			// The language level's entries point straight at the data entry:
			// the resource tree has exactly three directory levels,
			// type -> name/id -> language, then IMAGE_RESOURCE_DATA_ENTRY.
			dir(idDir[k], [][2]uint32{{lang, dataEnt[k]}})
			p32(dataEnt[k], blob[k]) // section-relative; patched via reloc below
			p32(dataEnt[k]+4, uint32(len(perType[t][i].data)))
			rvaFields = append(rvaFields, dataEnt[k])
			copy(buf[blob[k]:], perType[t][i].data)
		}
	}
	return buf, rvaFields
}

// ------------------------------------------------------------------- main --

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, `usage:
  go run ./packaging icons [dir]                     PNGs + AppIcon.icns + svg + .desktop
  go run ./packaging ico [dir]                       clouds.ico (Windows icon)
  go run ./packaging syso -arch amd64|arm64 [-ico F] [-out F]
                                                     COFF resource object for the exe icon`)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "icons":
		dir := "packaging/icons"
		if len(os.Args) > 2 {
			dir = os.Args[2]
		}
		err = genIcons(dir)
	case "ico":
		dir := "packaging/icons"
		if len(os.Args) > 2 {
			dir = os.Args[2]
		}
		err = genICO(filepath.Join(dir, "clouds.ico"))
	case "syso":
		fs := flag.NewFlagSet("syso", flag.ExitOnError)
		icoPath := fs.String("ico", filepath.Join("packaging", "icons", "clouds.ico"), "input .ico file")
		arch := fs.String("arch", "amd64", "amd64 or arm64")
		out := fs.String("out", "", "output path (default clouds_windows_<arch>.syso)")
		fs.Parse(os.Args[2:])
		var machine uint16
		switch *arch {
		case "amd64":
			machine = 0x8664
		case "arm64":
			machine = 0xAA64
		default:
			err = fmt.Errorf("unsupported arch %q", *arch)
		}
		if err == nil {
			outPath := *out
			if outPath == "" {
				outPath = "clouds_windows_" + *arch + ".syso"
			}
			err = genSYSO(*icoPath, outPath, machine)
			if err == nil {
				fmt.Printf("syso written to %s (%s)\n", outPath, *arch)
			}
		}
	case "app":
		err = appBundle(os.Args[2:])
	default:
		err = fmt.Errorf("unknown subcommand %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "packaging:", err)
		os.Exit(1)
	}
}
