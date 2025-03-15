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

		// write if the node is the last node in this path (terminal)
		// and the frequency of this node
		if err := write(combineTerminalFrequency(n)); err != nil {
			return err
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
			} else {
				diff := offset - prevChildOffset
				if err := writeVarint(w, uint64(diff)); err != nil {
					return err
				}
			}

			offset += int64(nodeSize(child))
			prevChildOffset = offset

			if err := serializeNode(child); err != nil {
				return nil
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

// writeVarint writes a uint64 value using variable-length encoding.
// Ref: https://protobuf.dev/programming-guides/encoding/
func writeVarint(w io.Writer, value uint64) error {
	var varintBuffer [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(varintBuffer[:], value) // [:] makes it a slice, wtf?
	_, err := w.Write(varintBuffer[:n])
	return err
}

// this packs the frequency and the terminal bool into 1 uint32.
// is this really a needed "optimization"? idk, but its a possible one.
func combineTerminalFrequency(n *Node) uint32 {
	result := n.Frequency & 0x7FFFFFFF
	if n.Terminal {
		result |= 0x80000000
	}

	return uint32(result)
}

// Size of a node (in bytes)
func nodeSize(node *Node) int {
	size := 0

	size += varintSize(uint64(len(node.Part)))
	size += len(node.Part)

	// Child count (varint)
	childCount := len(node.Children)
	size += varintSize(uint64(childCount))

	// Combined Terminal and Frequency (uint32)
	size += 4

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
