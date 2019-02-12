package nbt

import (
	"compress/gzip"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
)

// Prints a human-readable representation of an NBT file to stdout.
func Debug(compression Compression, in io.Reader) {
	new(debugState).init(compression, in).debug(0)
	return
}

type debugState struct {
	in io.Reader
}

func (d *debugState) init(compression Compression, in io.Reader) *debugState {
	if in == nil {
		panic(fmt.Errorf("nbt: Input stream is nil"))
	}

	switch compression {
	case Uncompressed:
		d.in = in
	case GZip:
		r, err := gzip.NewReader(in)
		if err != nil {
			panic(err)
		}
		d.in = r
	case ZLib:
		r, err := zlib.NewReader(in)
		if err != nil {
			panic(err)
		}
		d.in = r
	default:
		panic(fmt.Errorf("nbt: Unknown compression type: %d", compression))
	}

	return d
}

func (d *debugState) printf(indent int, format string, args ...interface{}) {
	fmt.Printf(fmt.Sprintf(fmt.Sprintf("%% %ds%%s\n", indent*4), " ", format), args...)
}

func (d *debugState) debug(indent int) bool {
	name, tag := d.readTag()
	if tag == tagEnd {
		d.printf(indent, "%s", tag)
		return false
	}
	d.printf(indent, "%s named [%d] %s:", tag, len(name), name)
	d.debugValue(indent+1, tag)
	return true
}

func (d *debugState) r(i interface{}) {
	err := binary.Read(d.in, binary.BigEndian, i)
	if err != nil {
		panic(err)
	}
}

// Returns the name of the tag that was read.
func (d *debugState) readTag() (string, Tag) {
	var tag Tag
	d.r(&tag)

	if tag == tagEnd {
		return "", tag
	}

	name := d.readString()

	return name, tag
}

func (d *debugState) readString() string {
	var length uint16
	d.r(&length)

	value := make([]byte, length)
	_, err := d.in.Read(value)
	if err != nil {
		panic(err)
	}

	return string(value)
}

func (d *debugState) debugValue(indent int, tag Tag) {
	switch tag {
	case tagByte:
		var value uint8
		d.r(&value)
		d.printf(indent, "0x%02x", value)

	case tagShort:
		var value uint16
		d.r(&value)
		d.printf(indent, "0x%04x", value)

	case tagInt:
		var value uint32
		d.r(&value)
		d.printf(indent, "0x%08x", value)

	case tagLong:
		var value uint64
		d.r(&value)
		d.printf(indent, "0x%016x", value)

	case tagFloat:
		var value float32
		d.r(&value)
		d.printf(indent, "%#v", value)

	case tagDouble:
		var value float64
		d.r(&value)
		d.printf(indent, "%#v", value)

	case tagByteArray:
		var length uint32
		d.r(&length)
		value := make([]byte, length)
		d.printf(indent, "Length: %d (0x%08x)", length, length)
		d.in.Read(value)
		d.printf(indent, "Value: %#v", value)

	case tagString:
		value := d.readString()
		d.printf(indent, "Length: %d", len(value))
		d.printf(indent, "Value: %s", value)

	case tagList:
		var inner Tag
		d.r(&inner)
		var length uint32
		d.r(&length)

		d.printf(indent, "Element type: %s", inner)
		d.printf(indent, "Length: %d", length)
		d.printf(indent, "Value: {")

		for i := uint32(0); i < length; i++ {
			d.debugValue(indent+1, inner)
		}

		d.printf(indent, "}")

	case tagCompound:
		d.printf(indent, "Values: {")
		for d.debug(indent + 1) {
		}
		d.printf(indent, "}")

	case tagIntArray:
		var length uint32
		d.r(&length)
		d.printf(indent, "Length: %d", length)
		d.printf(indent, "Values: {")
		for i := uint32(0); i < length; i++ {
			d.debugValue(indent+1, tagInt)
		}
		d.printf(indent, "}")

	case tagLongArray:
		var length uint32
		d.r(&length)
		d.printf(indent, "Length: %d", length)
		d.printf(indent, "Values: {")
		for i := uint32(0); i < length; i++ {
			d.debugValue(indent+1, tagLong)
		}
		d.printf(indent, "}")

	default:
		panic(fmt.Errorf("nbt: Unhandled tag: %s", tag))
	}
}
