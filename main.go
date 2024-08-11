package main

import (
    "encoding/binary"
    "errors"
    "fmt"
    "io"
    "os"
)

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

const MAGIC = "NEST"

func WriteFile(filename string, mainImage [][]PixeLink, nestedImages []NestedImage) error {
    file, err := os.Create(filename)
    if err != nil {
        return fmt.Errorf("failed to create file: %w", err)
    }
    defer file.Close()

    if len(mainImage) == 0 || len(mainImage[0]) == 0 {
        return errors.New("main image is empty")
    }

    header := FileHeader{
        Version:     1,
        Width:       uint32(len(mainImage[0])),
        Height:      uint32(len(mainImage)),
        TileSize:    256,
        NestedCount: uint32(len(nestedImages)),
    }
    copy(header.Magic[:], MAGIC)

    if err := binary.Write(file, binary.LittleEndian, &header); err != nil {
        return fmt.Errorf("failed to write header: %w", err)
    }

    for y := 0; y < len(mainImage); y += int(header.TileSize) {
        for x := 0; x < len(mainImage[0]); x += int(header.TileSize) {
            tile := extractTile(mainImage, x, y, int(header.TileSize))
            if err := binary.Write(file, binary.LittleEndian, tile); err != nil {
                return fmt.Errorf("failed to write tile at (%d, %d): %w", x, y, err)
            }
        }
    }

    for i, img := range nestedImages {
        if err := binary.Write(file, binary.LittleEndian, img.Width); err != nil {
            return fmt.Errorf("failed to write nested image %d width: %w", i, err)
        }
        if err := binary.Write(file, binary.LittleEndian, img.Height); err != nil {
            return fmt.Errorf("failed to write nested image %d height: %w", i, err)
        }
        if _, err := file.Write(img.Data); err != nil {
            return fmt.Errorf("failed to write nested image %d data: %w", i, err)
        }
    }

    return nil
}

func extractTile(image [][]PixeLink, x, y, size int) []PixeLink {
    tile := make([]PixeLink, 0, size*size)
    for j := y; j < y+size && j < len(image); j++ {
        for i := x; i < x+size && i < len(image[j]); i++ {
            tile = append(tile, image[j][i])
        }
    }
    return tile
}

func ReadFile(filename string) ([][]PixeLink, []NestedImage, error) {
    file, err := os.Open(filename)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to open file: %w", err)
    }
    defer file.Close()

    var header FileHeader
    if err := binary.Read(file, binary.LittleEndian, &header); err != nil {
        return nil, nil, fmt.Errorf("failed to read header: %w", err)
    }

    if string(header.Magic[:]) != MAGIC {
        return nil, nil, errors.New("invalid file format")
    }

    mainImage := make([][]PixeLink, header.Height)
    for i := range mainImage {
        mainImage[i] = make([]PixeLink, header.Width)
    }

    tileSize := int(header.TileSize)
    for y := 0; y < int(header.Height); y += tileSize {
        for x := 0; x < int(header.Width); x += tileSize {
            tile := make([]PixeLink, tileSize*tileSize)
            if err := binary.Read(file, binary.LittleEndian, &tile); err != nil {
                return nil, nil, fmt.Errorf("failed to read tile at (%d, %d): %w", x, y, err)
            }
            fillTile(mainImage, tile, x, y, tileSize)
        }
    }

    nestedImages := make([]NestedImage, header.NestedCount)
    for i := range nestedImages {
        if err := binary.Read(file, binary.LittleEndian, &nestedImages[i].Width); err != nil {
            return nil, nil, fmt.Errorf("failed to read nested image %d width: %w", i, err)
        }
        if err := binary.Read(file, binary.LittleEndian, &nestedImages[i].Height); err != nil {
            return nil, nil, fmt.Errorf("failed to read nested image %d height: %w", i, err)
        }
        nestedImages[i].Data = make([]byte, nestedImages[i].Width*nestedImages[i].Height*3) // Assuming RGB format
        if _, err := io.ReadFull(file, nestedImages[i].Data); err != nil {
            return nil, nil, fmt.Errorf("failed to read nested image %d data: %w", i, err)
        }
    }

    return mainImage, nestedImages, nil
}

func fillTile(mainImage [][]PixeLink, tile []PixeLink, x, y, tileSize int) {
    for j := 0; j < tileSize && y+j < len(mainImage); j++ {
        for i := 0; i < tileSize && x+i < len(mainImage[y+j]); i++ {
            mainImage[y+j][x+i] = tile[j*tileSize+i]
        }
    }
}

func main() {
    mainImage := generateSampleMainImage(1024, 768)
    nestedImages := generateSampleNestedImages(5)

    err := WriteFile("sample.nest", mainImage, nestedImages)
    if err != nil {
        fmt.Printf("Error writing file: %v\n", err)
        return
    }

    readMainImage, readNestedImages, err := ReadFile("sample.nest")
    if err != nil {
        fmt.Printf("Error reading file: %v\n", err)
        return
    }

    fmt.Printf("Read main image dimensions: %dx%d\n", len(readMainImage[0]), len(readMainImage))
    fmt.Printf("Read %d nested images\n", len(readNestedImages))
}