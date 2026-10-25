package hello

type Greeter struct{}

func (Greeter) Greet() string {
	return "hi"
}

