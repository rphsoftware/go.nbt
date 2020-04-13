package nbt

import (
	"compress/gzip"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
)

func Unmarshal(compression Compression, in io.Reader, v interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if s, ok := r.(string); ok {
				err = fmt.Errorf(s)
			} else {
				err = r.(error)
			}
		}
	}()
	new(decodeState).init(compression, in).unmarshal(v)
	in.Close()
	return
}

type decodeState struct {
	in io.Reader
}

func (d *decodeState) init(compression Compression, in io.Reader) *decodeState {
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

func (d *decodeState) unmarshal(v interface{}) {
	_, tag := d.readTag()
	d.readValue(tag, reflect.ValueOf(v).Elem())
}

func (d *decodeState) r(i interface{}) {
	err := binary.Read(d.in, binary.BigEndian, i)
	if err != nil {
		panic(err)
	}
}

// Returns the name of the tag that was read.
func (d *decodeState) readTag() (string, Tag) {
	var tag Tag
	d.r(&tag)

	if tag == tagEnd {
		return "", tag
	}

	name := d.readString()

	return name, tag
}

func (d *decodeState) allocate(tag Tag) reflect.Value {
	switch tag {
	case tagByte:
		return reflect.ValueOf(new(int8)).Elem()
	case tagShort:
		return reflect.ValueOf(new(int16)).Elem()
	case tagInt:
		return reflect.ValueOf(new(int32)).Elem()
	case tagLong:
		return reflect.ValueOf(new(int64)).Elem()
	case tagFloat:
		return reflect.ValueOf(new(float32)).Elem()
	case tagDouble:
		return reflect.ValueOf(new(float64)).Elem()
	case tagByteArray:
		return reflect.ValueOf(new([]byte)).Elem()
	case tagString:
		return reflect.ValueOf(new(string)).Elem()
	case tagList:
		return reflect.ValueOf(new([]interface{})).Elem()
	case tagCompound:
		return reflect.ValueOf(new(map[string]interface{})).Elem()
	case tagIntArray:
		return reflect.ValueOf(new([]int32)).Elem()
	case tagLongArray:
		return reflect.ValueOf(new([]int64)).Elem()
	}
	panic(fmt.Errorf("nbt: Unhandled tag %s", tag))
}

func (d *decodeState) readString() string {
	var length uint16
	d.r(&length)

	value := make([]byte, length)
	_, err := d.in.Read(value)
	if err != nil {
		panic(err)
	}

	return string(value)
}

func (d *decodeState) readValue(tag Tag, v reflect.Value) {
	switch v.Kind() {
	case reflect.Int, reflect.Uint:
		panic(fmt.Errorf("nbt: int and uint types are not supported for portability reasons. Try int32 or uint32."))
	case reflect.Interface:
		v.Set(d.allocate(tag))
		v = v.Elem()
	case reflect.Ptr:
		v.Set(reflect.New(v.Type().Elem()))
		v = v.Elem()
	}

	switch tag {
	case tagByte:
		var value uint8
		d.r(&value)
		switch v.Kind() {
		case reflect.Bool:
			v.SetBool(value != 0)
		case reflect.Int8:
			v.SetInt(int64(int8(value)))
		case reflect.Uint8:
			v.SetUint(uint64(value))
		default:
			panic(fmt.Errorf("nbt: Tag is %s, but I don't know how to put that in a %s!", tag, v.Kind()))
		}

	case tagShort:
		var value uint16
		d.r(&value)
		switch v.Kind() {
		case reflect.Int16:
			v.SetInt(int64(int16(value)))
		case reflect.Uint16:
			v.SetUint(uint64(value))
		default:
			panic(fmt.Errorf("nbt: Tag is %s, but I don't know how to put that in a %s!", tag, v.Kind()))
		}

	case tagInt:
		var value uint32
		d.r(&value)
		switch v.Kind() {
		case reflect.Int32:
			v.SetInt(int64(int16(value)))
		case reflect.Uint32:
			v.SetUint(uint64(value))
		default:
			panic(fmt.Errorf("nbt: Tag is %s, but I don't know how to put that in a %s!", tag, v.Kind()))
		}

	case tagLong:
		var value uint64
		d.r(&value)
		switch v.Kind() {
		case reflect.Int64:
			v.SetInt(int64(value))
		case reflect.Uint64:
			v.SetUint(value)
		default:
			panic(fmt.Errorf("nbt: Tag is %s, but I don't know how to put that in a %s!", tag, v.Kind()))
		}

	case tagFloat:
		var value float32
		d.r(&value)
		switch v.Kind() {
		case reflect.Float32:
			v.SetFloat(float64(value))
		default:
			panic(fmt.Errorf("nbt: Tag is %s, but I don't know how to put that in a %s!", tag, v.Kind()))
		}

	case tagDouble:
		var value float64
		d.r(&value)
		switch v.Kind() {
		case reflect.Float64:
			v.SetFloat(value)
		default:
			panic(fmt.Errorf("nbt: Tag is %s, but I don't know how to put that in a %s!", tag, v.Kind()))
		}

	case tagByteArray:
		var length uint32
		d.r(&length)

		switch v.Kind() {
		case reflect.Array, reflect.Slice:
			if v.Kind() == reflect.Array {
				if uint32(v.Len()) < length {
					panic(fmt.Errorf("nbt: Byte array is of length %d, but only the array given is only %d long!", length, v.Len()))
				}
			} else {
				if uint32(v.Len()) < length {
					v.Set(reflect.MakeSlice(v.Type(), int(length), int(length)))
				}
			}

			for i := 0; i < int(length); i++ {
				value := v.Index(i)
				d.readValue(tagByte, value)
			}

		default:
			panic(fmt.Errorf("nbt: Tag is %s, but I don't know how to put that in a %s!", tag, v.Kind()))
		}

	case tagString:
		switch v.Kind() {
		case reflect.String:
			v.SetString(d.readString())
		default:
			panic(fmt.Errorf("nbt: Tag is %s, but I don't know how to put that in a %s!", tag, v.Kind()))
		}

	case tagList:
		var inner Tag
		d.r(&inner)
		var length uint32
		d.r(&length)

		switch v.Kind() {
		case reflect.Slice:
			if uint32(v.Cap()) < length {
				v.Set(reflect.MakeSlice(v.Type(), 0, int(length)))
			} else {
				v.Set(v.Slice(0, 0))
			}
			kind := v.Type().Elem()

			var i uint32
			defer func() {
				if r := recover(); r != nil {
					panic(fmt.Errorf("%v\n\t\tat list index %d", r, i))
				}
			}()

			for i = 0; i < length; i++ {
				var value reflect.Value
				if kind.Kind() == reflect.Ptr {
					value = reflect.New(kind.Elem())
					d.readValue(inner, value.Elem())
				} else {
					if kind.Kind() == reflect.Interface {
						value = d.allocate(inner)
					} else {
						value = reflect.New(kind).Elem()
					}
					d.readValue(inner, value)
				}
				v.Set(reflect.Append(v, value))
			}

		default:
			panic(fmt.Errorf("nbt: Tag is %s, but I don't know how to put that in a %s!", tag, v.Kind()))
		}

	case tagCompound:
		switch v.Kind() {
		case reflect.Struct:
			fields := parseStruct(v)

			var name string
			defer func() {
				if r := recover(); r != nil {
					panic(fmt.Errorf("%v\n\t\tat struct field %#v", r, name))
				}
			}()

			for {
				var tag Tag
				name, tag = d.readTag()
				if tag == tagEnd {
					break
				}
				if field, ok := fields[name]; ok {
					d.readValue(tag, field)
				} else {
					panic(fmt.Errorf("nbt: Unhandled %s", tag))
				}
			}

		case reflect.Map:
			if v.IsNil() {
				v.Set(reflect.ValueOf(make(map[string]interface{})))
			}

			var name string
			defer func() {
				if r := recover(); r != nil {
					panic(fmt.Errorf("%v\n\t\tat struct field %#v", r, name))
				}
			}()

			for {
				var tag Tag
				name, tag = d.readTag()
				if tag == tagEnd {
					break
				}
				val := d.allocate(tag)
				d.readValue(tag, val)
				v.SetMapIndex(reflect.ValueOf(name), val)
			}

		default:
			panic(fmt.Errorf("nbt: Tag is %s, but I don't know how to put that in a %s!", tag, v.Kind()))
		}

	case tagIntArray:
		var length uint32
		d.r(&length)

		switch v.Kind() {
		case reflect.Array, reflect.Slice:
			if v.Kind() == reflect.Array {
				if uint32(v.Len()) < length {
					panic(fmt.Errorf("nbt: Int array is of length %d, but only the array given is only %d long!", length, v.Len()))
				}
			} else {
				if uint32(v.Len()) < length {
					v.Set(reflect.MakeSlice(v.Type(), int(length), int(length)))
				}
			}

			for i := 0; i < int(length); i++ {
				value := v.Index(i)
				d.readValue(tagInt, value)
			}

		default:
			panic(fmt.Errorf("nbt: Tag is %s, but I don't know how to put that in a %s!", tag, v.Kind()))
		}
	case tagLongArray:
		var length uint32
		d.r(&length)

		switch v.Kind() {
		case reflect.Array, reflect.Slice:
			if v.Kind() == reflect.Array {
				if uint32(v.Len()) < length {
					panic(fmt.Errorf("nbt: Int array is of length %d, but only the array given is only %d long!", length, v.Len()))
				}
			} else {
				if uint32(v.Len()) < length {
					v.Set(reflect.MakeSlice(v.Type(), int(length), int(length)))
				}
			}

			for i := 0; i < int(length); i++ {
				value := v.Index(i)
				d.readValue(tagLong, value)
			}

		default:
			panic(fmt.Errorf("nbt: Tag is %s, but I don't know how to put that in a %s!", tag, v.Kind()))
		}
	default:
		panic(fmt.Errorf("nbt: Unhandled tag: %s", tag))
	}
}
