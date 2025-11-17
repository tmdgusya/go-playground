# 포인터

`Go` 나 `C` 를 사용하다보면 `포인터` 라는 개념을 마주하게 된다. 어렵게 느껴질 수 있지만 이후에 `zero copy` 등의 개념을 이해하기 위해서는 필수적으로 이해해야 하는 요소 중에 하나이다. 오늘은 `Go` 를 통해 `포인터` 를 공부해보고 여러 최적화 기법을 공부해보자.

## 세가지 포인터

Go 에서 "메모리 주소" 를 다루는 축은 총 세가지이다.

1. 일반 포인터: **`*T`**
3. 정수 주소값: **`uintptr`**
2. 범용 포인터: **`unsafe.Pointer`**

각각 성격이 다르므로 오늘은 이 세가지에 대해 공부해보고 어느 상황에서 쓸 수 있는지를 학습해보도록 하자.


## `*T` - 타입이 있는 안전한 포인터

```go
var k int = 10
var y *int = &k

fmt.Println("Value of k:", k) // Value of k: 10
fmt.Println("Value of y:", y) // Value of y: 0x1400019c008
```

여기서 `y` 는 **"int 값이 있는 메모리 위치"** 를 가리킨다. 그래서 실제로 출력해보면 위와 같은 결과를 확인해볼 수 있다. 실습할 겸 `*int` 를 다른 포인터 변수에 한번 할당해보자.

```go
var o *int = (*int)(y)

fmt.Printf("Value of o: %d\n", *o) // 20 출력됨
```

여기서 만약 다시 `k` 를 20 으로 바꾸면 어떻게 될까?

```go
k = 20

fmt.Println("Value of k:", k) // Value of k: 20
fmt.Println("Value of y:", y) // Value of y: 0x1400019c008
fmt.Println("Value of o:", *o) // Value of o: 20
fmt.Println("has same addr? ", &k == o) // has same addr?  true
```

o 라는 변수는 포인터 변수로 k 의 pointer 주소를 저장하게 된다. 따라서 k 의 값이 변경되면 o 의 값도 변경된다. 여기까지는 포인터라는 개념에 조금 익숙하다면 쉽게 이해가 갈텐데 `타입` 이 있다는건 무슨 뜻일까? 타입이 있다는 뜻은 `*T` 를 이용하게 되면 컴파일러와 GC 는 `*int` 를 "진짜 포인터" 로 인식해서 `alias 분석`, `escape analysis`, `GC root / reference` 등에 대한 최적화를 할 수 있게 된다. 그렇다면 `float32` 타입에는 할당이 안되는걸까?

```go
var k int = 10
var y *int = &k
var o *int = (*float32)(y) // float32 타입에 할당 불가능 Error: cannot convert y (variable of type *int) to type *float32
```

위 에러와 같이 전혀 다른 포인터 타입으로는 할당이 불가능한 것을 확인할 수 있다.

## `uintptr` - 정수 주소값

```go
var z *int = new(int)
addr := uintptr(unsafe.Pointer(z))

fmt.Println(addr) // 1374390644192
```

`uintptr` 는 정수형 타입으로 포인터의 주소값을 저장할 수 있다. 이는 포인터의 주소값을 정수형으로 변환하거나 정수형으로부터 포인터를 생성할 수 있게 해준다. 위의 예시를 보면 1374390644192 라는 *int 포인터의 주소값을 uintptr 타입으로 변환한 것이다. 하지만 웃기게도 이 타입은 포인터가 아니다. 정수 타입이기 때문에 이런 덧셈 연산도 가능하다 `fmt.Println(addr + 1)`. 즉, 컴파일러는 이를 포인터 주소를 담고 있지만 포인터로 인식하지 않는 값이다 라고 생각하면 도움이 된다.

## `unsafe.Pointer` - 타입 없는 포인터

`unsafe.Pointer` 는 문서를 읽어보면 `Arbitrary type` 에 대한 포인터라고 나와 있다. 그리고 4 가지 특별한 연산이 가능하다고 하는데 그 연산은 아래와 같다.

1. 모든 유형의 포인터 값은 포인터로 변환될 수 있습니다.
2. 포인터는 모든 유형의 포인터 값으로 변환 될 수 있습니다.
3. `unitptr` 은 포인터로 변환될 수 있습니다.
4. 포인터는 `uintptr` 로 변환될 수 있습니다.

