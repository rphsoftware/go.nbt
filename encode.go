package nbt

import (
	"compress/gzip"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
)

func Marshal(compression Compression, out io.Writer, v interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if s, ok := r.(string); ok {
				err = fmt.Errorf(s)
			} else {
				err = r.(error)
			}
		}
	}()

	if out == nil {
		panic(fmt.Errorf("nbt: Output stream is nil"))
	}

	switch compression {
	case Uncompressed:
		break
	case GZip:
		w := gzip.NewWriter(out)
		defer w.Close()
		out = w
	case ZLib:
		w := zlib.NewWriter(out)
		defer w.Close()
		out = w
	default:
		panic(fmt.Errorf("nbt: Unknown compression type: %d", compression))
	}

	writeRootTag(out, reflect.ValueOf(v))

	return
}

func writeRootTag(out io.Writer, v reflect.Value) {
	writeTag(out, "", v)
}

func w(out io.Writer, v interface{}) {
	err := binary.Write(out, binary.BigEndian, v)
	if err != nil {
		panic(err)
	}
}

func writeTag(out io.Writer, name string, v reflect.Value) {
	v = reflect.Indirect(v)
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Errorf("%v\n\t\tat struct field %#v", r, name))
		}
	}()
	for v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Bool:
		w(out, tagByte)
		writeValue(out, tagString, name)
		if v.Bool() {
			writeValue(out, tagByte, byte(1))
		} else {
			writeValue(out, tagByte, byte(0))
		}

	case reflect.Int8:
		w(out, tagByte)
		writeValue(out, tagString, name)
		writeValue(out, tagByte, int8(v.Int()))

	case reflect.Uint8:
		w(out, tagByte)
		writeValue(out, tagString, name)
		writeValue(out, tagByte, uint8(v.Uint()))

	case reflect.Int16:
		w(out, tagShort)
		writeValue(out, tagString, name)
		writeValue(out, tagShort, int16(v.Int()))

	case reflect.Uint16:
		w(out, tagShort)
		writeValue(out, tagString, name)
		writeValue(out, tagShort, uint16(v.Uint()))

	case reflect.Int32:
		w(out, tagInt)
		writeValue(out, tagString, name)
		writeValue(out, tagInt, int32(v.Int()))

	case reflect.Uint32:
		w(out, tagInt)
		writeValue(out, tagString, name)
		writeValue(out, tagInt, uint32(v.Uint()))

	case reflect.Int64:
		w(out, tagLong)
		writeValue(out, tagString, name)
		writeValue(out, tagLong, v.Int())

	case reflect.Uint64:
		w(out, tagLong)
		writeValue(out, tagString, name)
		writeValue(out, tagLong, v.Uint())

	case reflect.Float32:
		w(out, tagFloat)
		writeValue(out, tagString, name)
		writeValue(out, tagFloat, float32(v.Float()))

	case reflect.Float64:
		w(out, tagDouble)
		writeValue(out, tagString, name)
		writeValue(out, tagDouble, v.Float())

	case reflect.String:
		w(out, tagString)
		writeValue(out, tagString, name)
		writeValue(out, tagString, v.String())

	case reflect.Array:
		switch v.Type().Elem().Kind() {
		case reflect.Uint8:
			w(out, tagByteArray)
			writeValue(out, tagString, name)
			writeValue(out, tagByteArray, v.Slice(0, v.Len()).Bytes())

		case reflect.Int32, reflect.Uint32:
			w(out, tagIntArray)
			writeValue(out, tagString, name)
			for i := 0; i < v.Len(); i++ {
				writeValue(out, tagInt, v.Index(i).Interface())
			}

		case reflect.Int64, reflect.Uint64:
			w(out, tagLongArray)
			writeValue(out, tagString, name)
			for i := 0; i < v.Len(); i++ {
				writeValue(out, tagLong, v.Index(i).Interface())
			}

		default:
			panic(fmt.Errorf("nbt: Unhandled array type: %v", v.Type().Elem()))
		}

	case reflect.Slice:
		w(out, tagList)
		writeValue(out, tagString, name)
		writeList(out, v)

	case reflect.Map:
		w(out, tagCompound)
		writeValue(out, tagString, name)
		writeMap(out, v)

	case reflect.Struct:
		w(out, tagCompound)
		writeValue(out, tagString, name)
		writeCompound(out, v)

	default:
		panic(fmt.Errorf("nbt: Unhandled type: %v (%v)", v.Type(), v.Interface()))
	}
}

