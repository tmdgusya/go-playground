## 들어가며

Go를 쓰다 보면 **“인터페이스가 뭔가 다른 언어와 다르게 신기하게 쓰이는 구나”** 하는 순간이 종종 있다. 특히 제네릭이 없던 시절의 Go 코드를 보면 아래처럼 interface{}를 마치 만능 컨테이너처럼 쓰는 코드가 흔했다.

```go
var values []interface{}
values = append(values, 10)
values = append(values, "hello")
values = append(values, []int{1, 2, 3})
```


그리고 나중에 이런 식으로 타입에 따라 분기한다.

```go
func Print(v interface{}) {
	switch x := v.(type) {
	case int:
		fmt.Println("int:", x)
	case string:
		fmt.Println("string:", x)
	case []int:
		fmt.Println("slice:", x)
	default:
		fmt.Printf("unknown type: %T\n", x)
	}
}
```

정적 타입 언어에서 이렇게 “무한정 아무 타입이나 넣고 꺼내는" 구조가 되는 건 꽤 놀랍다. 이게 어떻게 가능할까? 이 질문의 진짜 핵심은 Go 인터페이스의 내부 구조에 있다.

## eface / iface

Go 언어는 인터페이스를 두 가지 형태로 나누어 구현하고 있다.

- **eface**: 빈 인터페이스 (interface{})
- **iface**: 메서드가 있는 인터페이스

이 두 구조를 알면 Go 인터페이스를 이해하는데 큰 도움이 된다.

### eface — 빈 인터페이스 (interface{})

Go 런타임에 정의된 eface 구조체는 정말 단순하다.

```go
type eface struct {
    typ  *_type         // dynamic type metadata
    data unsafe.Pointer // pointer to actual data
}
```

interface 값은 정적 타입이 interface이지만, 내부에는 **“구체 타입 + 값”**이 살아 있다.

예를 들어:

```go
var x interface{} = 10
```

런타임 구조는 다음과 같다:

```go
x = (type=*int, data=&10)
```

이 사실 하나만 이해해도 왜 여러 타입을 한 slice에 넣을 수 있는지 왜 type assertion이 가능한지 왜 JSON 언마샬 결과가 map[string]interface{} 인지 왜 reflect가 interface 기반으로 동작하는지 전부 설명된다.

### iface — 메서드가 있는 인터페이스

예를 들어:

```go
type Reader interface {
    Read([]byte) (int, error)
}
```

이런 인터페이스는 메서드 테이블(itab)을 포함하는 iface 구조를 사용한다.

```go
type iface struct {
    tab  *itab
    data unsafe.Pointer
}
```

itab 안에는 아래와 같은 정보들이 존재한다.
- **인터페이스 정보**
- **구체 타입 정보**
- **인터페이스가 요구하는 메서드들의 함수 포인터(jump table)**

```go
var r io.Reader = bytes.NewBuffer([]byte("hi"))
r.Read(buf)
```

그래서 위와 같은 코드를 실행시켜보면 런타임에서 실제로는

```go
itab.funcs[0](data, buf)
```

이런 식으로 메서드가 호출된다.

### nil interface trap

이제 구조를 봤으니 이 trap도 제대로 이해할 수 있다.

```go
var x interface{} = (*User)(nil)

fmt.Println(x == nil) // false
```

왜 false일까? 인터페이스가 “진짜 nil”이 되려면 `typ == nil && data == nil` 이 되어야 한다. 그러나 위 코드는 `x = (typ = *User, data = nil)` 이다. 즉, type 포인터는 살아있고 data는 nil 이기 때문에 interface 전체는 nil이 아니다. 

### 실제 메모리 구조를 직접 확인

```go
package main

import (
    "fmt"
    "unsafe"
)

type eface struct {
    typ  uintptr
    data uintptr
}

func dump(label string, v interface{}) {
    p := (*eface)(unsafe.Pointer(&v))

    fmt.Printf("[%s]\n", label)
    fmt.Printf("  interface address : %p\n", &v)
    fmt.Printf("  type pointer      : 0x%x\n", p.typ)
    fmt.Printf("  data pointer      : 0x%x\n", p.data)

    // data를 실제 타입으로 역참조할 수 있는 경우 출력
    if p.data != 0 {
        fmt.Printf("  data as int?      : %d\n", *(*int)(unsafe.Pointer(p.data)))
    }
    fmt.Println()
}

func main() {
    dump("int", 10)
    dump("string", "hello")
    dump("slice", []int{1, 2, 3})

    var u *int = nil
    dump("nil pointer in interface", u)

    var x interface{}
    dump("true nil interface", x)
}
```

#### 실행 결과