첫번째 항목부터 보도록 하자. 첫번째 규칙에 따르면, 우리가 지금까지 써왔던 타입을 가진 포인터 변수 또한 `unsafe.Pointer` 로 변환될 수 있어야 할 것이다.

```go
var h = 30
var p *int = &h
var u unsafe.Pointer = unsafe.Pointer(p)
q := *(*int)(u)

fmt.Println("Pointer to int:", p) // Pointer to int: 0x14000106028
fmt.Println("Pointer to unsafe.Pointer:", u) // Pointer to unsafe.Pointer: 0x14000106028
fmt.Println("Pointer to int:", q) // Pointer to int: 30
```

위와 같이 `*int -> unsafe.Pointer -> *int` 로 변환하여 p 의 포인터가 가르키는 메모리 값에 저장된 정수를 복사하여 저장하는데 까지 성공했음을 알 수 있다. 즉, 모든 유형의 포인터 값은 포인터로 변환될 수 있다는 것을 확인했다. 이제 마지막으로 `unitptr` 의 저장된 주소값을 통해 `pointer` 변수를 만들 수 있는지 확인해보자.

```go
type S struct {
	A int32
	B int32
}

var instance *S = &S{A: 1, B: 2}
var addr uintptr = uintptr(unsafe.Pointer(instance))

fmt.Println(addr) // 1374390169024

var copy *S = (*S)(unsafe.Pointer(addr1))
fmt.Printf("Value of copy: %+v\n", copy) // Value of copy: &{A:1 B:2}
```

위의 예시를 통해 `uintptr` 에서 `unsafe.Pointer` 로 변환하여 `pointer` 변수를 만들 수 있다는 것을 확인했다. 그런데 보다보니 재미있는 생각이 떠오른다. 내가 몇일 전에 적은 `offset` 을 읽은 사람들은 알겠지만 결국이 구조체도 `byte` 배열로 쓰여지게 된다. 즉, `base` 에서 특정 `offset` 만큼 이동하면 어떠한 값을 읽어볼수도 있다는 소리다. 한번 이걸 이용해보자.

### Struct 필드 Offset 접근

unsafe 에는 `Offsetof` 함수가 존재하는데 이를 이용하면 구조체 내부에서 **특정 필드까지의 거리(Offset)** 을 알 수 있다. 그리고 우리는 아까 `unitptr` 이 덧셈 연산이 가능하다는 것을 알았다. 그렇다면 우리는 `base + offset` 을 이용하여 특정 필드에 직접적으로 접근할 수 있다.

```go
base := uintptr(unsafe.Pointer(instance))

offsetB := unsafe.Offsetof(instance.B)
pb := (*int32)(unsafe.Pointer(base + offsetB))
fmt.Printf("Value of pb: %d\n", *pb) // Value of pb: 2
```

이렇게 직접 접근하는 방법을 알게 되면 자연스럽게 Go 의 `padding & alignment` 의 개념을 이해하거나 테스트 해보게 된다. 만약, 이 개념을 모른다면 왜 아래 코드가 정상적으로 동작하지 않는지 한번 공부해봐도 좋다.

```go
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
pb := (*int32)(unsafe.Pointer(base + 1)) // base 에서 1 바이트 만큼을 더 이동
fmt.Printf("Value of pb: %d\n", *pb) // 33599038 ?? 이상한 값이 출력됨
```

위의 예시를 보면 당연히 A 가 1 byte 이므로 `+1` 만해줘도 B 의 값이 나와야 할거 같지만 이상하게도 `33599038` 와 같은 이상한 값들이 출력된다. 그래서 아래 예시와 같이 4바이트만큼 이동시켜야 정상적인 값이 출력되는데 이유는 왜일까? 한번 모른다면 공부해보길 바란다.

```go
// 정답은 4 바이트만큼 이동시켜야 함
base := uintptr(unsafe.Pointer(instance))
pb := (*int32)(unsafe.Pointer(base + 4))
fmt.Printf("Value of pb: %d\n", *pb) // Value of pb: 2 정상 출력!
```

### []byte <-> string 변환 (복사 없이)

요러한 방식은 `zero copy` 방식으로
