//go:build ignore

package main

import (
	"fmt"
	"time"

	"github.com/reusee/shmspmc"
)

type T = [49]byte

func main() {
	reader, err := shmspmc.New[T]("foo", false)
	if err != nil {
		panic(err)
	}
	defer reader.Close()

	n := 0
	t0 := time.Now()
	for {
		reader.Read()
		n++
		if n%1000_0000 == 0 {
			elapsed := time.Since(t0)
			fmt.Printf("read %v values in %v. %.3f ns per op\n",
				n,
				elapsed,
				float64(elapsed)/float64(n),
			)
		}
	}
}
