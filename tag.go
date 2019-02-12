package nbt

import "fmt"

// All tags are big endian.

type Tag byte

const (
	tagEnd       Tag = iota // No payload, no name.
	tagByte                 // Signed 8 bit integer.
	tagShort                // Signed 16 bit integer.
	tagInt                  // Signed 32 bit integer.
	tagLong                 // Signed 64 bit integer.
	tagFloat                // IEEE 754-2008 32 bit floating point number.
	tagDouble               // IEEE 754-2008 64 bit floating point number.
	tagByteArray            // size tagInt, then payload [size]byte.
	tagString               // length tagShort, then payload (utf-8) string (of length length).
	tagList                 // tagID tagByte, length tagInt, then payload [length]tagID.
	tagCompound             // { tagID tagByte, name tagString, payload tagID }... tagEnd
	tagIntArray             // size tagInt, then payload [size]tagInt
	tagLongArray
)

func (tag Tag) String() string {
	name := "Unknown"
	switch tag {
	case tagEnd:
		name = "TAG_End"
	case tagByte:
		name = "TAG_Byte"
	case tagShort:
		name = "TAG_Short"
	case tagInt:
		name = "TAG_Int"
	case tagLong:
		name = "TAG_Long"
	case tagFloat:
		name = "TAG_Float"
	case tagDouble:
		name = "TAG_Double"
	case tagByteArray:
		name = "TAG_Byte_Array"
	case tagString:
		name = "TAG_String"
	case tagList:
		name = "TAG_List"
	case tagCompound:
		name = "TAG_Compound"
	case tagIntArray:
		name = "TAG_Int_Array"
	case tagLongArray:
		name = "TAG_Long_Array"
	}
	return fmt.Sprintf("%s (0x%02x)", name, byte(tag))
}

type Compression byte

const (
	Uncompressed Compression = iota
	GZip
	ZLib
)
