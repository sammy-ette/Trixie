package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

func (t *Trie) Serialize(w io.Writer) error {
	var offset int64 = 0

	write := func(data interface{}) error {
		switch v := data.(type) {
		case []byte:
			n, err := w.Write(v)
			if err != nil {
				return err
			}
			offset += int64(n)
		default:
			err := binary.Write(w, binary.LittleEndian, data)
			if err != nil {
				return err
			}

			offset += int64(binary.Size(data))
		}
		return nil
	}

	var serializeNode func(n *Node) error
	serializeNode = func(n *Node) error {
		// write length of node Part, to know how long it is.
		if err := writeVarint(w, uint64(len(n.Part))); err != nil {
			return err
		}
		// and then write the part itself.
		if err := write([]byte(n.Part)); err != nil {
			return err
		}

		// VARIABLE SECTION
		// so, to minimize size, and to define some things.
		// if Frequency is > 1, up to this node is a full command string.
		if !n.Timestamp.IsZero() {
			if err := writeVarint(w, uint64(0b1)); err != nil {
				return err
			}

			// write timestamp
			if err := write(uint64(n.Timestamp.Unix())); err != nil {
				return err
			}

			// write the frequency of this node
			if err := writeVarint(w, uint64(n.Frequency)); err != nil {
				return err
			}

			// write node sequence
			if err := writeVarint(w, uint64(n.Sequence)); err != nil {
				return err
			}
		} else {
			if err := writeVarint(w, uint64(0b0)); err != nil {
				return err
			}
		}

		// now write stuff for the child nodes!
		// first is the amount of children
		if err := writeVarint(w, uint64(len(n.Children))); err != nil {
			return err
		}
		prevChildOffset := offset
		firstChild := true
		// and next up is writing the children...!
		for _, child := range n.Children {
			if firstChild {
				if err := writeVarint(w, uint64(offset)); err != nil {
					return err
				}
				firstChild = false
			} else {
				diff := offset - prevChildOffset
				if err := writeVarint(w, uint64(diff)); err != nil {
					return err
				}
			}

			offset += int64(nodeSize(child))
			prevChildOffset = offset

			if err := serializeNode(child); err != nil {
				return err
			}
		}

		return nil
	}

	// now lets actually start writing to the file (or our writer..)
	// trixie database header
	// first: magic
	if err := write(magic); err != nil {
		return err
	}
	// version
	if err := write([]byte(" v1 ")); err != nil {
		return err
	}

	return serializeNode(t.Root)
}

func (t *Trie) Deserialize(r io.Reader) error {
	read := func(data interface{}) error {
		return binary.Read(r, binary.LittleEndian, data)
	}

	readBytes := func(length int) ([]byte, error) {
		buf := make([]byte, length)
		_, err := io.ReadFull(r, buf)
		if err != nil {
			return nil, err
		}
		return buf, nil
	}

	readVarint := func() (uint64, error) {
		var buf [binary.MaxVarintLen64]byte
		var bytesRead int

		for {
			if bytesRead >= binary.MaxVarintLen64 {
				return 0, fmt.Errorf("varint too long")
			}

			_, err := r.Read(buf[bytesRead:bytesRead+1])
			if err != nil {
				return 0, err
			}

			bytesRead++

			if buf[bytesRead-1]&0x80 == 0 {
				val, _ := binary.Uvarint(buf[:bytesRead])
				return val, nil
			}
		}
	}

	// Header Check
	magicBytes := make([]byte, len(magic))
	if err := read(magicBytes); err != nil {
		return err
	}
	if string(magicBytes) != string(magic) {
		return fmt.Errorf("invalid magic bytes")
	}

	versionBytes := make([]byte, 4)
	if err := read(versionBytes); err != nil {
		return fmt.Errorf("invalid version")
	}
	if string(versionBytes) != " v1 " {
		return fmt.Errorf("invalid version")
	}

	var deserializeNode func() (*Node, error)
	deserializeNode = func() (*Node, error) {
		partLength, err := readVarint()
		if err != nil {
			panic(err)
			return nil, err
		}
		partBytes, err := readBytes(int(partLength))
		if err != nil {
			panic(err)
			return nil, err
		}
		part := string(partBytes)

		node := &Node{Part: part, Children: make(map[string]*Node)}

		// Variable Section
		hasFrequency, err := readVarint()
		if err != nil {
			return nil, err
		}
		if hasFrequency == 0b1 {
			var unixTime uint64
			if err := read(&unixTime); err != nil {
				panic(err)
				return nil, err
			}
			node.Timestamp = time.Unix(int64(unixTime), 0)

			frequency, err := readVarint()
			if err != nil {
				panic(err)
				return nil, err
			}
			node.Frequency = int(frequency)

			sequence, err := readVarint()
			if err != nil {
				panic(err)
				return nil, err
			}
			node.Sequence = uint32(sequence)
		}

		childCount, err := readVarint()
		if err != nil {
			panic(err)
			return nil, err
		}

		if childCount > 0 {
			//read and discard offsets, keeping backward compatibility
			for i := 0; i < int(childCount); i++ {
				_, err := readVarint()
				if err != nil {
					panic(err)
					return nil, err
				}

				childNode, err := deserializeNode()
				if err != nil {
					panic(err)
					return nil, err
				}
				node.Children[childNode.Part] = childNode
			}
		}

		return node, nil
	}

	root, err := deserializeNode()
	if err != nil {
		return err
	}

	t.Root = root
	return nil
}

func writeVarint(w io.Writer, value uint64) error {
	var varintBuffer [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(varintBuffer[:], value) // [:] makes it a slice, wtf?
	_, err := w.Write(varintBuffer[:n])
	return err
}

// Size of a node (in bytes)
func nodeSize(node *Node) int {
	size := 0

	size += varintSize(uint64(len(node.Part)))
	size += len(node.Part)

	size++ // Add 1 for the flag to know if the extra info (our timestamp check)
	// is present
	if !node.Timestamp.IsZero() {
		// +5 bytes, 1 for the boolean and 4 for the timestamp
		size += 5

		size += varintSize(uint64(node.Frequency))
		size += varintSize(uint64(node.Sequence))
	}
	// Child count (varint)
	childCount := len(node.Children)
	size += varintSize(uint64(childCount))

	// Child offsets (varints)
	prevOffset := 0
	i := 0
	for range node.Children {
		currentOffset := 0 //placeholder
		if i == 0 {
			size += varintSize(uint64(currentOffset))
		} else {
			// offset diff
			size += varintSize(uint64(currentOffset - prevOffset))
		}
		prevOffset = currentOffset
		i++
	}

	return size
}

func varintSize(value uint64) int {
	size := 1
	for value >= 128 {
		value >>= 7
		size++
	}
	return size
}
