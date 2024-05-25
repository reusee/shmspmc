package shmspmc

import (
	"encoding/binary"
	"testing"
)

func TestFile(t *testing.T) {
	name := "test-shmspmc"
	writer, err := New[[64]byte](name, true)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := New[[64]byte](name, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		reader.Close()
		writer.Close()
	})

	for i := 1; i < 1000_0000; i++ {
		var data [64]byte
		binary.PutUvarint(data[:], uint64(i))
		if err := writer.Write(data); err != nil {
			t.Fatal(err)
		}
		got := writer.Read()
		if got != data {
			t.Fatal()
		}
		got = reader.Read()
		if got != data {
			t.Fatal()
		}
	}

}

func BenchmarkReadWrite(b *testing.B) {
	name := "bench-shmspmc"
	writer, err := New[[64]byte](name, true)
	if err != nil {
		b.Fatal(err)
	}
	reader, err := New[[64]byte](name, false)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		reader.Close()
		writer.Close()
	})
	b.ResetTimer()

	for i := 1; i < b.N; i++ {
		var data [64]byte
		binary.PutUvarint(data[:], uint64(i))
		if err := writer.Write(data); err != nil {
			b.Fatal(err)
		}
		got := reader.Read()
		if got != data {
			b.Fatal()
		}
	}
}
