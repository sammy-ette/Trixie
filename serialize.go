package main

import (
	"encoding/binary"
	"io"
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
		if n.Frequency > 0 {
			if err := writeVarint(w, uint64(0b1)); err != nil {
				return err
			}

			// write timestamp
			if err := write(n.Timestamp.Unix()); err != nil {
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

	if node.Frequency > 0 {
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
