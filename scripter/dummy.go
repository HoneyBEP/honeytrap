package scripter

func MustDummy(options ...func(Scripter) error) Scripter {
	l, _ := Dummy()
	return l
}

func Dummy(options ...func(Scripter) error) (Scripter, error) {
	return &dummyScripter{}, nil
}

type dummyScripter struct {

}

func (*dummyScripter) SetVariable(string, string) error {
	return nil
}