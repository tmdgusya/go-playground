package main

import (
	"fmt"
	"unsafe"
)

func main() {
	var k int = 10
	var y *int = &k

	fmt.Println("Value of k:", k)
	fmt.Println("Value of y:", y)

	var o int = *(*int)(y)

	fmt.Printf("Value of o: %d\n", o)

	k = 20

	fmt.Println("Value of k:", k)
	fmt.Println("Value of y:", y)
	fmt.Println("Value of o:", o)
	fmt.Println("has same addr? ", &k == &o)

	var h = 30
	var p *int = &h
	var u unsafe.Pointer = unsafe.Pointer(p)
	q := *(*int)(u)

	fmt.Println("Pointer to int:", p)
	fmt.Println("Pointer to unsafe.Pointer:", u)
	fmt.Println("Pointer to int:", q)

	var z *int = new(int)
	addr := uintptr(unsafe.Pointer(z))

	fmt.Println(addr)

	type S struct {
		A int8  // 1byte
		B int32 // 4byte
	}

	var instance *S = &S{A: 1, B: 2}
	var addr1 uintptr = uintptr(unsafe.Pointer(instance))

	fmt.Println(addr1)

	var copy *S = (*S)(unsafe.Pointer(addr1))
	fmt.Printf("Value of copy: %+v\n", copy)

	base := uintptr(unsafe.Pointer(instance))
	pb := (*int32)(unsafe.Pointer(base + 1))
	fmt.Printf("Value of pb: %d\n", *pb)
}
