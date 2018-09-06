package matrixRoute

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	pageTop = `<!DOCTYPE HTML><html><head>
<style>.error{color:#FF0000;}.result{color:#0000FF}</style></head><title>Statistics</title>
<body><h3>Matrix multiplication</h3>
<p>Computes matrix multiplication b/w 2 matrices.</p>`
	form = `<form action="/homeC" method="POST">
<label for="matASize">Size of matrix A  (comma or space-separated) </label><br />
<input type="text" name="matASize" size="30"><br />
<label for="matBSize">Size of matrix B  (comma or space-separated) </label><br />
<input type="text" name="matBSize" size="30"><br />
<input type="submit" value="Calculate">
</form>`
	pageBottom = `</body></html>`
	anError    = `<p class="error">%s</p>`
)

func formatResult(result, timeTaken float64) string {
	return fmt.Sprintf(`<h4 class="result">The result is %f, time taken is %f</h4>`, result, timeTaken)
}

func createMat(matrixSize [2]int) [][]float64 {
	noOfRows, noOfCols := matrixSize[0], matrixSize[1]
	var mat = make([][]float64, noOfRows)
	for rowIdx := range mat {
		mat[rowIdx] = make([]float64, noOfCols)
		for colIdx := range mat[rowIdx] {
			mat[rowIdx][colIdx] = rand.Float64() * 1e3
		}
	}
	// fmt.Println(mat)
	return mat
}

func matrixMultiply(mat1 [][]float64, mat2 [][]float64, sumCh chan float64) float64 {
	rowsOfMat1 := len(mat1)
	colsOfMat2 := len(mat2[0])

	for mat1RowIdx := 0; mat1RowIdx < rowsOfMat1; mat1RowIdx++ {
		for mat2ColIdx := 0; mat2ColIdx < colsOfMat2; mat2ColIdx++ {
			go dotProduct(mat1, mat2, mat1RowIdx, mat2ColIdx, sumCh)
		}
	}

	var sum float64

	for i := 0; i < rowsOfMat1; i++ {
		for j := 0; j < colsOfMat2; j++ {
			sum += <-sumCh
		}
	}

	return sum
}

func dotProduct(mat1, mat2 [][]float64, mat1RowIdx, mat2ColIdx int, sumCh chan float64) {
	var result float64

	for k := 0; k < 100; k++ {
		for mat1ColIdx := range mat1[mat1RowIdx] {
			result += mat1[mat1RowIdx][mat1ColIdx] * mat2[mat1ColIdx][mat2ColIdx]
		}
	}
	// fmt.Println(result)
	sumCh <- result
}

func createMatAndMultiply(matAsize, matBsize [2]int) float64 {
	mat1 := createMat(matAsize)
	mat2 := createMat(matBsize)
	sumCh := make(chan float64, matAsize[0]*matBsize[1])
	sum := matrixMultiply(mat1, mat2, sumCh)
	return sum
}

func canMultiply(matAsize, matBsize [2]int) (bool, string) {
	if matAsize[1] == matBsize[0] {
		return true, ""
	}
	return false, fmt.Sprintf("matrix with size %d * %d cannot be multiplied with matrix of size %d * %d", matAsize[0], matAsize[1], matBsize[0], matBsize[1])
}

// MatrixHandler returns the home page with the requested computation
func MatrixHandler(writer http.ResponseWriter, request *http.Request) {
	err := request.ParseForm() // Must be called before writing response
	fmt.Fprint(writer, pageTop, form)
	if err != nil {
		fmt.Fprintf(writer, anError, err)
	} else {
		if len(request.Form) == 0 {
			fmt.Println("page requested for first time")
		} else {
			if matrixSizes, errorMessage, ok := processRequest(request); ok {
				if isTrue, errorMessage := canMultiply(matrixSizes[0], matrixSizes[1]); isTrue {
					result, timeTaken := timeit(createMatAndMultiply)(matrixSizes[0], matrixSizes[1])
					fmt.Fprint(writer, formatResult(result, timeTaken))
				} else {
					fmt.Fprintf(writer, anError, errorMessage)
				}
			} else {
				fmt.Fprintf(writer, anError, errorMessage)
			}
		}
	}
	fmt.Fprint(writer, pageBottom)
}

func processRequest(request *http.Request) ([2][2]int, string, bool) {
	var matSizes [2][2]int
	for matID, matName := range []string{"matASize", "matBSize"} {
		slice, found := request.Form[matName]
		userInputString := slice[0]
		if found && len(userInputString) > 0 {
			var sizeValues = strings.Fields(strings.Replace(userInputString, ",", " ", -1))
			if len(sizeValues) < 2 {
				return matSizes, "2 numbers needed, only 1 received", false
			}
			for idx, stringValue := range sizeValues {
				if intValue, err := strconv.Atoi(stringValue); err != nil {
					return matSizes, stringValue + " is an invalid input", false
				} else {
					matSizes[matID][idx] = intValue
				}
			}
		} else {
			return matSizes, "2 numbers needed, 0 received", false
		}
	}
	return matSizes, "", true
}

func timeit(function func([2]int, [2]int) float64) func([2]int, [2]int) (float64, float64) {
	return func(arg1, arg2 [2]int) (float64, float64) {
		start := time.Now()
		result := function(arg1, arg2)
		timeTaken := time.Now().Sub(start).Seconds()
		return result, timeTaken
	}
}
