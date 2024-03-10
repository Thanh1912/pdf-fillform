package main

import (
	"fmt"
	"io/ioutil"
)

func main() {
	templateData, err := ioutil.ReadFile("testdata/fw8ben.pdf")
	fmt.Println(err)
	filler, err := NewPDFFormFiller(templateData)
	if err != nil {
		fmt.Println(err)
		return
	}
	//filler.SetTextFieldByName("f_2[0]", "test 01", false)
	//filler.SetCheckboxFieldByName("c1_02[0]", "1", false)

	// Create the form values.
	form := Form{
		"f_2[0]":   "Hello",
		"c1_02[0]": true,
	}
	filler.Fill(form, true)
	result, err := filler.WriteToBytes()

	ioutil.WriteFile("testdata/fw8ben-labeled.pdf", result, 0644)
}

//edit file pdf: https://www.sejda.com/en/pdf-forms
