//go:build ignore

package main

import (
	"encoding/binary"
	"fmt"
	"net/http"
	"time"

	"github.com/reusee/shmspmc"
	_ "net/http/pprof"
)

func init() {
	go func() {
		http.ListenAndServe(":8899", nil)
	}()
}

type T = [49]byte

func main() {
	writer, err := shmspmc.New[T]("foo", true)
	if err != nil {
		panic(err)
	}
	defer writer.Close()

	var data T
	t0 := time.Now()
	for i := uint64(1); ; i++ {
		binary.PutUvarint(data[:], i)
		if err := writer.Write(data); err != nil {
			panic(err)
		}
		if i%1000_0000 == 0 {
			elapsed := time.Since(t0)
			fmt.Printf("write %v values in %v. %.3f ns per op\n",
				i,
				elapsed,
				float64(elapsed)/float64(i),
			)
		}
	}
}
