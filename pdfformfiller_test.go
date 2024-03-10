package main

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPDFFormFiller(t *testing.T) {
	templateData, err := ioutil.ReadFile("testdata/fw8ben.pdf")
	fmt.Println(err)
	filler, err := NewPDFFormFiller(templateData)
	if err != nil {
		fmt.Println(err)
		return
	}

	signatureImage, err := ioutil.ReadFile("testdata/signature.png")

	//filler.FillFormFieldsWithItsIdName()

	//filler.SetTextFieldById(302, "中文", true)
	//filler.SetCheckboxFieldById(312, "1", true)
	//filler.AddImageOverObjectById(323, signatureImage)

	filler.SetTextFieldByName("f_2[0]", "test 01", false)
	filler.SetCheckboxFieldByName("c1_02[0]", "test 02", false)
	filler.AddImageOverObjectByName("Date[0]", signatureImage)

	result, err := filler.WriteToBytes()
	assert.Nil(t, err)
	assert.True(t, len(result) > 0)

	ioutil.WriteFile("testdata/fw8ben-labeled.pdf", result, 0644)
}