func writeValue(out io.Writer, tag Tag, v interface{}) {
	switch tag {
	case tagByte, tagShort, tagInt, tagLong, tagFloat, tagDouble:
		w(out, v)

	case tagString:
		w(out, uint16(len(v.(string))))
		_, err := out.Write([]byte(v.(string)))
		if err != nil {
			panic(err)
		}

	case tagByteArray:
		w(out, uint32(len(v.([]byte))))
		_, err := out.Write(v.([]byte))
		if err != nil {
			panic(err)
		}

	default:
		panic(fmt.Errorf("nbt: Unhandled tag: %s (%v)", tag, v))
	}
}

func writeList(out io.Writer, v reflect.Value) {
	var tag Tag
	mustConvertBool := false
	mustConvertMap := false
	switch v.Type().Elem().Kind() {
	case reflect.Bool:
		mustConvertBool = true
		fallthrough
	case reflect.Int8, reflect.Uint8:
		tag = tagByte

	case reflect.Int16, reflect.Uint16:
		tag = tagShort

	case reflect.Int32, reflect.Uint32:
		tag = tagInt

	case reflect.Int64, reflect.Uint64:
		tag = tagLong

	case reflect.Float32:
		tag = tagFloat

	case reflect.Float64:
		tag = tagDouble

	case reflect.String:
		tag = tagString

	case reflect.Array:
		switch v.Type().Elem().Elem().Kind() {
		case reflect.Uint8:
			tag = tagByteArray

		case reflect.Int32, reflect.Uint32:
			tag = tagIntArray

		case reflect.Int64, reflect.Uint64:
			tag = tagLongArray

		default:
			panic(fmt.Errorf("nbt: Unhandled array type: %v", v.Type().Elem().Elem()))
		}

	case reflect.Slice:
		tag = tagList

	case reflect.Map:
		mustConvertMap = true
		fallthrough
	case reflect.Struct:
		tag = tagCompound

	case reflect.Ptr: // TODO: Is there ever a case where tagCompound would be wrong here?
		tag = tagCompound

	default:
		panic(fmt.Errorf("nbt: Unhandled list element type: %v", v.Type().Elem()))
	}
	w(out, tag)
	w(out, uint32(v.Len()))

	var i int
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Errorf("%v\n\t\tat list index %d", r, i))
		}
	}()
	for i = 0; i < v.Len(); i++ {
		if mustConvertBool {
			if v.Index(i).Bool() {
				writeValue(out, tagByte, uint8(1))
			} else {
				writeValue(out, tagByte, uint8(0))
			}
		} else if tag == tagCompound {
			if mustConvertMap {
				writeMap(out, v.Index(i))
			} else {
				writeCompound(out, reflect.Indirect(v.Index(i)))
			}
		} else if tag == tagList {
			writeList(out, v.Index(i))
		} else if tag == tagByteArray {
			writeValue(out, tag, v.Index(i).Bytes())
		} else if tag == tagIntArray {
			for j := 0; j < v.Index(i).Len(); j++ {
				writeValue(out, tagInt, v.Index(i).Index(j).Interface())
			}
		} else if tag == tagLongArray {
			for j := 0; j < v.Index(i).Len(); j++ {
				writeValue(out, tagLong, v.Index(i).Index(j).Interface())
			}
		} else {
			writeValue(out, tag, v.Index(i).Interface())
		}
	}
}

func writeMap(out io.Writer, v reflect.Value) {
	for _, name := range v.MapKeys() {
		writeTag(out, name.String(), reflect.Indirect(v.MapIndex(name)))
	}
	w(out, tagEnd)
}

func writeCompound(out io.Writer, v reflect.Value) {
	v = reflect.Indirect(v)
	fields := parseStruct(v)

	for name, value := range fields {
		writeTag(out, name, value)
	}
	w(out, tagEnd)
}
