package iracing

import "fmt"

type IracingVariable struct {
	Name  string
	Value float32
}

type Output struct {
	out chan *IracingVariable
}

// for now this will receive variables and output them somewhere
func (o *Output) OutputChannel() chan *IracingVariable {
	if o.out == nil {
		o.out = make(chan *IracingVariable, 5)
		go func() {
			for {
				iv := <-o.out
				fmt.Printf("%s, %f \r\n", iv.Name, iv.Value)
			}
		}()
	}
	return o.out
}
