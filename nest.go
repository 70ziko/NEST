package nest

import (
    "encoding/binary"
    "errors"
    "fmt"
    "io"
    "os"
    "math/rand"
    "time"
)

type ImageWriter interface {
    Write(writer io.Writer) error
}

type ImageReader interface {
    Read(reader io.Reader) error
}

type FileHeader struct {
    Magic       [4]byte
    Version     uint16
    Width       uint32
    Height      uint32
    TileSize    uint16
    NestedCount uint32
}

type PixeLink struct {
    R, G, B   byte
    NestedIdx uint32
}

type Tile struct {
    PixeLinks []PixeLink
}

type NestedImage struct {
    Width  uint16
    Height uint16
    Data   []byte
}

type NestedImageFile struct {
    Header       FileHeader
    MainImage    [][]PixeLink
    NestedImages []NestedImage
}

const MAGIC = "NEST"

func (nif *NestedImageFile) Write(writer io.Writer) error {
    if err := binary.Write(writer, binary.LittleEndian, &nif.Header); err != nil {
        return fmt.Errorf("failed to write header: %w", err)
    }

    for y := 0; y < len(nif.MainImage); y += int(nif.Header.TileSize) {
        for x := 0; x < len(nif.MainImage[0]); x += int(nif.Header.TileSize) {
            tile := nif.extractTile(x, y, int(nif.Header.TileSize))
            if err := binary.Write(writer, binary.LittleEndian, tile); err != nil {
                return fmt.Errorf("failed to write tile at (%d, %d): %w", x, y, err)
            }
        }
    }

    for i, img := range nif.NestedImages {
        if err := img.Write(writer); err != nil {
            return fmt.Errorf("failed to write nested image %d: %w", i, err)
        }
    }

    return nil
}

func (nif *NestedImageFile) Read(reader io.Reader) error {
    if err := binary.Read(reader, binary.LittleEndian, &nif.Header); err != nil {
        return fmt.Errorf("failed to read header: %w", err)
    }

    if string(nif.Header.Magic[:]) != MAGIC {
        return errors.New("invalid file format")
    }

    nif.MainImage = make([][]PixeLink, nif.Header.Height)
    for i := range nif.MainImage {
        nif.MainImage[i] = make([]PixeLink, nif.Header.Width)
    }

    tileSize := int(nif.Header.TileSize)
    for y := 0; y < int(nif.Header.Height); y += tileSize {
        for x := 0; x < int(nif.Header.Width); x += tileSize {
            tile := make([]PixeLink, tileSize*tileSize)
            if err := binary.Read(reader, binary.LittleEndian, &tile); err != nil {
                return fmt.Errorf("failed to read tile at (%d, %d): %w", x, y, err)
            }
            nif.fillTile(tile, x, y, tileSize)
        }
    }

    nif.NestedImages = make([]NestedImage, nif.Header.NestedCount)
    for i := range nif.NestedImages {
        if err := nif.NestedImages[i].Read(reader); err != nil {
            return fmt.Errorf("failed to read nested image %d: %w", i, err)
        }
    }

    return nil
}

func (nif *NestedImageFile) extractTile(x, y, size int) []PixeLink {
    tile := make([]PixeLink, 0, size*size)
    for j := y; j < y+size && j < len(nif.MainImage); j++ {
        for i := x; i < x+size && i < len(nif.MainImage[j]); i++ {
            tile = append(tile, nif.MainImage[j][i])
        }
    }
    return tile
}

func (nif *NestedImageFile) fillTile(tile []PixeLink, x, y, tileSize int) {
    for j := 0; j < tileSize && y+j < len(nif.MainImage); j++ {
        for i := 0; i < tileSize && x+i < len(nif.MainImage[y+j]); i++ {
            nif.MainImage[y+j][x+i] = tile[j*tileSize+i]
        }
    }
}

func (ni *NestedImage) Write(writer io.Writer) error {
    if err := binary.Write(writer, binary.LittleEndian, ni.Width); err != nil {
        return fmt.Errorf("failed to write nested image width: %w", err)
    }
    if err := binary.Write(writer, binary.LittleEndian, ni.Height); err != nil {
        return fmt.Errorf("failed to write nested image height: %w", err)
    }
    if _, err := writer.Write(ni.Data); err != nil {
        return fmt.Errorf("failed to write nested image data: %w", err)
    }
    return nil
}