```sh
[int]
  interface address : 0x140000a4010
  type pointer      : 0x102a924c0
  data pointer      : 0x102a83538
  data as int?      : 10

[string]
  interface address : 0x140000a4030
  type pointer      : 0x102a92280
  data pointer      : 0x102aad068
  data as int?      : 4339381145

[slice]
  interface address : 0x140000a4050
  type pointer      : 0x102a912e0
  data pointer      : 0x140000aa000
  data as int?      : 1374390214680

[nil pointer in interface]
  interface address : 0x140000a4070
  type pointer      : 0x102a8f260
  data pointer      : 0x0

[true nil interface]
  interface address : 0x140000a4090
  type pointer      : 0x0
  data pointer      : 0x0
```

nil pointer in interface 의 경우 type pointer 값이 존재하므로 interface 자체는 nil이 아니다. true nil interface 의 경우 type/data 둘 다 nil일 때만 진짜 nil 이다.

## 자주보이는 interface pattern들

### 여러 타입을 한 컨테이너에 담기 ([]interface{})

`eface` 구조 덕분에 어떤 타입이든 담을 수 있다.

```go
list := []interface{}{1, "hello", []int{1,2,3}}
```

dynamic type이 살아있으니 type switch로 잘 처리된다.

### type switch로 다형성 처리

```go
switch v := x.(type) {
case int, string:
    ...
}
```

interface 내부 type pointer 비교만 하므로 성능도 꽤 빠른 편이다.

###  JSON / YAML 동적 구조 (map[string]interface{})

Go의 정적 타입 구조상, JSON 같은 문서 기반 구조를 표현하려면 value를 interface로 받을 수밖에 없다. 그리고 자연스럽게 type switch로 핸들링한다.

### error 인터페이스 기반 다형성

```go
type error interface {
    Error() string
}
```

iface 구조 덕분에 다양한 에러 타입을 단일 인터페이스로 다루고, wrapping/unwrapping도 dynamic type 정보 기반으로 자연스럽게 구현된다.

## 그렇다면 문제는 없을까?

얼핏 보기에 정적 타입 내부에서 꽤나 자유롭게 `Interface` 를 이용할 수 있어 보인다. 하지만 `escape analysis` 에 의한 `interface` 타입의 `heap allocation` 이 발생할 수 있다. 아마 `rust` 를 공부했던 사람들은 익숙할 수 있는데 이 `heap allocation` 을 이해하기 위해 아래 예시를 함께 보자.

```go
func f() interface{} {
    x := 10
    return x
}
```

위 함수를 보면 x 에 정수(int) 값 10을 할당하고 return type 으로 `interface{}` 타입을 반환한다. 우리의 상식으로는 `int` 는 정수값으로 원래 스택 변수이기 때문에 `heap allocation` 이 발생하지 않을 것 같지만, 실제로는 `heap allocation` 이 발생한다. 이는 `escape analysis` 에 의한 `interface` 타입의 `heap allocation` 이 발생하기 때문이다.

이유는 무엇일까? 예를 들어 `a = f()` 라는 부분이 외부에 있다고 해보자. 그런데 x 의 값이 사라지면 interface 의 data pointer 가 dangling pointer 가 되어버린다. 따라서 컴파일러는 이를 최대한 안전하게 처리하기 위해 `heap` 에 올려버린다.

> x escapes to heap because it’s stored in interface

실제 예시와 `go build` 에서 flag 를 통해 확인해보자

```go
package main

func aa() interface{} {
	x := 100
	return x
}

func main() {
	a := aa()
	_ = a
}
```

```sh
❯ go build -gcflags="-m" interface_prac/main.go
# command-line-arguments
interface_prac/main.go:3:6: can inline aa
interface_prac/main.go:8:6: can inline main
interface_prac/main.go:9:13: inlining call to aa
interface_prac/main.go:5:9: 100 escapes to heap[^1]
interface_prac/main.go:9:13: 100 does not escape
```

위와 같이 100 이 heap 으로 escape 되는 것을 확인할 수 있다. 만약 `int` 타입으로 리턴됬다면 어떨까?

```go
package main

func aa() int {
	x := 100
	return x
}

func main() {
	a := aa()
	_ = a
}
```

```sh
❯ go build -gcflags="-m" interface_prac/main.go
# command-line-arguments
interface_prac/main.go:3:6: can inline aa
interface_prac/main.go:8:6: can inline main
interface_prac/main.go:9:9: inlining call to aa[^2]
```

`int` 를 리턴하는 경우에는 위 예시처럼 `escape` 가 일어나지 않음을 확인할 수 있다.

---
[^1]: “100 escapes to heap”은 x 변수가 escape했다는 뜻이 아니다. literal 100 자체가 interface wrapping 과정에서 힙으로 복사된 것이다. 함수의 반환 타입이 interface이기 때문에 literal은 임시 메모리에 둘 수 없어서 힙으로 올라가는 것이다.

[^2]: 여기서 “inlining call to aa”는 escape analysis와 무관하다. 단순히 컴파일러가 aa() 함수를 main 함수 안으로 인라인 최적화 한 것뿐이다. escape 여부는 '100 escapes to heap' 같은 별도의 메시지에서 판별된다.
