package main

import (
	"fmt"
	"gonum.org/v1/gonum/mat"
	"log"
	"math"
	"testing"
)

func TestNewVecDense(t *testing.T) {
	//u := mat.NewVecDense(3, []float64{1, 2, 3})
	//v := mat.NewVecDense(3, []float64{4, 5, 6})
	//logger.Debugf("%v %v", u, v)

	//vectorA := []float64{11.0, 5.2, -1.3}
	//vectorB := []float64{-7.2, 4.2, 5.1}
	//dotProduct := floats.Dot(vectorA, vectorB)
	//fmt.Printf("The dot product of A and B is: %0.2f\n", dotProduct)
	//floats.Scale(1.5, vectorA)
	//fmt.Printf("Scaling A by 1.5 gives: %v\n", vectorA)
	//normB := floats.Norm(vectorB, 2)
	//fmt.Printf("The norm/length of B is: %0.2f\n", normB)

	//vectorA := mat.NewVecDense(3, []float64{11.0, 5.2, -1.3})
	//vectorB := mat.NewVecDense(3, []float64{-7.2, 4.2, 5.1})
	//dotProduct := mat.Dot(vectorA, vectorB)
	//fmt.Printf("The dot product of A and B is: %0.2f\n", dotProduct)
	//vectorA.ScaleVec(1.5, vectorA)
	//fmt.Printf("Scaling A by 1.5 gives: %v\n", vectorA)
	//normB := blas64.Nrm2(vectorB.RawVector())
	//fmt.Printf("The norm/length of B is: %0.2f\n", normB)
	//
	//components := []float64{
	//	1.2, -5.7,
	//	-2.4, 7.3,
	//}
	//a := mat.NewDense(2, 2, components)
	//fmt.Printf("%v\n", a)
	//fa := mat.Formatted(a, mat.Prefix("      "))
	//fmt.Printf("mat = %v\n\n", fa)

	a := mat.NewDense(3, 3, []float64{
		1, 2, 3,
		0, 4, 5,
		0, 0, 6,
	})
	b := mat.NewDense(3, 3, []float64{
		8, 9, 10,
		1, 4, 2,
		9, 0, 2,
	})
	fa := mat.Formatted(a, mat.Prefix(""))
	fmt.Printf("a\n%v\n\n", fa)
	fb := mat.Formatted(b, mat.Prefix(""))
	fmt.Printf("b\n%v\n\n", fb)
	c := mat.NewDense(3, 2, []float64{
		3, 2,
		1, 4,
		0, 8,
	})
	d := mat.NewDense(3, 3, nil)
	d.Add(a, b)
	fd := mat.Formatted(d, mat.Prefix(""))
	fmt.Printf("d = a + b\n%0.4v\n\n", fd)

	f := mat.NewDense(3, 2, nil)
	f.Mul(a, c)
	ff := mat.Formatted(f, mat.Prefix(""))
	fmt.Printf("f = a c\n%0.4v\n\n", ff)
	e := mat.NewDense(3, 3, []float64{
		8, 9, 10,
		1, 4, 2,
		9, 0, 2,
	})
	g := mat.NewDense(3, 1, []float64{
		3,
		1,
		0,
	})
	h := mat.NewDense(3, 1, nil)
	h.Mul(e, g)
	hf := mat.Formatted(h, mat.Prefix(""))
	fmt.Printf("h = gxh\n%0.4v\n\n", hf)

	k := mat.NewDense(3, 3, nil)
	k.Apply(func(_, _ int, v float64) float64{
		return math.Sqrt(v)
	}, e)
	fmt.Printf("k = \n%0.4v\n\n", mat.Formatted(k, mat.Prefix("")))
}


func TestEigenValue(t *testing.T) {
	a := mat.NewDense(3, 3, []float64{1, 2, 3, 0, 4, 5, 0, 0, 6})
	fmt.Printf("a = \n%v\n\n", mat.Formatted(a, mat.Prefix("")))
	fmt.Printf("a^T = \n%v\n\n", mat.Formatted(a.T(), mat.Prefix("")))

	deta := mat.Det(a)
	fmt.Printf("det(a) = %.2f\n\n", deta)

	aInverse := mat.NewDense(3, 3, nil)
	if err := aInverse.Inverse(a); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("a^-1 = \n%v\n\n", mat.Formatted(aInverse, mat.Prefix("")))
}