func (ni *NestedImage) Read(reader io.Reader) error {
    if err := binary.Read(reader, binary.LittleEndian, &ni.Width); err != nil {
        return fmt.Errorf("failed to read nested image width: %w", err)
    }
    if err := binary.Read(reader, binary.LittleEndian, &ni.Height); err != nil {
        return fmt.Errorf("failed to read nested image height: %w", err)
    }
    ni.Data = make([]byte, ni.Width*ni.Height*3) // Assuming RGB format
    if _, err := io.ReadFull(reader, ni.Data); err != nil {
        return fmt.Errorf("failed to read nested image data: %w", err)
    }
    return nil
}

func WriteNestedImageFile(filename string, nif *NestedImageFile) error {
    file, err := os.Create(filename)
    if err != nil {
        return fmt.Errorf("failed to create file: %w", err)
    }
    defer file.Close()

    return nif.Write(file)
}

func ReadNestedImageFile(filename string) (*NestedImageFile, error) {
    file, err := os.Open(filename)
    if err != nil {
        return nil, fmt.Errorf("failed to open file: %w", err)
    }
    defer file.Close()

    nif := &NestedImageFile{}
    if err := nif.Read(file); err != nil {
        return nil, err
    }

    return nif, nil
}

func generateSampleMainImage(width, height int) [][]PixeLink {
    rant := rand.New(rand.NewSource(time.Now().UnixNano()))
    mainImage := make([][]PixeLink, height)
    for y := range mainImage {
        mainImage[y] = make([]PixeLink, width)
        for x := range mainImage[y] {
            mainImage[y][x] = PixeLink{
                R:         byte(rant.Intn(256)),
                G:         byte(rant.Intn(256)),
                B:         byte(rant.Intn(256)),
                NestedIdx: uint32(rant.Intn(6)), // 0-5, where 0 means no nested image
            }
        }
    }
    return mainImage
}

func generateSampleNestedImages(count int) []NestedImage {
    rant := rand.New(rand.NewSource(time.Now().UnixNano()))
    nestedImages := make([]NestedImage, count)
    for i := range nestedImages {
        width := uint16(rant.Intn(100) + 50)  // Random width between 50 and 149
        height := uint16(rant.Intn(100) + 50) // Random height between 50 and 149
        nestedImages[i] = NestedImage{
            Width:  width,
            Height: height,
            Data:   generateRandomImageData(int(width), int(height)),
        }
    }
    return nestedImages
}

func generateRandomImageData(width, height int) []byte {
    size := width * height * 3 // 3 bytes per pixel for RGB
    data := make([]byte, size)
    for i := 0; i < size; i++ {
        data[i] = byte(rand.Intn(256))
    }
    return data
}

func main() {
    nif := &NestedImageFile{
        Header: FileHeader{
            Magic:       [4]byte{'N', 'E', 'S', 'T'},
            Version:     1,
            Width:       1024,
            Height:      768,
            TileSize:    256,
            NestedCount: 5,
        },
        MainImage:    generateSampleMainImage(1024, 768),
        NestedImages: generateSampleNestedImages(5),
    }

    err := WriteNestedImageFile("sample.nest", nif)
    if err != nil {
        fmt.Printf("Error writing file: %v\n", err)
        return
    }

    readNif, err := ReadNestedImageFile("sample.nest")
    if err != nil {
        fmt.Printf("Error reading file: %v\n", err)
        return
    }

    fmt.Printf("Read main image dimensions: %dx%d\n", len(readNif.MainImage[0]), len(readNif.MainImage))
    fmt.Printf("Read %d nested images\n", len(readNif.NestedImages))

    verifyNestedImageFile(nif, readNif)
}

func verifyNestedImageFile(original, read *NestedImageFile) {
    fmt.Println("Verifying NestedImageFile...")

    if original.Header != read.Header {
        fmt.Println("Header mismatch!")
    } else {
        fmt.Println("Header matches.")
    }

    if len(original.MainImage) != len(read.MainImage) || len(original.MainImage[0]) != len(read.MainImage[0]) {
        fmt.Println("Main image dimensions mismatch!")
    } else {
        fmt.Println("Main image dimensions match.")
    }

    if len(original.NestedImages) != len(read.NestedImages) {
        fmt.Println("Number of nested images mismatch!")
    } else {
        fmt.Println("Number of nested images matches.")
    }
}
