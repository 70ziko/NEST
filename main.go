package main

import (
    "encoding/binary"
    "os"
    // "fmt"
)

type FileHeader struct {
    Magic       [4]byte  // Identifier for the file format, e.g., "NEST"
    Version     uint16   // Version of the file format
    Width       uint32   // Width of the main image
    Height      uint32   // Height of the main image
    TileSize    uint16   // Size of each tile (e.g., 256x256)
    NestedCount uint32   // Number of nested images
}

type PixeLink struct {
    R, G, B    byte    // RGB color values
    NestedIdx  uint32  // Index of the nested image (0 if no nested image)
}

type Tile struct {
    PixeLinks []PixeLink // PixeLinks in the tile, row by row
}

type NestedImage struct {
    Width  uint16
    Height uint16
    Data   []byte // Raw image data (e.g., JPEG or PNG)
}

const MAGIC = "NEST"

func WriteFile(filename string, mainImage [][]PixeLink, nestedImages []NestedImage) error {
    file, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer file.Close()

    header := FileHeader{
        Magic:       [4]byte{'N', 'E', 'S', 'T'},
        Version:     1,
        Width:       uint32(len(mainImage[0])),
        Height:      uint32(len(mainImage)),
        TileSize:    256,
        NestedCount: uint32(len(nestedImages)),
    }

    // Write header
    err = binary.Write(file, binary.LittleEndian, &header)
    if err != nil {
        return err
    }

    // Write main image data (tiles)
    for y := 0; y < len(mainImage); y += int(header.TileSize) {
        for x := 0; x < len(mainImage[0]); x += int(header.TileSize) {
            tile := extractTile(mainImage, x, y, int(header.TileSize))
            err = binary.Write(file, binary.LittleEndian, tile)
            if err != nil {
                return err
            }
        }
    }

    // Write nested images
    for _, img := range nestedImages {
        err = binary.Write(file, binary.LittleEndian, img.Width)
        if err != nil {
            return err
        }
        err = binary.Write(file, binary.LittleEndian, img.Height)
        if err != nil {
            return err
        }
        _, err = file.Write(img.Data)
        if err != nil {
            return err
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

// ReadFile function would be implemented similarly, reading the header first
// and then the tiles and nested images.